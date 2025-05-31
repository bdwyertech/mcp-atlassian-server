// Package utils provides environment and configuration helpers.
package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
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

// ParseJiraTime tries multiple layouts to parse a time string for Jira worklog
func ParseJiraTime(input string) (string, error) {
	jiraFormat := "2006-01-02T15:04:05.000-0700"
	layouts := []string{
		jiraFormat,
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.Parse(layout, input)
		if err == nil {
			return t.Format(jiraFormat), nil
		}
	}
	return "", fmt.Errorf("could not parse time: %s", input)
}
