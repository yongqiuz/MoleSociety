package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type Relayer struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
	Nonce      int64
	mu         sync.Mutex
}

type App struct {
	ctx            context.Context
	rdb            *redis.Client
	client         *ethclient.Client
	relayers       []*Relayer
	relayerCounter uint64
	chainID        *big.Int
	social         *SocialService
	auth           *AuthService
}

type CommonResponse struct {
	OK      bool   `json:"ok"`
	Status  string `json:"status,omitempty"`
	TxHash  string `json:"txHash,omitempty"`
	Error   string `json:"error,omitempty"`
	Role    string `json:"role,omitempty"`
	Address string `json:"address,omitempty"`
}

type relayMintRequest struct {
	Dest     string `json:"dest"`
	CodeHash string `json:"codeHash"`
}

type saveCodeRequest struct {
	CodeHash   string `json:"codeHash"`
	Address    string `json:"address"`
	ReferrerID string `json:"referrerId"`
}

type rewardRequest struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"`
}

func main() {
	godotenv.Load()

	ctx := context.Background()
	app := &App{ctx: ctx}
	app.rdb = initRedis(ctx)
	app.client, app.chainID = initRPC()
	app.loadRelayers()
	app.social = NewSocialService(ctx, app.rdb)
	app.auth = NewAuthService(ctx, app.rdb, app.social)

	router := mux.NewRouter()
	app.registerLegacyRoutes(router)
	app.registerAuthRoutes(router)
	app.registerSocialRoutes(router)
	router.HandleFunc("/healthz", app.healthHandler).Methods(http.MethodGet, http.MethodOptions)

	addr := envOrDefault("BACKEND_ADDR", "0.0.0.0:8080")
	log.Printf("MoleSociety backend listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, requestLogger(cors(router))))
}

func initRedis(ctx context.Context) *redis.Client {
	addr := envOrDefault("REDIS_ADDR", "127.0.0.1:6379")
	client := redis.NewClient(&redis.Options{Addr: addr})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		log.Printf("redis unavailable, using in-memory state: %v", err)
		return nil
	}
	return client
}

func initRPC() (*ethclient.Client, *big.Int) {
	rpcURL := strings.TrimSpace(os.Getenv("RPC_URL"))
	if rpcURL == "" {
		return nil, nil
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Printf("rpc unavailable, relay mint will be simulated: %v", err)
		return nil, nil
	}

	chainIDStr := envOrDefault("CHAIN_ID", "0")
	parsed, err := strconv.ParseInt(chainIDStr, 10, 64)
	if err != nil {
		log.Printf("invalid CHAIN_ID %q, using 0", chainIDStr)
		return client, big.NewInt(0)
	}

	return client, big.NewInt(parsed)
}

func (a *App) loadRelayers() {
	if a.client == nil {
		return
	}

	count, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("RELAYER_COUNT")))
	for i := 0; i < count; i++ {
		key := strings.TrimSpace(os.Getenv(fmt.Sprintf("PRIVATE_KEY_%d", i)))
		if key == "" {
			continue
		}

		priv, err := crypto.HexToECDSA(strings.TrimPrefix(key, "0x"))
		if err != nil {
			log.Printf("skip relayer %d: %v", i, err)
			continue
		}

		addr := crypto.PubkeyToAddress(priv.PublicKey)
		nonce, err := a.client.PendingNonceAt(a.ctx, addr)
		if err != nil {
			log.Printf("skip relayer %d nonce: %v", i, err)
			continue
		}

		a.relayers = append(a.relayers, &Relayer{
			PrivateKey: priv,
			Address:    addr,
			Nonce:      int64(nonce),
		})
	}
}

