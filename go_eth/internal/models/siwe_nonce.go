package models

import "time"

type SiweNonce struct {
	Nonce     string     `gorm:"primaryKey;size:64"`
	Address   string     `gorm:"size:42;index"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	UsedAt    *time.Time `gorm:"index"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
}
