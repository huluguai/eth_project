// Package db 提供数据库连接与初始化相关的封装。
package db

import (
	"fmt"
	"strings"

	"eth_project/go_eth/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Open 打开（或创建）sqlite 数据库连接，并在必要时执行自动迁移。
func Open(dsn string) (*gorm.DB, error) {
	// 使用纯 Go 的 sqlite driver（glebarez/sqlite），避免本地开发需要额外的 CGO / sqlite 动态库依赖。
	// 如果调用方没有显式配置 journal_mode，则默认启用 WAL，提升多连接并发读写的可靠性。
	if !strings.Contains(strings.ToLower(dsn), "journal_mode=") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn = dsn + sep + "_journal_mode=WAL"
	}
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// AutoMigrate 让本地/测试环境“开箱即用”。
	// 生产环境若需要更可控的变更流程，可替换为显式 migration（这里保持轻量）。
	if err := gdb.AutoMigrate(&models.Transfer{}, &models.IndexerState{}, &models.SiweNonce{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return gdb, nil
}
