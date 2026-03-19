package http

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type SIWEMessage struct {
	Domain    string
	Address   string
	Statement string
	URI       string
	Version   string
	ChainID   int64
	Nonce     string
	IssuedAt  time.Time
}

// ParseSIWEMessage 将 SIWE（EIP-4361）明文消息解析为结构化字段。
func ParseSIWEMessage(msg string) (SIWEMessage, error) {
	var out SIWEMessage
	lines := splitLinesPreserveEmpty(msg)
	if len(lines) < 2 {
		return out, fmt.Errorf("invalid siwe message: too short")
	}

	first := strings.TrimSpace(lines[0])
	const suffix = " wants you to sign in with your Ethereum account:"
	if !strings.HasSuffix(first, suffix) {
		return out, fmt.Errorf("invalid siwe message: bad header")
	}
	out.Domain = strings.TrimSpace(strings.TrimSuffix(first, suffix))
	out.Address = strings.ToLower(strings.TrimSpace(lines[1]))
	if out.Domain == "" || out.Address == "" {
		return out, fmt.Errorf("invalid siwe message: missing domain/address")
	}

	// SIWE（EIP-4361）文本格式是“头 + address + 可选 statement + 一组 key:value 字段”。
	// statement 可能为空/多行，因此我们不按固定行号解析，而是用第一个字段（URI:）作为分界点更稳健。
	fieldStart := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "URI:") {
			fieldStart = i
			break
		}
	}
	if fieldStart == -1 {
		return out, fmt.Errorf("invalid siwe message: missing URI field")
	}

	// statement candidate range: lines[2:fieldStart)
	if fieldStart > 2 {
		stmtLines := lines[2:fieldStart]
		// trim leading/trailing empty lines
		for len(stmtLines) > 0 && strings.TrimSpace(stmtLines[0]) == "" {
			stmtLines = stmtLines[1:]
		}
		for len(stmtLines) > 0 && strings.TrimSpace(stmtLines[len(stmtLines)-1]) == "" {
			stmtLines = stmtLines[:len(stmtLines)-1]
		}
		out.Statement = strings.TrimSpace(strings.Join(stmtLines, "\n"))
	}

	for _, l := range lines[fieldStart:] {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		switch {
		case strings.HasPrefix(l, "URI:"):
			out.URI = strings.TrimSpace(strings.TrimPrefix(l, "URI:"))
		case strings.HasPrefix(l, "Version:"):
			out.Version = strings.TrimSpace(strings.TrimPrefix(l, "Version:"))
		case strings.HasPrefix(l, "Chain ID:"):
			var cid int64
			_, err := fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(l, "Chain ID:")), "%d", &cid)
			if err != nil {
				return out, fmt.Errorf("invalid Chain ID: %w", err)
			}
			out.ChainID = cid
		case strings.HasPrefix(l, "Nonce:"):
			out.Nonce = strings.TrimSpace(strings.TrimPrefix(l, "Nonce:"))
		case strings.HasPrefix(l, "Issued At:"):
			ts := strings.TrimSpace(strings.TrimPrefix(l, "Issued At:"))
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return out, fmt.Errorf("invalid Issued At: %w", err)
			}
			out.IssuedAt = t
		}
	}

	// 这些字段是登录校验必需的最小集合；缺失时应拒绝，避免构造“弱约束”消息绕过后续校验逻辑。
	if out.URI == "" || out.Version == "" || out.Nonce == "" || out.ChainID == 0 || out.IssuedAt.IsZero() {
		return out, fmt.Errorf("invalid siwe message: missing required fields")
	}
	return out, nil
}

func splitLinesPreserveEmpty(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

// NewNonce 生成安全的随机 nonce（用于抵抗 SIWE 重放攻击）。
func NewNonce(nBytes int) (string, error) {
	if nBytes < 8 {
		// nonce 太短会降低抗重放能力；这里做下限保护，同时允许调用方传更长的 nonce。
		nBytes = 8
	}
	b := make([]byte, nBytes)
	// 使用 crypto/rand，确保 nonce 不可预测（用于 SIWE 防重放）。
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
