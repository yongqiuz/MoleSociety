package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const (
	authSessionCookieName = "molesociety_session"
	authChallengeTTL      = 5 * time.Minute
	authSessionTTL        = 7 * 24 * time.Hour
)

var (
	errAuthSessionMissing = errors.New("authentication required")
	errAuthSessionInvalid = errors.New("invalid session")
)

type AuthChallenge struct {
	Nonce     string `json:"nonce"`
	Address   string `json:"address"`
	Message   string `json:"message"`
	ChainID   int64  `json:"chainId"`
	IssuedAt  string `json:"issuedAt"`
	ExpiresAt string `json:"expiresAt"`
}

type AuthSessionRecord struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Address   string `json:"address"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

type AuthChallengeRequest struct {
	Address string `json:"address"`
	ChainID int64  `json:"chainId"`
}

type VerifyWalletRequest struct {
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

type PasswordLoginRequest struct {
	Identifier string `json:"identifier"` // username or email
	Password   string `json:"password"`
}

type RegisterRequest struct {
	Username      string `json:"username"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	WalletAddress string `json:"walletAddress"`
	ChainID       int64  `json:"chainId"`
	Signature     string `json:"signature"`
	Nonce         string `json:"nonce"`
}

type BindChallengeRequest struct {
	WalletAddress string `json:"walletAddress"`
	ChainID       int64  `json:"chainId"`
}

type Account struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
	Wallet       string `json:"wallet"`
	UserID       string `json:"userId"` // linked SocialUser ID
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type AuthSessionResponse struct {
	ID           string      `json:"id"`
	Handle       string      `json:"handle"`
	DisplayName  string      `json:"displayName"`
	Instance     string      `json:"instance"`
	Bio          string      `json:"bio"`
	AvatarURL    string      `json:"avatarUrl"`
	Wallet       string      `json:"wallet"`
	Fields       []UserField `json:"fields"`
	FeaturedTags []string    `json:"featuredTags"`
	IsBot        bool        `json:"isBot"`
}

type AuthService struct {
	ctx        context.Context
	rdb        *redis.Client
	social     *SocialService
	mu         sync.Mutex
	challenges map[string]AuthChallenge
	sessions   map[string]AuthSessionRecord
	accounts   map[string]Account // keyed by account ID
}

func NewAuthService(ctx context.Context, rdb *redis.Client, social *SocialService) *AuthService {
	s := &AuthService{
		ctx:        ctx,
		rdb:        rdb,
		social:     social,
		challenges: map[string]AuthChallenge{},
		sessions:   map[string]AuthSessionRecord{},
		accounts:   map[string]Account{},
	}
	s.loadAccountsFromRedis()
	return s
}

func (s *AuthService) loadAccountsFromRedis() {
	if s.rdb == nil {
		return
	}
	raw, err := s.rdb.Get(s.ctx, "auth:accounts:snapshot").Result()
	if err != nil {
		return
	}
	var accounts []Account
	if err := json.Unmarshal([]byte(raw), &accounts); err != nil {
		return
	}
	for _, acc := range accounts {
		s.accounts[acc.ID] = acc
	}
	log.Printf("loaded %d accounts from redis", len(accounts))
}

func (s *AuthService) persistAccountsLocked() {
	if s.rdb == nil {
		return
	}
	accounts := make([]Account, 0, len(s.accounts))
	for _, acc := range s.accounts {
		accounts = append(accounts, acc)
	}
	raw, err := json.Marshal(accounts)
	if err != nil {
		return
	}
	_ = s.rdb.Set(s.ctx, "auth:accounts:snapshot", raw, 0).Err()
}

func (a *App) registerAuthRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/auth/challenge", a.authChallengeHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/verify", a.authVerifyHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/me", a.authMeHandler).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/logout", a.authLogoutHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/password-login", a.authPasswordLoginHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/register", a.authRegisterHandler).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/v1/auth/bind-challenge", a.authBindChallengeHandler).Methods(http.MethodPost, http.MethodOptions)
}

