package plugins

import (
	"bytes"
	"strings"
	"testing"
)

// Benchmark old implementation (bytes conversion)
func splitAndTrimOld(s, sep string) []string {
	var result []string
	for _, part := range bytes.Split([]byte(s), []byte(sep)) {
		result = append(result, string(bytes.TrimSpace(part)))
	}
	return result
}

// Benchmark new implementation (strings package)
func splitAndTrimNew(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func BenchmarkSplitAndTrim_Old(b *testing.B) {
	input := "text/html, application/json  ,  text/plain,application/xml"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = splitAndTrimOld(input, ",")
	}
}

func BenchmarkSplitAndTrim_New(b *testing.B) {
	input := "text/html, application/json  ,  text/plain,application/xml"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = splitAndTrimNew(input, ",")
	}
}

// Benchmark old matchesContentType
func matchesContentTypeOld(ct string, allowed []string) bool {
	for _, a := range allowed {
		if len(ct) >= len(a) && ct[:len(a)] == a {
			return true
		}
	}
	return false
}

// Benchmark new matchesContentType
func matchesContentTypeNew(ct string, allowed []string) bool {
	for _, a := range allowed {
		if strings.HasPrefix(ct, a) {
			return true
		}
	}
	return false
}

func BenchmarkMatchesContentType_Old(b *testing.B) {
	ct := "text/html; charset=utf-8"
	allowed := []string{"text/html", "text/plain", "application/json"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matchesContentTypeOld(ct, allowed)
	}
}

func BenchmarkMatchesContentType_New(b *testing.B) {
	ct := "text/html; charset=utf-8"
	allowed := []string{"text/html", "text/plain", "application/json"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matchesContentTypeNew(ct, allowed)
	}
}

// Comprehensive functionality tests
func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		sep      string
		expected []string
	}{
		{
			name:     "normal case",
			input:    "a, b, c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with extra spaces",
			input:    "text/html  ,  application/json  ,  text/plain",
			sep:      ",",
			expected: []string{"text/html", "application/json", "text/plain"},
		},
		{
			name:     "empty strings filtered",
			input:    "a,,b,  ,c",
			sep:      ",",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "single item",
			input:    "text/html",
			sep:      ",",
			expected: []string{"text/html"},
		},
		{
			name:     "empty input",
			input:    "",
			sep:      ",",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Expected %q at index %d, got %q", tt.expected[i], i, result[i])
				}
			}
		})
	}
}

func TestMatchesContentType(t *testing.T) {
	tests := []struct {
		name     string
		ct       string
		allowed  []string
		expected bool
	}{
		{
			name:     "exact match",
			ct:       "text/html",
			allowed:  []string{"text/html", "text/plain"},
			expected: true,
		},
		{
			name:     "prefix match with charset",
			ct:       "text/html; charset=utf-8",
			allowed:  []string{"text/html"},
			expected: true,
		},
		{
			name:     "no match",
			ct:       "image/png",
			allowed:  []string{"text/html", "application/json"},
			expected: false,
		},
		{
			name:     "empty content type",
			ct:       "",
			allowed:  []string{"text/html"},
			expected: false,
		},
		{
			name:     "empty allowed list",
			ct:       "text/html",
			allowed:  []string{},
			expected: false,
		},
		{
			name:     "case sensitive",
			ct:       "TEXT/HTML",
			allowed:  []string{"text/html"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesContentType(tt.ct, tt.allowed)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
