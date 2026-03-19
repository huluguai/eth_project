// Package models 定义数据库持久化结构（GORM model）。
package models

import "time"

type Transfer struct {
	// ID 使用自增主键，便于分页/调试；业务幂等主要依赖 TxHash+LogIndex 的唯一约束。
	ID           uint64 `gorm:"primaryKey"`
	TokenAddress string `gorm:"size:42;not null;index"`
	TxHash       string `gorm:"size:66;not null;uniqueIndex:uniq_tx_log"`
	LogIndex     uint   `gorm:"not null;uniqueIndex:uniq_tx_log"`
	BlockNumber  uint64 `gorm:"not null;index"`
	FromAddress  string `gorm:"size:42;not null;index"`
	ToAddress    string `gorm:"size:42;not null;index"`
	Amount       string `gorm:"not null"`
	BlockTime    *time.Time
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}
