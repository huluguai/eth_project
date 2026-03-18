package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const CtxAddressKey = "address"

func JWTMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只接受标准 Bearer token。这里不支持 query/cookie，是为了避免 token 被日志/缓存意外记录。
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		tok, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
			// 明确限制签名算法，避免 alg=none / 算法混淆攻击。
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
			}
			return []byte(secret), nil
		})
		if err != nil || tok == nil || !tok.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := tok.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		// MapClaims 经过 JSON 反序列化后，数字通常会变成 float64。
		if exp, ok := claims["exp"].(float64); ok {
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
				return
			}
		}
		// sub 作为用户标识（这里是钱包地址）。统一 lower-case 便于后续 DB 查询与比较。
		sub, _ := claims["sub"].(string)
		sub = strings.ToLower(strings.TrimSpace(sub))
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}
		c.Set(CtxAddressKey, sub)
		c.Next()
	}
}
