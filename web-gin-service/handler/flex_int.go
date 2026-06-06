package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexInt64 is an int64 that can unmarshal from both JSON numbers and JSON strings.
// This is needed because protojson always encodes int64/uint64 as JSON strings
// (per the protobuf JSON spec), but the frontend may send them back as either
// strings or numbers depending on whether the value was transformed before sending.
type FlexInt64 int64

func (f *FlexInt64) UnmarshalJSON(data []byte) error {
	// Try number first
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexInt64(n)
		return nil
	}
	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as int64: %w", s, err)
		}
		*f = FlexInt64(v)
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into int64", string(data))
}

// FlexUint64 is a uint64 that can unmarshal from both JSON numbers and JSON strings.
type FlexUint64 uint64

func (f *FlexUint64) UnmarshalJSON(data []byte) error {
	// Try number first
	var n uint64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexUint64(n)
		return nil
	}
	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as uint64: %w", s, err)
		}
		*f = FlexUint64(v)
		return nil
	}
	return fmt.Errorf("cannot unmarshal %s into uint64", string(data))
}
