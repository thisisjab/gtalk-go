package envreader

import (
	"os"
	"strconv"
	"strings"
)

type EnvReader struct {
	prefix string
}

func New(prefix string) *EnvReader {
	return &EnvReader{
		prefix: prefix,
	}
}

func (r *EnvReader) String(key, fallback string) string {
	value := os.Getenv(r.prefix + key)

	if value == "" {
		return fallback
	}

	return value
}

func (r *EnvReader) Bool(key string, fallback bool) bool {
	value := r.String(key, "")

	if strings.EqualFold(value, "true") || strings.EqualFold(value, "1") {
		return true
	}

	return false
}

func (r *EnvReader) Int(key string, fallback int) int {
	value := r.String(key, "")

	if value == "" {
		return fallback
	}

	if i, err := strconv.Atoi(value); err == nil {
		return i
	}

	return fallback
}

func (r *EnvReader) Choice(key string, choices []string, fallback string) string {
	value := r.String(key, "")

	if value == "" {
		return fallback
	}

	for _, choice := range choices {
		if strings.EqualFold(value, choice) {
			return choice
		}
	}

	return fallback
}