func (a *App) authChallengeHandler(w http.ResponseWriter, r *http.Request) {
	var req AuthChallengeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	chainID := req.ChainID
	if chainID == 0 {
		if a.chainID != nil {
			chainID = a.chainID.Int64()
		} else {
			chainID = 1
		}
	}

	challenge, err := a.auth.CreateChallenge(req.Address, chainID, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": challenge})
}

func (a *App) authVerifyHandler(w http.ResponseWriter, r *http.Request) {
	var req VerifyWalletRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	user, session, err := a.auth.VerifyWalletLogin(req.Address, req.Nonce, req.Signature)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	setSessionCookie(w, session.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": authSessionResponseFromUser(user)})
}

func (a *App) authMeHandler(w http.ResponseWriter, r *http.Request) {
	user, err := a.optionalAuthenticatedUser(r)
	if err != nil {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": errAuthSessionMissing.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": authSessionResponseFromUser(*user)})
}

func (a *App) authLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(authSessionCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		a.auth.DeleteSession(strings.TrimSpace(cookie.Value))
	}

	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": map[string]any{"loggedOut": true}})
}

// --- Password Login Handler ---
func (a *App) authPasswordLoginHandler(w http.ResponseWriter, r *http.Request) {
	var req PasswordLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	password := strings.TrimSpace(req.Password)
	if identifier == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "identifier and password are required"})
		return
	}

	user, session, err := a.auth.PasswordLogin(identifier, password)
	if err != nil {
		log.Printf("password-login failed for %q: %v", identifier, err)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	setSessionCookie(w, session.ID)
	log.Printf("password-login success: user=%s handle=%s", user.ID, user.Handle)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": authSessionResponseFromUser(user)})
}

// --- Register + Bind Wallet Handler ---
func (a *App) authRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "username and password are required"})
		return
	}
	if len(password) < 6 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "password must be at least 6 characters"})
		return
	}

	user, session, err := a.auth.Register(username, strings.TrimSpace(req.Email), password, strings.TrimSpace(req.WalletAddress), req.ChainID, req.Nonce, req.Signature, r)
	if err != nil {
		log.Printf("register failed for %q: %v", username, err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	setSessionCookie(w, session.ID)
	log.Printf("register success: user=%s handle=%s", user.ID, user.Handle)
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "data": authSessionResponseFromUser(user)})
}

// --- Bind Challenge Handler ---
func (a *App) authBindChallengeHandler(w http.ResponseWriter, r *http.Request) {
	var req BindChallengeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid JSON"})
		return
	}

	chainID := req.ChainID
	if chainID == 0 {
		if a.chainID != nil {
			chainID = a.chainID.Int64()
		} else {
			chainID = 1
		}
	}

	challenge, err := a.auth.CreateChallenge(req.WalletAddress, chainID, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": challenge})
}

func (a *App) optionalAuthenticatedUser(r *http.Request) (*SocialUser, error) {
	return a.auth.UserFromRequest(r)
}

func (a *App) requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (*SocialUser, bool) {
	user, err := a.optionalAuthenticatedUser(r)
	if err != nil {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": err.Error()})
		return nil, false
	}
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": errAuthSessionMissing.Error()})
		return nil, false
	}
	return user, true
}

func (s *AuthService) CreateChallenge(address string, chainID int64, r *http.Request) (AuthChallenge, error) {
	normalizedAddress, err := normalizeWalletAddress(address)
	if err != nil {
		return AuthChallenge{}, err
	}

	nonce, err := randomToken(16)
	if err != nil {
		return AuthChallenge{}, fmt.Errorf("generate nonce: %w", err)
	}

	issuedAt := time.Now().UTC()
	expiresAt := issuedAt.Add(authChallengeTTL)

	challenge := AuthChallenge{
		Nonce:     nonce,
		Address:   normalizedAddress,
		Message:   buildWalletSignMessage(normalizedAddress, chainID, requestOriginURI(r), issuedAt, expiresAt, nonce),
		ChainID:   chainID,
		IssuedAt:  issuedAt.Format(time.RFC3339),
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}

	if err := s.saveChallenge(challenge); err != nil {
		return AuthChallenge{}, err
	}

	return challenge, nil
}

