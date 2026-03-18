package http

import (
	"eth_project/go_eth/internal/config"

	"gorm.io/gorm"
)

type Handlers struct {
	Cfg config.Config
	DB  *gorm.DB
}
