// Package utils provides environment and configuration helpers.
package utils

import (
	"os"
	"strings"
)

// GetEnv returns the value of the environment variable or fallback.
func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// SplitAndTrim splits a comma-separated string and trims whitespace from each element
func SplitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
