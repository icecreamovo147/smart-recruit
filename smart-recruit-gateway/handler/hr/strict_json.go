package hr

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
)

// bindStrictJSON prevents removed or misspelled public request fields from
// being silently ignored at the Gateway boundary.
func bindStrictJSON(c *gin.Context, target any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain a single JSON object")
		}
		return err
	}
	return nil
}
