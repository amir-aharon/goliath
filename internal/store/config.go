package store

import (
	"os"
	"strconv"
)

type MemoryConfig struct {
	SweepIntervalSec int
	SweepSampleSize  int
}

func LoadMemoryConfig() MemoryConfig {
	cfg := MemoryConfig{
		SweepIntervalSec: 60,
		SweepSampleSize:  20,
	}

	if v := os.Getenv("EXPIRED_SWEEP_INTERVAL"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			cfg.SweepIntervalSec = parsed
		}
	}

	if v := os.Getenv("SWEEP_SAMPLE_SIZE"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			cfg.SweepSampleSize = parsed
		}
	}

	return cfg
}
