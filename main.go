package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// Solana 常量
const (
	rpcURL         = "https://api.mainnet-beta.solana.com"
	wsURL          = "wss://api.mainnet-beta.solana.com"
	LamportsPerSOL = 1_000_000_000 // 1 SOL = 10^9 lamports
)

// 生成或加载 Solana 密钥对
func generateKeypair() *solana.PrivateKey {
	privateKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		log.Fatalf("生成密钥对失败: %v", err)
	}
	return &privateKey
}

// 提现功能：从一个账户向另一个账户转账 SOL
func withdraw(client *rpc.Client, fromPrivateKey *solana.PrivateKey, toPublicKey solana.PublicKey, amount uint64) {
	// 获取最近的区块哈希
	recent, err := client.GetRecentBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		log.Fatalf("获取区块哈希失败: %v", err)
	}

	// 创建转账指令
	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				amount,                     // 转账金额（lamports）
				fromPrivateKey.PublicKey(), // 付款方公钥
				toPublicKey,                // 收款方公钥
			).Build(),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(fromPrivateKey.PublicKey()),
	)
	if err != nil {
		log.Fatalf("创建交易失败: %v", err)
	}

	// 签名交易
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if key.Equals(fromPrivateKey.PublicKey()) {
				return fromPrivateKey
			}
			return nil
		},
	)
	if err != nil {
		log.Fatalf("签名交易失败: %v", err)
	}

	// 发送交易
	sig, err := client.SendTransactionWithOpts(
		context.Background(),
		tx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		log.Fatalf("发送交易失败: %v", err)
	}

	fmt.Printf("提现交易成功，交易签名: %s\n", sig.String())
}

// 充值检测：监听账户余额变化
func monitorDeposits(wsClient *ws.Client, account solana.PublicKey) {
	// 订阅账户变化
	sub, err := wsClient.AccountSubscribe(
		account,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		log.Fatalf("订阅账户失败: %v", err)
	}
	defer sub.Unsubscribe()

	fmt.Printf("开始监听账户 %s 的充值...\n", account.String())

	for {
		result, err := sub.Recv(context.Background())
		if err != nil {
			log.Printf("接收订阅数据失败: %v", err)
			continue
		}

		// 检查账户余额变化
		lamports := result.Value.Lamports
		sol := float64(lamports) / LamportsPerSOL
		fmt.Printf("检测到账户更新！账户: %s, 余额: %.9f SOL\n", account.String(), sol)
	}
}

func main() {
	// 初始化 RPC 和 WebSocket 客户端
	rpcClient := rpc.New(rpcURL)
	wsClient, err := ws.Connect(context.Background(), wsURL)
	if err != nil {
		log.Fatalf("连接 WebSocket 失败: %v", err)
	}
	defer wsClient.Close()

	// 生成或加载账户
	sender := generateKeypair()
	receiverPubKey, err := solana.PublicKeyFromBase58("RECIPIENT_PUBLIC_KEY_HERE") // 替换为实际接收者公钥
	if err != nil {
		log.Fatalf("无效的接收者公钥: %v", err)
	}

	fmt.Printf("发送者公钥: %s\n", sender.PublicKey().String())

	// 执行提现（示例：转账 0.1 SOL）
	amount := uint64(0.1 * LamportsPerSOL) // 0.1 SOL
	go withdraw(rpcClient, sender, receiverPubKey, amount)

	// 监听充值
	monitorDeposits(wsClient, sender.PublicKey())
}
