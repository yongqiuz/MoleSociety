//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var (
	ctx    = context.Background()
	rdb    *redis.Client
	client *ethclient.Client
)

type CommonResponse struct {
	Ok     bool   `json:"ok,omitempty"`
	Status string `json:"status,omitempty"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

func main() {
	godotenv.Load()
	rdb = redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})

	var err error
	client, err = ethclient.Dial(os.Getenv("RPC_URL"))
	if err != nil {
		log.Fatalf("无法连接到 RPC: %v", err)
	}

	router := mux.NewRouter()

	// --- 路由 1: 自动补录型预检接口 ---
	router.HandleFunc("/secret/verify", func(w http.ResponseWriter, r *http.Request) {
		codeHash := r.URL.Query().Get("codeHash")
		address := r.URL.Query().Get("address")

		// 1. 如果用户提供了地址，进行确权或自动补录
		if address != "" {
			savedAddr, err := rdb.Get(ctx, "bind:"+codeHash).Result()
			if err == redis.Nil {
				// 【核心逻辑】如果已领过但没记录绑定关系（老数据）
				// 检查它是否在已领集合中，若是，则将此地址作为“主地址”补录
				isUsed, _ := rdb.SIsMember(ctx, "vault:codes:used", codeHash).Result()
				if isUsed {
					rdb.Set(ctx, "bind:"+codeHash, address, 0)
					sendJSON(w, http.StatusOK, CommonResponse{Ok: true, Status: "ALREADY_OWNED"})
					return
				}
			} else if err == nil && savedAddr == address {
				// 正常匹配成功
				sendJSON(w, http.StatusOK, CommonResponse{Ok: true, Status: "ALREADY_OWNED"})
				return
			}
		}

		// 2. 排除错误项逻辑
		isValid, _ := rdb.SIsMember(ctx, "vault:codes:valid", codeHash).Result()
		if !isValid {
			isUsed, _ := rdb.SIsMember(ctx, "vault:codes:used", codeHash).Result()
			if isUsed {
				// 返回 409 触发前端确权 UI
				sendJSON(w, http.StatusConflict, CommonResponse{Ok: false, Error: "USED"})
			} else {
				sendJSON(w, http.StatusForbidden, CommonResponse{Ok: false, Error: "INVALID"})
			}
			return
		}
		sendJSON(w, http.StatusOK, CommonResponse{Ok: true})
	}).Methods("GET")

	// --- 路由 2: 链上铸造接口 (含实时绑定) ---
	router.HandleFunc("/relay/mint", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Dest     string `json:"dest"`
			CodeHash string `json:"codeHash"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSON(w, http.StatusBadRequest, CommonResponse{Error: "参数错误"})
			return
		}

		valid, _ := rdb.SIsMember(ctx, "vault:codes:valid", req.CodeHash).Result()
		if !valid {
			sendJSON(w, http.StatusForbidden, CommonResponse{Error: "兑换码无效"})
			return
		}

		txHash, err := executeMint(req.Dest)
		if err != nil {
			sendJSON(w, http.StatusInternalServerError, CommonResponse{Error: "铸造失败"})
			return
		}

		// 原子化操作：不仅标记已用，且立即绑定
		pipe := rdb.Pipeline()
		pipe.SRem(ctx, "vault:codes:valid", req.CodeHash)
		pipe.SAdd(ctx, "vault:codes:used", req.CodeHash)
		pipe.Set(ctx, "bind:"+req.CodeHash, req.Dest, 0)
		pipe.Exec(ctx)

		go notifyMatrix(req.Dest, txHash) //
		sendJSON(w, http.StatusOK, CommonResponse{Status: "submitted", TxHash: txHash})
	}).Methods("POST")

	cors := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	fmt.Printf("[%s] 🚀 自动补录版后端已启动 :8080\n", time.Now().Format("15:04:05"))
	http.ListenAndServe(":8080", cors(router))
}

func executeMint(destAddr string) (string, error) {
	privateKey, _ := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, _ := client.PendingNonceAt(ctx, fromAddress)
	gasPrice, _ := client.SuggestGasPrice(ctx)
	chainID, _ := strconv.Atoi(os.Getenv("CHAIN_ID"))
	data := append(common.FromHex("6a627842"), common.LeftPadBytes(common.HexToAddress(destAddr).Bytes(), 32)...)
	tx := types.NewTransaction(nonce, common.HexToAddress(os.Getenv("CONTRACT_ADDR")), big.NewInt(0), 200000, gasPrice, data)
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(int64(chainID))), privateKey)
	err := client.SendTransaction(ctx, signedTx)
	return signedTx.Hash().Hex(), err
}

func notifyMatrix(dest, txHash string) {
	msg := fmt.Sprintf("🎉 鲸鱼金库：新 NFT 铸造！\n地址: %s\n哈希: %s", dest, txHash)
	url := fmt.Sprintf("%s/_matrix/client/r0/rooms/%s/send/m.room.message?access_token=%s",
		os.Getenv("MATRIX_URL"), os.Getenv("MATRIX_ROOM_ID"), os.Getenv("MATRIX_ACCESS_TOKEN"))
	payload, _ := json.Marshal(map[string]interface{}{"msgtype": "m.text", "body": msg})
	http.Post(url, "application/json", bytes.NewBuffer(payload))
}

func sendJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}
