package logger

// GormLogConfig controls GORM SQL logging behavior.
type GormLogConfig struct {
	EnableSQLLog              bool `yaml:"enable_sql_log"`
	SlowThresholdMS           int  `yaml:"slow_threshold_ms"`
	IgnoreRecordNotFoundError bool `yaml:"ignore_record_not_found"`
}

// LogConfig controls logger initialization.
type LogConfig struct {
	Level         string        `yaml:"level"`
	Format        string        `yaml:"format"`
	EnableFile    bool          `yaml:"enable_file"`
	FilePath      string        `yaml:"file_path"`
	MaxSizeMB     int           `yaml:"max_size_mb"`
	MaxBackups    int           `yaml:"max_backups"`
	MaxAgeDays    int           `yaml:"max_age_days"`
	Compress      bool          `yaml:"compress"`
	EnableConsole bool          `yaml:"enable_console"`
	Gorm          GormLogConfig `yaml:"gorm"`
}

// DefaultLogConfig returns sensible defaults.
func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:         "info",
		Format:        "json",
		EnableFile:    false,
		FilePath:      "logs/logic-grpc-service.log",
		MaxSizeMB:     100,
		MaxBackups:    3,
		MaxAgeDays:    28,
		Compress:      true,
		EnableConsole: true,
		Gorm: GormLogConfig{
			EnableSQLLog:              false,
			SlowThresholdMS:           200,
			IgnoreRecordNotFoundError: true,
		},
	}
}