func (s *AuthService) VerifyWalletLogin(address, nonce, signature string) (SocialUser, AuthSessionRecord, error) {
	challenge, err := s.getChallenge(strings.TrimSpace(nonce))
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	normalizedAddress, err := normalizeWalletAddress(address)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}
	if !strings.EqualFold(normalizedAddress, challenge.Address) {
		return SocialUser{}, AuthSessionRecord{}, errors.New("wallet address does not match login challenge")
	}

	if err := verifyWalletSignature(challenge.Message, signature, challenge.Address); err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	user, err := s.social.EnsureWalletUser(challenge.Address)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	now := time.Now().UTC()
	sessionID, err := randomToken(32)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("generate session: %w", err)
	}

	session := AuthSessionRecord{
		ID:        sessionID,
		UserID:    user.ID,
		Address:   challenge.Address,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(authSessionTTL).Format(time.RFC3339),
	}

	if err := s.saveSession(session); err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	_ = s.deleteChallenge(challenge.Nonce)
	return user, session, nil
}

// --- Password Login Logic ---
func (s *AuthService) PasswordLogin(identifier, password string) (SocialUser, AuthSessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var account *Account
	lowerIdentifier := strings.ToLower(identifier)
	for _, acc := range s.accounts {
		if strings.ToLower(acc.Username) == lowerIdentifier || (acc.Email != "" && strings.ToLower(acc.Email) == lowerIdentifier) {
			copy := acc
			account = &copy
			break
		}
	}
	if account == nil {
		return SocialUser{}, AuthSessionRecord{}, errors.New("account not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)); err != nil {
		return SocialUser{}, AuthSessionRecord{}, errors.New("invalid password")
	}

	user, err := s.social.GetUser(account.UserID)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, errors.New("linked user not found")
	}

	now := time.Now().UTC()
	sessionID, err := randomToken(32)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("generate session: %w", err)
	}

	session := AuthSessionRecord{
		ID:        sessionID,
		UserID:    user.ID,
		Address:   account.Wallet,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(authSessionTTL).Format(time.RFC3339),
	}

	if err := s.saveSession(session); err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	return *user, session, nil
}

// --- Register + Bind Wallet Logic ---
func (s *AuthService) Register(username, email, password, walletAddress string, chainID int64, nonce, signature string, r *http.Request) (SocialUser, AuthSessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check username uniqueness
	lowerUsername := strings.ToLower(username)
	for _, acc := range s.accounts {
		if strings.ToLower(acc.Username) == lowerUsername {
			return SocialUser{}, AuthSessionRecord{}, errors.New("username already taken")
		}
	}

	// Check email uniqueness if provided
	if email != "" {
		lowerEmail := strings.ToLower(email)
		for _, acc := range s.accounts {
			if acc.Email != "" && strings.ToLower(acc.Email) == lowerEmail {
				return SocialUser{}, AuthSessionRecord{}, errors.New("email already registered")
			}
		}
	}

	// If wallet is provided, verify signature
	normalizedWallet := ""
	if walletAddress != "" {
		var err error
		normalizedWallet, err = normalizeWalletAddress(walletAddress)
		if err != nil {
			return SocialUser{}, AuthSessionRecord{}, err
		}

		// Check wallet uniqueness
		for _, acc := range s.accounts {
			if strings.EqualFold(acc.Wallet, normalizedWallet) {
				return SocialUser{}, AuthSessionRecord{}, errors.New("wallet already bound to another account")
			}
		}

		// Verify bind challenge signature
		if nonce != "" && signature != "" {
			challenge, err := s.getChallengeNoLock(nonce)
			if err != nil {
				return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("bind challenge: %w", err)
			}
			if !strings.EqualFold(normalizedWallet, challenge.Address) {
				return SocialUser{}, AuthSessionRecord{}, errors.New("wallet address does not match bind challenge")
			}
			if err := verifyWalletSignature(challenge.Message, signature, challenge.Address); err != nil {
				return SocialUser{}, AuthSessionRecord{}, err
			}
			_ = s.deleteChallengeNoLock(nonce)
		}
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("hash password: %w", err)
	}

	// Create social user
	user, err := s.social.CreateUser(CreateUserRequest{
		Handle:      username,
		DisplayName: username,
		Bio:         "MoleSociety member",
		Instance:    "vault.social",
		Wallet:      normalizedWallet,
	})
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("create user: %w", err)
	}

	now := time.Now().UTC()
	accountID := "acc_" + fmt.Sprintf("%d", now.UnixNano())
	account := Account{
		ID:           accountID,
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Wallet:       normalizedWallet,
		UserID:       user.ID,
		Status:       "active",
		CreatedAt:    now.Format(time.RFC3339),
		UpdatedAt:    now.Format(time.RFC3339),
	}
	s.accounts[accountID] = account
	s.persistAccountsLocked()

	// Create session
	sessionID, err := randomToken(32)
	if err != nil {
		return SocialUser{}, AuthSessionRecord{}, fmt.Errorf("generate session: %w", err)
	}

	session := AuthSessionRecord{
		ID:        sessionID,
		UserID:    user.ID,
		Address:   normalizedWallet,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(authSessionTTL).Format(time.RFC3339),
	}

	if err := s.saveSession(session); err != nil {
		return SocialUser{}, AuthSessionRecord{}, err
	}

	return user, session, nil
}

