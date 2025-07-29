package utils

import (
	"regexp"
	"testing"
)

// Simple regex benchmark
func BenchmarkRegexMatchString(b *testing.B) {
	re := regexp.MustCompile(`^[A-Za-z0-9_\\]*$`)
	testString := "simple_test_123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = re.MatchString(testString)
	}
}

// Optimized character check benchmark
func BenchmarkOptimizedCharCheck(b *testing.B) {
	testString := "simple_test_123"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isPathChar(testString)
	}
}

// Benchmark ConvertComponentIdIntoFriendlyPathSearch with various inputs
func BenchmarkConvertComponentPath_Simple(b *testing.B) {
	path := "#/components/schemas/Pet"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertComponentIdIntoFriendlyPathSearch(path)
	}
}

func BenchmarkConvertComponentPath_Complex(b *testing.B) {
	path := "#/paths/~1v2~1customers~1my~1invoices~1%7Binvoice_uuid%7D/get/parameters/0"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertComponentIdIntoFriendlyPathSearch(path)
	}
}

func BenchmarkConvertComponentPath_VeryComplex(b *testing.B) {
	path := "#/paths/~1crazy~1ass~1references/get/responses/404/content/application~1xml;%20charset=utf-8/schema"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertComponentIdIntoFriendlyPathSearch(path)
	}
}