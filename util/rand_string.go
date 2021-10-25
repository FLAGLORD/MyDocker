package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const characterTable = "0123456789qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"

// RandString generates a n-length string
func RandString(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("length should be greater than zero")
	}

	rand.Seed(time.Now().UnixMicro())
	var builder strings.Builder
	builder.Grow(n)
	for i := 0; i < n; i++ {
		builder.WriteByte(characterTable[rand.Intn(len(characterTable))])
	}
	return builder.String(), nil
}
