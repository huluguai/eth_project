// Package http 封装基于 Gin 的 HTTP 路由与处理器。
package http

import (
	"eth_project/go_eth/internal/config"

	"gorm.io/gorm"
)

type Handlers struct {
	// Handlers 作为各 HTTP endpoint 的依赖容器（配置 + 数据库）。
	// Cfg 保存运行时配置（例如链 ID、JWT secret、允许域名）。
	Cfg config.Config
	// DB 为所有处理器提供数据库访问能力。
	DB  *gorm.DB
}
