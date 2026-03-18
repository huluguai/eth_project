package db

import (
	"fmt"

	"eth_project/go_eth/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Open(dsn string) (*gorm.DB, error) {
	// 使用纯 Go 的 sqlite driver（glebarez/sqlite），避免本地开发需要额外的 CGO / sqlite 动态库依赖。
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
