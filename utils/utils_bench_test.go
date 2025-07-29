package utils

import (
	"regexp"
	"testing"
)

// Local regex for benchmarking
var testPathCharExp = regexp.MustCompile(`^[A-Za-z0-9_\\]*$`)

// Benchmark the regex-based pathCharExp.MatchString
func BenchmarkPathCharExp_Regex(b *testing.B) {
	testCases := []string{
		"simple",
		"SimpleCase",
		"with_underscore",
		"with-dash",
		"with spaces",
		"special!char",
		"123numeric",
		"back\\slash",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = testPathCharExp.MatchString(tc)
		}
	}
}

func BenchmarkPathCharExp_Optimized(b *testing.B) {
	testCases := []string{
		"simple",
		"SimpleCase", 
		"with_underscore",
		"with-dash",
		"with spaces",
		"special!char",
		"123numeric",
		"back\\slash",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = isPathChar(tc)
		}
	}
}