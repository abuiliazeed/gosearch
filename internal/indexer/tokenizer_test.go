package indexer

import (
	"reflect"
	"testing"
)

func TestTokenizer_Tokenize(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Simple sentence",
			text:     "Hello World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "With punctuation",
			text:     "Hello, World!",
			expected: []string{"hello", "world"},
		},
		{
			name:     "Multiple spaces",
			text:     "Hello   World",
			expected: []string{"hello", "world"},
		},
		{
			name:     "Empty string",
			text:     "",
			expected: []string{},
		},
		{
			name:     "Stop words (if default enabled)",
			text:     "the quick brown fox",
			expected: []string{"quick", "brown", "fox"},
		},
	}

	cfg := DefaultTokenizerConfig()
	tokenizer := NewTokenizer(cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := tokenizer.Tokenize(tt.text)

			var got []string
			for _, token := range tokens {
				got = append(got, token.Text)
			}

			// Handle empty slice vs nil slice
			if len(got) == 0 && len(tt.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Tokenize input %q\n got:  %v\n want: %v", tt.text, got, tt.expected)
			}
		})
	}
}

func TestTokenizer_Normalization(t *testing.T) {
	cfg := DefaultTokenizerConfig()
	tokenizer := NewTokenizer(cfg)

	text := "GOLANG"
	expected := []string{"golang"}

	tokens := tokenizer.Tokenize(text)
	var got []string
	for _, token := range tokens {
		got = append(got, token.Text)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Tokenize input %q\n got:  %v\n want: %v", text, got, expected)
	}
}
