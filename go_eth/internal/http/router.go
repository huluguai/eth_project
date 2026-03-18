package http

import (
	"eth_project/go_eth/internal/config"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(cfg config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	h := &Handlers{
		Cfg: cfg,
		DB:  db,
	}

	// SIWE 登录：先取 nonce，再提交 message+signature 换取 JWT。
	auth := r.Group("/auth/siwe")
	{
		auth.POST("/nonce", h.SiweNonce)
		auth.POST("/login", h.SiweLogin)
	}

	api := r.Group("/api")
	// API 默认要求 JWT；中间件会把钱包地址写入 ctx，handler 只关心业务逻辑。
	api.Use(JWTMiddleware(cfg.JWTSecret))
	{
		api.GET("/transfers", h.GetTransfers)
	}

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	return r
}
