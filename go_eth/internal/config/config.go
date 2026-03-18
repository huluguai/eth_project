package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr      string
	EthRPCURL     string
	TokenAddress  string
	ChainID       int64
	StartBlock    *uint64
	Confirmations uint64
	PollInterval  time.Duration

	DBDSN     string
	JWTSecret string

	AllowedDomain string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:  getEnv("HTTP_ADDR", ":8080"),
		EthRPCURL: getEnv("ETH_RPC_URL", "https://sepolia.drpc.org"),
		// TokenAddress/地址字段统一 lower-case，避免写库与查询时因大小写不一致导致漏匹配。
		TokenAddress:  strings.ToLower(getEnv("TOKEN_ADDRESS", "0x0b18F517d8e66b3bd6fB799d44A0ebee473Df20C")),
		ChainID:       mustInt64(getEnv("CHAIN_ID", "11155111")),
		Confirmations: mustUint64(getEnv("CONFIRMATIONS", "6")),
		PollInterval:  mustDuration(getEnv("POLL_INTERVAL", "8s")),
		// sqlite DSN：busy_timeout 避免并发写入时立即报 “database is locked”；foreign_keys 打开外键约束（若后续加关联表）。
		DBDSN: getEnv("DB_DSN", "file:go_eth.db?_busy_timeout=5000&_foreign_keys=1"),
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
