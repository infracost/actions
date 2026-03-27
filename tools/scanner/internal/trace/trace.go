package trace

import (
	"crypto/rand"
)

var (
	ID string
)

func init() {
	ID = generateTraceID()
}

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func generateTraceID() string {
	return "infracost-ci-" + randomString(8) + "-" + randomString(8)
}

func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// Fallback (extremely rare)
		return "00000000"
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
