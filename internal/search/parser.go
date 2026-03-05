// Package search provides query processing and search functionality for gosearch.
//
// It includes query parsing, boolean operations, fuzzy matching,
// phrase queries, and result ranking with caching.
package search

import (
	"strings"

	"github.com/abuiliazeed/gosearch/internal/indexer"
)

// Parser parses search queries into ParsedQuery structures.
type Parser struct {
	config    *Config
	tokenizer *indexer.Tokenizer
}

// NewParser creates a new query parser.
func NewParser(config *Config) *Parser {
	if config == nil {
		config = DefaultConfig()
	}

	return &Parser{
		config:    config,
		tokenizer: indexer.NewTokenizer(indexer.DefaultTokenizerConfig()),
	}
}

// Parse parses a search query string into a ParsedQuery.
// Supports:
// - Phrase queries: "exact match"
// - Boolean operators: AND (implicit), OR, NOT (-)
// - Fuzzy matching: term~
func (p *Parser) Parse(query string) *ParsedQuery {
	query = strings.TrimSpace(query)

	if query == "" {
		return &ParsedQuery{
			Type:     QueryTypeTerm,
			Terms:    []string{},
			Original: query,
		}
	}

	// Check for phrase query
	if p.config.PhraseEnabled && p.isPhraseQuery(query) {
		return p.parsePhraseQuery(query)
	}

	// Check for fuzzy query
	if p.config.FuzzyEnabled && p.isFuzzyQuery(query) {
		return p.parseFuzzyQuery(query)
	}

	// Parse as boolean query if enabled
	if p.config.BooleanEnabled && p.isBooleanQuery(query) {
		return p.parseBooleanQuery(query)
	}

	// Default: simple term query
	return p.parseTermQuery(query)
}

// isPhraseQuery checks if the query is a phrase query (enclosed in quotes).
func (p *Parser) isPhraseQuery(query string) bool {
	return len(query) >= 2 && query[0] == '"' && query[len(query)-1] == '"'
}

// isFuzzyQuery checks if the query contains a fuzzy operator (~).
func (p *Parser) isFuzzyQuery(query string) bool {
	return strings.Contains(query, "~")
}

// isBooleanQuery checks if the query contains boolean operators.
func (p *Parser) isBooleanQuery(query string) bool {
	return strings.Contains(strings.ToUpper(query), " OR ") ||
		strings.Contains(query, "-") ||
		strings.Contains(strings.ToUpper(query), " NOT ")
}

// parsePhraseQuery parses a phrase query.
func (p *Parser) parsePhraseQuery(query string) *ParsedQuery {
	// Remove quotes
	phrase := query[1 : len(query)-1]

	// Tokenize the phrase
	terms := p.tokenize(phrase)

	return &ParsedQuery{
		Type:     QueryTypePhrase,
		Terms:    terms,
		Phrase:   phrase,
		Original: query,
	}
}

// parseFuzzyQuery parses a fuzzy query.
func (p *Parser) parseFuzzyQuery(query string) *ParsedQuery {
	// Extract the fuzzy term (before ~)
	parts := strings.Split(query, "~")
	term := strings.TrimSpace(parts[0])

	return &ParsedQuery{
		Type:      QueryTypeFuzzy,
		Terms:     []string{term},
		FuzzyTerm: term,
		Original:  query,
	}
}

