package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func NormalizeContent(content string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
}

func ContentHash(content string) string {
	normalized := NormalizeContent(content)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