func (a *App) registerLegacyRoutes(r *mux.Router) {
	r.HandleFunc("/secret/get-binding", a.getBindingHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/secret/verify", a.verifyHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/relay/mint", a.mintHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/relay/save-code", a.saveCodeHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/relay/reward", a.rewardHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/relay/stats", a.relayStatsHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/admin/check-access", a.checkAdminAccessHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/analytics/distribution", a.distributionHandler).Methods(http.MethodGet, http.MethodOptions)
}

func (a *App) registerSocialRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/social/bootstrap", a.bootstrapHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/instances", a.instancesHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/users", a.listUsersHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/users", a.createUserHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/social/users/{id}", a.getUserHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/users/{id}", a.updateUserHandler).Methods(http.MethodPatch, http.MethodOptions)
	r.HandleFunc("/api/v1/social/feed", a.feedHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/posts", a.createPostHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/social/posts/{id}", a.getPostHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/posts/{id}/thread", a.getPostThreadHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/posts/{id}/replies", a.listPostRepliesHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/media", a.listMediaHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/media", a.createMediaHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/social/conversations", a.listConversationsHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/conversations", a.createConversationHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/social/conversations/{id}", a.getConversationHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/social/conversations/{id}/messages", a.addMessageHandler).Methods(http.MethodPost, http.MethodOptions)
}

func (a *App) healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"service":     "molesociety-social-backend",
		"redis":       a.rdb != nil,
		"relayReady":  a.client != nil && len(a.relayers) > 0,
		"socialStats": a.social.Stats(),
	})
}

func (a *App) mintHandler(w http.ResponseWriter, r *http.Request) {
	var req relayMintRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonResponse{OK: false, Error: "invalid JSON"})
		return
	}

	if a.rdb != nil && req.CodeHash != "" {
		isPublisher, _ := a.rdb.SIsMember(a.ctx, "vault:roles:publishers_codes", req.CodeHash).Result()
		if isPublisher {
			writeJSON(w, http.StatusOK, CommonResponse{OK: true, Status: "PUBLISHER_WELCOME", Role: "publisher"})
			return
		}

		removed, _ := a.rdb.SRem(a.ctx, "vault:codes:valid", req.CodeHash).Result()
		if removed == 0 {
			writeJSON(w, http.StatusForbidden, CommonResponse{OK: false, Error: "code used or invalid"})
			return
		}
	}

	txHash, err := a.executeMint(req.Dest)
	if err != nil {
		if a.rdb != nil && req.CodeHash != "" {
			a.rdb.SAdd(a.ctx, "vault:codes:valid", req.CodeHash)
		}
		writeJSON(w, http.StatusInternalServerError, CommonResponse{OK: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, CommonResponse{OK: true, TxHash: txHash, Role: "reader"})
}

func (a *App) verifyHandler(w http.ResponseWriter, r *http.Request) {
	if a.rdb == nil {
		writeJSON(w, http.StatusOK, CommonResponse{OK: true, Role: "reader"})
		return
	}

	addr := strings.ToLower(r.URL.Query().Get("address"))
	code := r.URL.Query().Get("codeHash")

	isPublisherCode, _ := a.rdb.SIsMember(a.ctx, "vault:roles:publishers_codes", code).Result()
	if isPublisherCode {
		isPublisherAddr, _ := a.rdb.SIsMember(a.ctx, "vault:roles:publishers", addr).Result()
		if isPublisherAddr {
			writeJSON(w, http.StatusOK, CommonResponse{OK: true, Role: "publisher"})
			return
		}
	}

	isReader, _ := a.rdb.SIsMember(a.ctx, "vault:codes:valid", code).Result()
	if isReader {
		writeJSON(w, http.StatusOK, CommonResponse{OK: true, Role: "reader"})
		return
	}

	writeJSON(w, http.StatusForbidden, CommonResponse{OK: false, Error: "unauthorized"})
}

func (a *App) getBindingHandler(w http.ResponseWriter, r *http.Request) {
	if a.rdb == nil {
		writeJSON(w, http.StatusOK, CommonResponse{OK: true})
		return
	}

	code := r.URL.Query().Get("codeHash")
	data, _ := a.rdb.HGetAll(a.ctx, "vault:bind:"+code).Result()
	writeJSON(w, http.StatusOK, CommonResponse{OK: true, Address: data["address"]})
}

func (a *App) saveCodeHandler(w http.ResponseWriter, r *http.Request) {
	var req saveCodeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	if a.rdb != nil && req.CodeHash != "" {
		a.rdb.HSet(a.ctx, "vault:bind:"+req.CodeHash, map[string]any{
			"address":    req.Address,
			"referrerId": req.ReferrerID,
			"savedAt":    time.Now().UTC().Format(time.RFC3339),
		})
		if req.ReferrerID != "" {
			a.rdb.ZIncrBy(a.ctx, "vault:referrers", 1, req.ReferrerID)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "saved"})
}

func (a *App) rewardHandler(w http.ResponseWriter, r *http.Request) {
	var req rewardRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	if a.rdb != nil && req.Address != "" {
		a.rdb.HIncrBy(a.ctx, "vault:rewards:"+req.Address, "total", req.Amount)
		a.rdb.HSet(a.ctx, "vault:rewards:"+req.Address, "updatedAt", time.Now().UTC().Format(time.RFC3339))
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "queued", "address": req.Address, "amount": req.Amount})
}

