package core

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

func newID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%s-%s", prefix, time.Now().Format("20060102-150405"), strings.TrimRight(base64.RawURLEncoding.EncodeToString(b), "="))
}

func NewToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func TaskTitle(intent string) string {
	s := strings.TrimSpace(strings.ReplaceAll(intent, "\n", " "))
	if len(s) > 72 {
		s = s[:72] + "…"
	}
	if s == "" {
		return "Untitled task"
	}
	return s
}
