package models

import "time"

type SiweNonce struct {
	// Nonce 是服务端生成并下发给客户端的随机字符串。
	Nonce     string     `gorm:"primaryKey;size:64"`
	Address   string     `gorm:"size:42;index"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	UsedAt    *time.Time `gorm:"index"`
	CreatedAt time.Time  `gorm:"autoCreateTime"`
}
