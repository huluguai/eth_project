package http

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"eth_project/go_eth/internal/models"

	"github.com/gin-gonic/gin"
)

type transferDTO struct {
	TokenAddress string `json:"tokenAddress"`
	TxHash       string `json:"txHash"`
	LogIndex     uint   `json:"logIndex"`
	BlockNumber  uint64 `json:"blockNumber"`
	FromAddress  string `json:"from"`
	ToAddress    string `json:"to"`
	Amount       string `json:"amount"`
}

type transfersResp struct {
	Items      []transferDTO `json:"items"`
	NextCursor string        `json:"nextCursor,omitempty"`
}

func (h *Handlers) GetTransfers(c *gin.Context) {
	addrAny, ok := c.Get(CtxAddressKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing auth context"})
		return
	}
	address := strings.ToLower(strings.TrimSpace(fmt.Sprint(addrAny)))

	limit := parseLimit(c.Query("limit"), 50, 1, 200)
	cursor := strings.TrimSpace(c.Query("cursor"))

	// 这里使用 (block_number DESC, log_index DESC) 作为稳定排序键：
	// - 同一 block 内的日志顺序由 log_index 决定
	// - 用两列组合可实现“确定性”分页，避免只按 block_number 时出现重复/跳过。
	q := h.DB.Model(&models.Transfer{}).
		Where("(from_address = ? OR to_address = ?)", address, address).
		Order("block_number DESC").Order("log_index DESC").
		Limit(limit)

	if cursor != "" {
		// 游标分页：cursor 编码的是上一页最后一条的 (block_number, log_index)。
		// 下一页查询严格“小于”该二元组，保证不会重复返回 last item。
		cb, cl, err := decodeCursor(cursor)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
			return
		}
		q = q.Where("(block_number < ?) OR (block_number = ? AND log_index < ?)", cb, cb, cl)
	}

	var rows []models.Transfer
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	items := make([]transferDTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, transferDTO{
			TokenAddress: r.TokenAddress,
			TxHash:       r.TxHash,
			LogIndex:     r.LogIndex,
			BlockNumber:  r.BlockNumber,
			FromAddress:  r.FromAddress,
			ToAddress:    r.ToAddress,
			Amount:       r.Amount,
		})
	}

	resp := transfersResp{Items: items}
	if len(rows) == limit {
		last := rows[len(rows)-1]
		resp.NextCursor = encodeCursor(last.BlockNumber, last.LogIndex)
	}
	c.JSON(http.StatusOK, resp)
}

func parseLimit(s string, def, min, max int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func encodeCursor(block uint64, logIndex uint) string {
	// base64-url 编码便于在 URL query 里安全传递；RawURLEncoding 不带 padding，更短也更常见。
	raw := fmt.Sprintf("%d:%d", block, logIndex)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cur string) (uint64, uint, error) {
	b, err := base64.RawURLEncoding.DecodeString(cur)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.SplitN(string(b), ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad cursor")
	}
	blk, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	li64, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return blk, uint(li64), nil
}