// parseBooleanQuery parses a boolean query with AND, OR, NOT operators.
func (p *Parser) parseBooleanQuery(query string) *ParsedQuery {
	booleanQuery := NewBooleanQuery()

	// Split by OR (highest precedence)
	orGroups := p.splitOr(query)

	// Process each OR group
	for _, group := range orGroups {
		group = strings.TrimSpace(group)

		// Split by NOT
		notParts := p.splitNot(group)

		// First part is the AND terms
		andTerms := p.tokenize(notParts[0])

		// Add remaining parts as NOT terms
		for i := 1; i < len(notParts); i++ {
			notTerm := strings.TrimSpace(notParts[i])
			if notTerm != "" {
				booleanQuery.AddNot(notTerm)
			}
		}

		// Add terms based on number of OR groups
		if len(orGroups) == 1 {
			// No OR, add to AND
			for _, term := range andTerms {
				booleanQuery.AddAnd(term)
			}
		} else {
			// Has OR, add to OR list
			for _, term := range andTerms {
				booleanQuery.AddOr(term)
			}
		}
	}

	// If no AND terms but we have OR terms, that's fine
	// If no OR terms but we have AND terms, that's also fine

	return &ParsedQuery{
		Type:     QueryTypeBoolean,
		Terms:    append(booleanQuery.AndTerms, booleanQuery.OrTerms...),
		Boolean:  booleanQuery,
		Original: query,
	}
}

// parseTermQuery parses a simple term query.
func (p *Parser) parseTermQuery(query string) *ParsedQuery {
	terms := p.tokenize(query)

	return &ParsedQuery{
		Type:     QueryTypeTerm,
		Terms:    terms,
		Original: query,
	}
}

// tokenize splits a query string into terms.
// Handles lowercase conversion and removes special characters.
// tokenize splits a query string into terms.
// Handles lowercase conversion and removes special characters using the indexer's tokenizer.
func (p *Parser) tokenize(query string) []string {
	tokens := p.tokenizer.Tokenize(query)

	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, token.Text)
	}

	return result
}

// splitOr splits a query by OR operators (case-insensitive).
func (p *Parser) splitOr(query string) []string {
	query = strings.TrimSpace(query)

	// Split by " OR " (case-insensitive)
	lowerQuery := strings.ToLower(query)
	var groups []string
	start := 0

	for {
		idx := strings.Index(lowerQuery[start:], " or ")
		if idx == -1 {
			groups = append(groups, query[start:])
			break
		}

		groups = append(groups, query[start:start+idx])
		start += idx + 4 // Skip " or "
	}

	return groups
}

// splitNot splits a query by NOT operators (case-insensitive or - prefix).
func (p *Parser) splitNot(query string) []string {
	query = strings.TrimSpace(query)

	parts := make([]string, 0)
	current := strings.Builder{}

	i := 0
	for i < len(query) {
		// Check for " NOT " (case-insensitive)
		if i <= len(query)-5 && strings.ToLower(query[i:i+5]) == " not " {
			parts = append(parts, current.String())
			current.Reset()
			i += 5
			continue
		}

		// Check for "-" prefix at word boundary
		if query[i] == '-' && (i == 0 || query[i-1] == ' ') {
			// Save current term before the -
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}

			// Start new term after the -
			i++
			continue
		}

		current.WriteByte(query[i])
		i++
	}

	// Add last part
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// Normalize normalizes a query string by removing extra whitespace and converting to lowercase.
func (p *Parser) Normalize(query string) string {
	// Trim whitespace
	query = strings.TrimSpace(query)

	// Replace multiple spaces with single space
	words := strings.Fields(query)
	normalized := strings.Join(words, " ")

	return strings.ToLower(normalized)
}

// ExtractTerms extracts all search terms from a parsed query.
func (p *Parser) ExtractTerms(parsed *ParsedQuery) []string {
	switch parsed.Type {
	case QueryTypeTerm, QueryTypeFuzzy:
		return parsed.Terms
	case QueryTypePhrase:
		return parsed.Terms
	case QueryTypeBoolean:
		if parsed.Boolean != nil {
			terms := make([]string, 0, len(parsed.Boolean.AndTerms)+len(parsed.Boolean.OrTerms))
			terms = append(terms, parsed.Boolean.AndTerms...)
			terms = append(terms, parsed.Boolean.OrTerms...)
			return terms
		}
		return []string{}
	default:
		return []string{}
	}
}

// GetQueryType returns the type of query without full parsing.
func (p *Parser) GetQueryType(query string) QueryType {
	parsed := p.Parse(query)
	return parsed.Type
}
