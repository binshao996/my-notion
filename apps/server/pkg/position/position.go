package position

import (
	"strings"
)

const digits = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func Between(a, b string) string {
	if a == "" {
		a = "0"
	}
	if b == "" {
		return increment(a)
	}
	if a >= b {
		return increment(a)
	}

	// Find a string lexicographically between a and b
	result := make([]byte, 0, max(len(a), len(b))+1)
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		result = append(result, a[i])
		i++
	}

	if i < len(a) {
		result = append(result, a[i])
		for j := i + 1; j < len(a); j++ {
			if a[j] != 'Z' {
				result = append(result, a[j]+1)
				return string(result)
			}
		}
	}

	// Need a new digit between a[i] and b[i] or a[i] and 'Z'
	nextChar := byte('m')
	if i < len(a) && i < len(b) {
		mid := (charIndex(a[i]) + charIndex(b[i])) / 2
		nextChar = digits[mid]
	} else if i < len(b) {
		mid := charIndex(b[i]) / 2
		nextChar = digits[mid]
	}

	result = append(result, nextChar)
	return string(result)
}

func increment(s string) string {
	if s == "" {
		return "a0"
	}
	lastIdx := len(s) - 1
	lastChar := s[lastIdx]
	idx := charIndex(lastChar)
	if idx < len(digits)-1 {
		return s[:lastIdx] + string(digits[idx+1])
	}
	return s + "0"
}

func GenerateN(n int) []string {
	positions := make([]string, n)
	pos := "a0"
	for i := 0; i < n; i++ {
		positions[i] = pos
		pos = increment(pos)
	}
	return positions
}

func charIndex(c byte) int {
	return strings.IndexByte(digits, c)
}
