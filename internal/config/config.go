package config

import (
	"os"
	"strconv"
	"time"
)

func GetEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func GetEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func GetEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func GetEnvDurationMs(key string, defMs int) time.Duration {
	return time.Duration(GetEnvInt(key, defMs)) * time.Millisecond
}

func GetEnvDurationSeconds(key string, defSec int) time.Duration {
	return time.Duration(GetEnvInt(key, defSec)) * time.Second
}
