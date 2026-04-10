//go:build ignore
// +build ignore

package main

import (
	"context"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
)

// --- 配置区 ---
const (
	RPC_URL       = "https://rpc.api.moonbase.moonbeam.network"
	CHAIN_ID      = 1287
	CONTRACT_ADDR = "0xd0d2380ff21B0daB5Cd75DDA064146a6d36dC6C2"                       // 填入你部署的合约地址
	PRIVATE_KEY   = "f5e9d1dc4dcd90bb0e0b9350c8aa5973011635729926387256ac5ea66324ed2b" // 填入私钥（不带0x）
	HASH_FILE     = "/opt/Whale-Vault/backend/hash-code.txt"
)

type RelayRequest struct {
	Dest     string `json:"dest"`
	CodeHash string `json:"codeHash"`
}

type RelayResponse struct {
	Status string `json:"status"`
	TxHash string `json:"txHash,omitempty"`
	Error  string `json:"error,omitempty"`
}

// 物理接管：直接读取本地文件校验
func verifyCodeFromFile(codeHash string) (bool, error) {
	content, err := os.ReadFile(HASH_FILE)
	if err != nil {
		log.Printf("错误: 无法读取校验文件 %s: %v", HASH_FILE, err)
		return false, err
	}
	validCode := strings.TrimSpace(string(content))
	// 支持包含匹配（防止换行符干扰）
	return strings.Contains(validCode, codeHash), nil
}

func main() {
	// 彻底移除 Redis 初始化逻辑
	ctx := context.Background()

	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		log.Fatalf("无法连接区块链节点: %v", err)
	}

	router := mux.NewRouter()

	// 核心领取接口
	router.HandleFunc("/relay/mint", func(w http.ResponseWriter, r *http.Request) {
		var req RelayRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// 1. 校验哈希 (来自本地文件)
		ok, err := verifyCodeFromFile(req.CodeHash)
		if !ok || err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(RelayResponse{Status: "error", Error: "兑换码无效或服务器配置错误"})
			return
		}

		// 2. 构造交易
		privateKey, err := crypto.HexToECDSA(PRIVATE_KEY)
		if err != nil {
			log.Printf("私钥解析失败: %v", err)
			return
		}

		fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
		nonce, _ := client.PendingNonceAt(ctx, fromAddress)
		gasPrice, _ := client.SuggestGasPrice(ctx)

		toAddr := common.HexToAddress(req.Dest)
		// Mint(address) Selector: 0x6a627842
		data := append(common.FromHex("6a627842"), common.LeftPadBytes(toAddr.Bytes(), 32)...)

		tx := types.NewTransaction(nonce, common.HexToAddress(CONTRACT_ADDR), big.NewInt(0), 150000, gasPrice, data)
		signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(CHAIN_ID)), privateKey)

		// 3. 发送并立即返回结果
		err = client.SendTransaction(ctx, signedTx)
		if err != nil {
			log.Printf("链上发送失败: %v", err)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(RelayResponse{Status: "error", Error: "区块链上链失败"})
			return
		}

		txHash := signedTx.Hash().Hex()
		log.Printf("成功！地址: %s, Hash: %s", req.Dest, txHash)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RelayResponse{Status: "submitted", TxHash: txHash})
	}).Methods("POST")

	log.Printf("🚀 Whale Vault 纯文件验证版已启动: :8080")
	http.ListenAndServe(":8080", router)
}