func (s *AuthService) UserFromRequest(r *http.Request) (*SocialUser, error) {
	cookie, err := r.Cookie(authSessionCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, nil
		}
		return nil, errAuthSessionInvalid
	}

	sessionID := strings.TrimSpace(cookie.Value)
	if sessionID == "" {
		return nil, nil
	}

	session, err := s.getSession(sessionID)
	if err != nil {
		if errors.Is(err, errAuthSessionMissing) || errors.Is(err, errAuthSessionInvalid) {
			return nil, errAuthSessionInvalid
		}
		return nil, err
	}

	user, err := s.social.GetUser(session.UserID)
	if err != nil {
		_ = s.DeleteSession(sessionID)
		return nil, errAuthSessionInvalid
	}

	return user, nil
}

func (s *AuthService) DeleteSession(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}

	if s.rdb != nil {
		return s.rdb.Del(s.ctx, authSessionKey(sessionID)).Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
	return nil
}

func (s *AuthService) saveChallenge(challenge AuthChallenge) error {
	if s.rdb != nil {
		return saveJSONWithTTL(s.ctx, s.rdb, authChallengeKey(challenge.Nonce), challenge, authChallengeTTL)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.challenges[challenge.Nonce] = challenge
	return nil
}

func (s *AuthService) getChallenge(nonce string) (AuthChallenge, error) {
	if strings.TrimSpace(nonce) == "" {
		return AuthChallenge{}, errors.New("nonce is required")
	}

	if s.rdb != nil {
		var challenge AuthChallenge
		if err := loadJSON(s.ctx, s.rdb, authChallengeKey(nonce), &challenge); err != nil {
			if errors.Is(err, redis.Nil) {
				return AuthChallenge{}, errors.New("login challenge expired or missing")
			}
			return AuthChallenge{}, err
		}
		if isExpired(challenge.ExpiresAt) {
			_ = s.deleteChallenge(nonce)
			return AuthChallenge{}, errors.New("login challenge expired or missing")
		}
		return challenge, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[nonce]
	if !ok || isExpired(challenge.ExpiresAt) {
		delete(s.challenges, nonce)
		return AuthChallenge{}, errors.New("login challenge expired or missing")
	}
	return challenge, nil
}

func (s *AuthService) deleteChallenge(nonce string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteChallengeNoLock(nonce)
}

func (s *AuthService) deleteChallengeNoLock(nonce string) error {
	if s.rdb != nil {
		return s.rdb.Del(s.ctx, authChallengeKey(nonce)).Err()
	}

	delete(s.challenges, nonce)
	return nil
}

func (s *AuthService) getChallengeNoLock(nonce string) (AuthChallenge, error) {
	if strings.TrimSpace(nonce) == "" {
		return AuthChallenge{}, errors.New("nonce is required")
	}

	if s.rdb != nil {
		var challenge AuthChallenge
		if err := loadJSON(s.ctx, s.rdb, authChallengeKey(nonce), &challenge); err != nil {
			if errors.Is(err, redis.Nil) {
				return AuthChallenge{}, errors.New("bind challenge expired or missing")
			}
			return AuthChallenge{}, err
		}
		if isExpired(challenge.ExpiresAt) {
			_ = s.deleteChallengeNoLock(nonce)
			return AuthChallenge{}, errors.New("bind challenge expired or missing")
		}
		return challenge, nil
	}

	challenge, ok := s.challenges[nonce]
	if !ok || isExpired(challenge.ExpiresAt) {
		delete(s.challenges, nonce)
		return AuthChallenge{}, errors.New("bind challenge expired or missing")
	}
	return challenge, nil
}

func (s *AuthService) saveSession(session AuthSessionRecord) error {
	if s.rdb != nil {
		return saveJSONWithTTL(s.ctx, s.rdb, authSessionKey(session.ID), session, authSessionTTL)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *AuthService) getSession(sessionID string) (AuthSessionRecord, error) {
	if strings.TrimSpace(sessionID) == "" {
		return AuthSessionRecord{}, errAuthSessionMissing
	}

	if s.rdb != nil {
		var session AuthSessionRecord
		if err := loadJSON(s.ctx, s.rdb, authSessionKey(sessionID), &session); err != nil {
			if errors.Is(err, redis.Nil) {
				return AuthSessionRecord{}, errAuthSessionInvalid
			}
			return AuthSessionRecord{}, err
		}
		if isExpired(session.ExpiresAt) {
			_ = s.DeleteSession(sessionID)
			return AuthSessionRecord{}, errAuthSessionInvalid
		}
		return session, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok || isExpired(session.ExpiresAt) {
		delete(s.sessions, sessionID)
		return AuthSessionRecord{}, errAuthSessionInvalid
	}
	return session, nil
}

func saveJSONWithTTL(ctx context.Context, rdb *redis.Client, key string, payload any, ttl time.Duration) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, raw, ttl).Err()
}

func loadJSON(ctx context.Context, rdb *redis.Client, key string, target any) error {
	raw, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), target)
}

