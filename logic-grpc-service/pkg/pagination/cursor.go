package pagination

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// EncodeCursor creates an opaque cursor from a timestamp and id.
// Format: base64(unix_nano,id) e.g. "MTcwMDAwMDAwMDAwMDAwMDAwMCwxMjM="
func EncodeCursor(t time.Time, id int64) string {
	raw := fmt.Sprintf("%d,%d", t.UnixNano(), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor parses a cursor back into timestamp and id.
// Returns zero values if the cursor is empty or malformed.
func DecodeCursor(cursor string) (time.Time, int64, error) {
	if cursor == "" {
		return time.Time{}, 0, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(b), ",", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid cursor format")
	}
	ns, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid cursor id: %w", err)
	}
	return time.Unix(0, ns), id, nil
}
