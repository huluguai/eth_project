package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"eth_project/go_eth/internal/models"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type siweNonceResp struct {
	Nonce string `json:"nonce"`
}

// SiweNonce 生成一次性 nonce，供客户端构造 SIWE message 后完成签名登录。
func (h *Handlers) SiweNonce(c *gin.Context) {
	nonce, err := NewNonce(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate nonce"})
		return
	}
	now := time.Now()
	rec := models.SiweNonce{
		Nonce: nonce,
		// nonce 只在短时间内有效，用于抵抗重放；过期后即使签名正确也应拒绝。
		ExpiresAt: now.Add(5 * time.Minute),
	}
	if err := h.DB.Create(&rec).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist nonce"})
		return
	}
	c.JSON(http.StatusOK, siweNonceResp{Nonce: nonce})
}

type siweLoginReq struct {
	Message   string `json:"message" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

type siweLoginResp struct {
	Token   string `json:"token"`
	Address string `json:"address"`
}

// SiweLogin 校验 SIWE message + 签名，并为已验证的钱包签发 JWT。
func (h *Handlers) SiweLogin(c *gin.Context) {
	var req siweLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	msg := strings.TrimSpace(req.Message)
	sig := strings.TrimSpace(req.Signature)

	// 先解析消息再校验签名：这样可以先做域名/chainId/nonce 等“便宜校验”，避免无意义的椭圆曲线恢复开销。
	parsed, err := ParseSIWEMessage(msg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if parsed.ChainID != h.Cfg.ChainID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong chain id"})
		return
	}
	if h.Cfg.AllowedDomain != "" && !strings.EqualFold(parsed.Domain, h.Cfg.AllowedDomain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain not allowed"})
		return
	}

	// nonce 必须来自服务端发放且未过期/未使用：这是 SIWE 防重放的核心。
	var nonce models.SiweNonce
	if err := h.DB.First(&nonce, "nonce = ?", parsed.Nonce).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid nonce"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	if nonce.UsedAt != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "nonce already used"})
		return
	}
	if time.Now().After(nonce.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "nonce expired"})
		return
	}

	// 按 personal_sign（EIP-191）规则恢复地址；这与 eth_sign / EIP-712 的哈希方式不同。
	recovered, err := recoverAddressPersonalSign(msg, sig)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}
	if strings.ToLower(recovered) != strings.ToLower(parsed.Address) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "signature address mismatch"})
		return
	}

	now := time.Now()
	// 原子地“消费 nonce”：要求 nonce 未使用且未过期，并检查 RowsAffected，确保并发情况下只有一个请求能签发 JWT。
	consume := h.DB.Model(&models.SiweNonce{}).
		Where("nonce = ? AND used_at IS NULL AND expires_at > ?", parsed.Nonce, now).
		Updates(map[string]any{"address": strings.ToLower(parsed.Address), "used_at": &now})
	if consume.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark nonce used"})
		return
	}
	if consume.RowsAffected != 1 {
		// 可能原因：并发请求已先消费了 nonce，或在此处短时间内已过期。
		if time.Now().After(nonce.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "nonce expired"})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "nonce already used"})
		}
		return
	}

	jwtt, err := issueJWT(h.Cfg.JWTSecret, strings.ToLower(parsed.Address), time.Now().Add(24*time.Hour))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign jwt"})
		return
	}

	c.JSON(http.StatusOK, siweLoginResp{Token: jwtt, Address: strings.ToLower(parsed.Address)})
}

func issueJWT(secret, address string, exp time.Time) (string, error) {
	claims := jwt.MapClaims{
		// sub 作为主体标识：这里用钱包地址，配合中间件写入 ctx，驱动后续数据查询。
		"sub": address,
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(secret))
}

// recoverAddressPersonalSign 使用 EIP-191（personal_sign）规则恢复签名对应的钱包地址。
func recoverAddressPersonalSign(message, sigHex string) (string, error) {
	sigHex = strings.TrimPrefix(sigHex, "0x")
	sigBytes, err := hexutil.Decode("0x" + sigHex)
	if err != nil {
		return "", err
	}
	if len(sigBytes) != 65 {
		return "", errors.New("signature must be 65 bytes")
	}
	// 兼容部分钱包/库返回的 v 为 27/28（而 go-ethereum 期望 0/1）。
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}
	if sigBytes[64] != 0 && sigBytes[64] != 1 {
		return "", errors.New("invalid recovery id")
	}

	// accounts.TextHash 会加上 "\x19Ethereum Signed Message:\n" 前缀（EIP-191），对应 personal_sign。
	hash := accounts.TextHash([]byte(message))
	pub, err := crypto.SigToPub(hash, sigBytes)
	if err != nil {
		return "", err
	}
	addr := crypto.PubkeyToAddress(*pub)
	return addr.Hex(), nil
}