func authChallengeKey(nonce string) string {
	return "auth:challenge:" + nonce
}

func authSessionKey(sessionID string) string {
	return "auth:session:" + sessionID
}

func authSessionResponseFromUser(user SocialUser) AuthSessionResponse {
	return AuthSessionResponse{
		ID:          user.ID,
		Handle:      user.Handle,
		DisplayName: user.DisplayName,
		Instance:    user.Instance,
		Bio:         user.Bio,
		AvatarURL:   user.AvatarURL,
		Wallet:      user.Wallet,
	}
}

func normalizeWalletAddress(address string) (string, error) {
	trimmed := strings.TrimSpace(address)
	if !common.IsHexAddress(trimmed) {
		return "", errors.New("invalid wallet address")
	}
	return common.HexToAddress(trimmed).Hex(), nil
}

func verifyWalletSignature(message, signature, expectedAddress string) error {
	decoded := common.FromHex(strings.TrimSpace(signature))
	if len(decoded) != crypto.SignatureLength {
		return errors.New("invalid wallet signature")
	}

	signatureCopy := append([]byte(nil), decoded...)
	if signatureCopy[crypto.RecoveryIDOffset] >= 27 {
		signatureCopy[crypto.RecoveryIDOffset] -= 27
	}
	if signatureCopy[crypto.RecoveryIDOffset] > 1 {
		return errors.New("invalid wallet signature")
	}

	hash := accounts.TextHash([]byte(message))
	publicKey, err := crypto.SigToPub(hash, signatureCopy)
	if err != nil {
		return errors.New("failed to verify wallet signature")
	}

	recovered := crypto.PubkeyToAddress(*publicKey).Hex()
	if !strings.EqualFold(recovered, expectedAddress) {
		return errors.New("wallet signature does not match address")
	}

	return nil
}

func buildWalletSignMessage(address string, chainID int64, uri string, issuedAt time.Time, expiresAt time.Time, nonce string) string {
	return fmt.Sprintf(
		"MoleSociety wants you to sign in with your wallet:\n%s\n\nSign in to MoleSociety.\n\nURI: %s\nVersion: 1\nChain ID: %d\nNonce: %s\nIssued At: %s\nExpiration Time: %s",
		address,
		uri,
		chainID,
		nonce,
		issuedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339),
	)
}

func requestOriginURI(r *http.Request) string {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		return origin
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "localhost"
	}
	if parsed, err := url.Parse(fmt.Sprintf("%s://%s", scheme, host)); err == nil {
		return parsed.String()
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func randomToken(byteLength int) (string, error) {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func isExpired(timestamp string) bool {
	if strings.TrimSpace(timestamp) == "" {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return true
	}
	return time.Now().UTC().After(expiresAt)
}

func setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sessionCookieSecure(),
		MaxAge:   int(authSessionTTL.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   sessionCookieSecure(),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func sessionCookieSecure() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true")
}
