package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eth_project/go_eth/internal/config"
	"eth_project/go_eth/internal/db"
	ethidx "eth_project/go_eth/internal/eth"
	httpapi "eth_project/go_eth/internal/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present (local dev convenience).
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	gdb, err := db.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}

	ethc, err := ethclient.Dial(cfg.EthRPCURL)
	if err != nil {
		log.Fatalf("eth rpc dial error: %v", err)
	}

	// 使用 NotifyContext 统一管理退出信号：HTTP 与 indexer 都共享同一 ctx，便于优雅停止。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	indexer := ethidx.NewIndexer(ethc, gdb, ethidx.IndexerConfig{
		TokenAddress:  cfg.TokenAddress,
		StartBlock:    cfg.StartBlock,
		Confirmations: cfg.Confirmations,
		PollInterval:  cfg.PollInterval,
	})

	// indexer 常驻后台轮询；与 HTTP 并行运行，互不阻塞。
	go func() {
		if err := indexer.Run(ctx); err != nil {
			log.Printf("indexer stopped: %v", err)
		}
	}()

	router := httpapi.NewRouter(cfg, gdb)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("http listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()
	// 给 HTTP 一个有限的 shutdown 窗口，确保在退出时完成正在进行的请求（或超时强退）。
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Printf("bye")
}
