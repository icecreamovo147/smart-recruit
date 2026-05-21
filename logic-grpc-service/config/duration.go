package config

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Tag == "!!int" {
			seconds, err := strconv.ParseInt(value.Value, 10, 64)
			if err != nil {
				return err
			}
			d.Duration = time.Duration(seconds) * time.Second
			return nil
		}
		parsed, err := time.ParseDuration(value.Value)
		if err != nil {
			return err
		}
		d.Duration = parsed
		return nil
	default:
		return fmt.Errorf("invalid duration node kind %d", value.Kind)
	}
}