func (a *App) relayStatsHandler(w http.ResponseWriter, _ *http.Request) {
	type relayRank struct {
		Referrer string  `json:"referrer"`
		Score    float64 `json:"score"`
	}

	referrers := []relayRank{}
	if a.rdb != nil {
		items, _ := a.rdb.ZRevRangeWithScores(a.ctx, "vault:referrers", 0, 9).Result()
		for _, item := range items {
			member, _ := item.Member.(string)
			referrers = append(referrers, relayRank{Referrer: member, Score: item.Score})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"referrers":   referrers,
		"socialStats": a.social.Stats(),
	})
}

func (a *App) checkAdminAccessHandler(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimSpace(r.URL.Query().Get("address"))
	if address == "" {
		address = strings.TrimSpace(r.Header.Get("X-Admin-Wallet"))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"address": address,
		"access":  address != "",
	})
}

func (a *App) distributionHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"distribution": a.social.Distribution(),
	})
}

func (a *App) executeMint(to string) (string, error) {
	if a.client == nil || len(a.relayers) == 0 || a.chainID == nil {
		return fmt.Sprintf("simulated-%d", time.Now().UnixNano()), nil
	}

	idx := atomic.AddUint64(&a.relayerCounter, 1) % uint64(len(a.relayers))
	relayer := a.relayers[idx]
	relayer.mu.Lock()
	defer relayer.mu.Unlock()

	gasPrice, err := a.client.SuggestGasPrice(a.ctx)
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(uint64(relayer.Nonce), common.HexToAddress(to), big.NewInt(0), 100000, gasPrice, nil)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(a.chainID), relayer.PrivateKey)
	if err != nil {
		return "", err
	}

	if err := a.client.SendTransaction(a.ctx, signed); err != nil {
		return "", err
	}

	relayer.Nonce++
	return signed.Hash().Hex(), nil
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func parseLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 100 {
		return 100
	}
	return value
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyCORSHeaders(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func applyCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" && isAllowedOrigin(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Wallet")
}

func isAllowedOrigin(origin string) bool {
	configured := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if configured != "" {
		for _, item := range strings.Split(configured, ",") {
			if strings.EqualFold(strings.TrimSpace(item), origin) {
				return true
			}
		}
		return false
	}

	allowedPrefixes := []string{
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
		"http://[::1]",
		"https://[::1]",
		"http://192.168.",
		"https://192.168.",
		"http://10.",
		"https://10.",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(origin, prefix) {
			return true
		}
	}

	return false
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.code = code
	sw.ResponseWriter.WriteHeader(code)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s [origin=%s]", r.Method, r.URL.Path, sw.code, time.Since(start).Round(time.Millisecond), r.Header.Get("Origin"))
	})
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
