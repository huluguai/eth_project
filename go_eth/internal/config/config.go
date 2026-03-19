// Package config 负责从环境变量/默认值构建应用配置。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 表示服务运行所需的配置集合。
type Config struct {
	HTTPAddr      string
	EthRPCURL     string
	TokenAddress  string
	TokenAddresses []string
	ChainID       int64
	StartBlock    *uint64
	Confirmations uint64
	PollInterval  time.Duration

	DBDSN     string
	JWTSecret string

	AllowedDomain string
}

// Load 从环境变量加载配置，并对必须项做基础校验。
func Load() (Config, error) {
	// TOKEN_ADDRESSES 优先：支持逗号分隔的多合约地址。
	// 若为空，则回退到单地址 TOKEN_ADDRESS（保持向后兼容）。
	//
	// 注意：这里只做轻量格式校验（去空格、lower-case、长度/0x 前缀），
	// 更严格的校验/解析由 indexer 层完成即可。
	rawTokenAddresses := strings.TrimSpace(getEnv("TOKEN_ADDRESSES", ""))
	var tokenAddresses []string
	if rawTokenAddresses != "" {
		for _, part := range strings.Split(rawTokenAddresses, ",") {
			a := strings.ToLower(strings.TrimSpace(part))
			if a == "" {
				continue
			}
			// 0x + 40 hex chars
			if !strings.HasPrefix(a, "0x") || len(a) != 42 {
				return Config{}, fmt.Errorf("invalid TOKEN_ADDRESSES item: %q", a)
			}
			tokenAddresses = append(tokenAddresses, a)
		}
	}

	if len(tokenAddresses) == 0 {
		a := strings.ToLower(getEnv("TOKEN_ADDRESS", "0x0b18F517d8e66b3bd6fB799d44A0ebee473Df20C"))
		if !strings.HasPrefix(a, "0x") || len(a) != 42 {
			return Config{}, fmt.Errorf("invalid TOKEN_ADDRESS: %q", a)
		}
		tokenAddresses = []string{a}
	}

	// 去重（避免重复地址导致 indexer_state/key 变化但业务不变）。
	deduped := make([]string, 0, len(tokenAddresses))
	seen := make(map[string]struct{}, len(tokenAddresses))
	for _, a := range tokenAddresses {
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		deduped = append(deduped, a)
	}

	// 为兼容老字段：把第一个 token 作为 TokenAddress。
	tokenAddress := deduped[0]

	cfg := Config{
		HTTPAddr:  getEnv("HTTP_ADDR", ":8080"),
		EthRPCURL: getEnv("ETH_RPC_URL", "https://sepolia.drpc.org"),
		TokenAddress: tokenAddress,
		// TokenAddresses 是多合约索引的输入集合。
		TokenAddresses: deduped,
		ChainID:       mustInt64(getEnv("CHAIN_ID", "11155111")),
		Confirmations: mustUint64(getEnv("CONFIRMATIONS", "6")),
		PollInterval:  mustDuration(getEnv("POLL_INTERVAL", "8s")),
		// sqlite DSN：
		// - busy_timeout 避免并发写入时立即报 “database is locked”
		// - journal_mode=WAL 提高读写并发能力（多实例场景更稳）
		// - foreign_keys 打开外键约束（若后续加关联表）
		DBDSN: getEnv("DB_DSN", "file:go_eth.db?_busy_timeout=5000&_foreign_keys=1&_journal_mode=WAL"),
		// JWT_SECRET 必须由部署方提供；不设默认值，避免“弱 secret”在不知情情况下进入生产。
		JWTSecret:     getEnv("JWT_SECRET", ""),
		AllowedDomain: strings.TrimSpace(getEnv("ALLOWED_DOMAIN", "")),
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	// StartBlock 为可选项：不设时索引器会从“最新确认高度”附近开始（更快冷启动）。
	if start := strings.TrimSpace(os.Getenv("START_BLOCK")); start != "" {
		v, err := strconv.ParseUint(start, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid START_BLOCK: %w", err)
		}
		cfg.StartBlock = &v
	}

	return cfg, nil
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func mustInt64(s string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		// 这些 must* 用于“配置即失败”：让错误尽早暴露，避免服务带着错误配置跑起来后产生隐蔽行为。
		panic(err)
	}
	return v
}

func mustUint64(s string) uint64 {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

func mustDuration(s string) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		panic(err)
	}
	return d
}
