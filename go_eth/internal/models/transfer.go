package models

import "time"

type Transfer struct {
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
