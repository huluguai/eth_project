package models

import "time"

type IndexerState struct {
	// Key 用于区分不同 token/实例的索引进度，避免多实例共享同一进度导致重复扫或跳块。
	Key                string    `gorm:"primaryKey;size:128"`
	LastProcessedBlock uint64    `gorm:"not null"`
	// BlockHash 为 LastProcessedBlock 对应区块头 hash（hex 字符串）。
	// 用于检测链重组（reorg）时“同高度但不同区块”的情况，从而触发回退重扫。
	BlockHash          string    `gorm:"size:66"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}
