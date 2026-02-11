package indexer

import (
	"strings"
	"unicode"
)

// Tokenizer handles text tokenization with stopword removal.
//
// The Tokenizer splits text into tokens using Unicode-aware word boundary
// detection, removes stopwords, and filters tokens by minimum length.
type Tokenizer struct {
	stopwords   map[string]bool
	minTokenLen int
}

// NewTokenizer creates a new Tokenizer with the given configuration.
func NewTokenizer(config *TokenizerConfig) *Tokenizer {
	if config == nil {
		config = DefaultTokenizerConfig()
	}
	return &Tokenizer{
		stopwords:   config.Stopwords,
		minTokenLen: config.MinTokenLen,
	}
}

// Tokenize splits text into tokens with positions.
//
// Tokens are extracted using Unicode word boundaries, converted to lowercase,
// filtered by minimum length, and stopwords are removed. Each token is
// assigned a position indicating its order in the original text.
func (t *Tokenizer) Tokenize(text string) []Token {
	if text == "" {
		return nil
	}

	tokens := make([]Token, 0, strings.Count(text, " ")+1)
	position := 0

	// Use a scanner to tokenize by Unicode word boundaries
	scanner := newWordScanner(text)
	for scanner.Scan() {
		word := scanner.Text()

		// Normalize to lowercase
		word = strings.ToLower(word)

		// Skip if too short
		if len(word) < t.minTokenLen {
			continue
		}

		// Skip if stopword
		if t.stopwords[word] {
			continue
		}

		// Skip if not a valid word (only letters and numbers)
		if !t.isValidWord(word) {
			continue
		}

		tokens = append(tokens, Token{
			Text:     word,
			Position: position,
		})
		position++
	}

	return tokens
}

// isValidWord checks if a word contains valid characters.
// Valid words consist primarily of letters and numbers.
func (t *Tokenizer) isValidWord(word string) bool {
	if word == "" {
		return false
	}

	hasLetter := false
	for _, r := range word {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '-' {
			return false
		}
	}

	return hasLetter
}

// wordScanner scans text word by word using Unicode boundaries.
type wordScanner struct {
	text string
	pos  int
	word string
}

// newWordScanner creates a new word scanner for the given text.
func newWordScanner(text string) *wordScanner {
	return &wordScanner{text: text, pos: 0}
}

// Scan advances to the next word.
// Returns false when there are no more words.
func (s *wordScanner) Scan() bool {
	if s.pos >= len(s.text) {
		return false
	}

	// Skip non-word characters
	for s.pos < len(s.text) && !s.isWordChar(s.text[s.pos]) {
		s.pos++
	}

	if s.pos >= len(s.text) {
		return false
	}

	// Find the end of the word
	start := s.pos
	for s.pos < len(s.text) && s.isWordChar(s.text[s.pos]) {
		s.pos++
	}

	s.word = s.text[start:s.pos]
	return true
}

// Text returns the current word.
func (s *wordScanner) Text() string {
	return s.word
}

// isWordChar checks if a byte is a word character.
func (s *wordScanner) isWordChar(c byte) bool {
	// ASCII word characters (letters, digits, apostrophe, hyphen)
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '\'' || c == '-'
}

// TokenizeMultiple tokenizes multiple texts and returns combined tokens.
// The position is reset for each text input.
func (t *Tokenizer) TokenizeMultiple(texts []string) [][]Token {
	if len(texts) == 0 {
		return nil
	}

	results := make([][]Token, len(texts))
	for i, text := range texts {
		results[i] = t.Tokenize(text)
	}
	return results
}

// Normalize normalizes a token for indexing.
// This converts to lowercase and applies basic normalization rules.
func (t *Tokenizer) Normalize(token string) string {
	token = strings.ToLower(token)
	return token
}

// IsStopword returns true if the given word is a stopword.
func (t *Tokenizer) IsStopword(word string) bool {
	word = strings.ToLower(word)
	return t.stopwords[word]
}

// AddStopword adds a word to the stopword list.
func (t *Tokenizer) AddStopword(word string) {
	word = strings.ToLower(word)
	t.stopwords[word] = true
}

// RemoveStopword removes a word from the stopword list.
func (t *Tokenizer) RemoveStopword(word string) {
	word = strings.ToLower(word)
	delete(t.stopwords, word)
}

// SetStopwords replaces the entire stopword list.
func (t *Tokenizer) SetStopwords(words map[string]bool) {
	t.stopwords = words
}

// GetStopwords returns a copy of the stopword list.
func (t *Tokenizer) GetStopwords() map[string]bool {
	result := make(map[string]bool, len(t.stopwords))
	for word, enabled := range t.stopwords {
		result[word] = enabled
	}
	return result
}
