package models

import "time"

type IndexerState struct {
	Key                string    `gorm:"primaryKey;size:128"`
	LastProcessedBlock uint64    `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}
