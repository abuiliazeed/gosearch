// Package search provides query processing and search functionality for gosearch.
package search

import (
	"math"
	"testing"
)

// TestLevenshteinDistance tests the Levenshtein distance function.
func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "Identical strings",
			a:        "hello",
			b:        "hello",
			expected: 0,
		},
		{
			name:     "Empty strings",
			a:        "",
			b:        "",
			expected: 0,
		},
		{
			name:     "First string empty",
			a:        "",
			b:        "hello",
			expected: 5,
		},
		{
			name:     "Second string empty",
			a:        "hello",
			b:        "",
			expected: 5,
		},
		{
			name:     "Single character difference",
			a:        "cat",
			b:        "bat",
			expected: 1,
		},
		{
			name:     "Insertion",
			a:        "cat",
			b:        "cats",
			expected: 1,
		},
		{
			name:     "Deletion",
			a:        "cats",
			b:        "cat",
			expected: 1,
		},
		{
			name:     "Substitution",
			a:        "kitten",
			b:        "sitting",
			expected: 3,
		},
		{
			name:     "Completely different",
			a:        "abc",
			b:        "xyz",
			expected: 3,
		},
		{
			name:     "One character strings",
			a:        "a",
			b:        "b",
			expected: 1,
		},
		{
			name:     "Longer strings with one diff",
			a:        "algorithm",
			b:        "alogrithm", //nolint:misspell // Intentional typo for distance test
			expected: 2,
		},
		{
			name:     "Case sensitive",
			a:        "Hello",
			b:        "hello",
			expected: 1,
		},
		{
			name:     "Unicode characters",
			a:        "café",
			b:        "cafe",
			expected: 1,
		},
		{
			name:     "Transposition counts as 2",
			a:        "ab",
			b:        "ba",
			expected: 2,
		},
		{
			name:     "Length optimization test a > b",
			a:        "longerstring",
			b:        "short",
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

// TestDamerauLevenshteinDistance tests the Damerau-Levenshtein distance function.
func TestDamerauLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "Identical strings",
			a:        "hello",
			b:        "hello",
			expected: 0,
		},
		{
			name:     "Empty strings",
			a:        "",
			b:        "",
			expected: 0,
		},
		{
			name:     "First string empty",
			a:        "",
			b:        "hello",
			expected: 5,
		},
		{
			name:     "Second string empty",
			a:        "hello",
			b:        "",
			expected: 5,
		},
		{
			name:     "Single character difference",
			a:        "cat",
			b:        "bat",
			expected: 1,
		},
		{
			name:     "Insertion",
			a:        "cat",
			b:        "cats",
			expected: 1,
		},
		{
			name:     "Substitution",
			a:        "kitten",
			b:        "sitting",
			expected: 3,
		},
		{
			name:     "Transposition of adjacent characters",
			a:        "ab",
			b:        "ba",
			expected: 1, // Unlike Levenshtein which returns 2
		},
		{
			name:     "Transposition in longer string",
			a:        "ca",
			b:        "ac",
			expected: 1,
		},
		{
			name:     "Common typo example",
			a:        "recieve", //nolint:misspell // Intentional typo for distance test
			b:        "receive",
			expected: 1,
		},
		{
			name:     "Double transposition",
			a:        "abcd",
			b:        "badc",
			expected: 2,
		},
		{
			name:     "Transposition with other edits",
			a:        "algorithm",
			b:        "alogrithm", //nolint:misspell // Intentional typo for distance test
			expected: 1,           // "al" -> "la" is a transposition
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DamerauLevenshteinDistance(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("DamerauLevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

// TestSimilarity tests the similarity function.
func TestSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected float64
		delta    float64
	}{
		{
			name:     "Identical strings",
			a:        "hello",
			b:        "hello",
			expected: 1.0,
			delta:    0.0,
		},
		{
			name:     "Empty strings",
			a:        "",
			b:        "",
			expected: 1.0,
			delta:    0.0,
		},
		{
			name:     "One empty string",
			a:        "",
			b:        "hello",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "One character difference",
			a:        "cat",
			b:        "bat",
			expected: 2.0 / 3.0, // 1 - 1/3
			delta:    0.001,
		},
		{
			name:     "Half similar",
			a:        "test",
			b:        "tent",
			expected: 0.75, // 1 - 1/4
			delta:    0.001,
		},
		{
			name:     "Completely different",
			a:        "abc",
			b:        "xyz",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "One character match out of three",
			a:        "cat",
			b:        "cut",
			expected: 2.0 / 3.0,
			delta:    0.001,
		},
		{
			name:     "Short strings different",
			a:        "a",
			b:        "b",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "Similar longer strings",
			a:        "kitten",
			b:        "sitting",
			expected: 1.0 - 3.0/7.0, // 1 - 3/7
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Similarity(tt.a, tt.b)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("Similarity(%q, %q) = %f, want %f (delta %f)", tt.a, tt.b, got, tt.expected, tt.delta)
			}
		})
	}
}

// TestJaroSimilarity tests the Jaro similarity function.
func TestJaroSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected float64
		delta    float64
	}{
		{
			name:     "Identical strings",
			a:        "hello",
			b:        "hello",
			expected: 1.0,
			delta:    0.0,
		},
		{
			name:     "Empty strings",
			a:        "",
			b:        "",
			expected: 1.0, // Jaro returns 1.0 for identical empty strings
			delta:    0.0,
		},
		{
			name:     "One empty string",
			a:        "",
			b:        "hello",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "No matches",
			a:        "abcd",
			b:        "wxyz",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "MARTHA vs MARTHA",
			a:        "MARTHA",
			b:        "MARTHA",
			expected: 1.0,
			delta:    0.0,
		},
		{
			name:     "MARTHA vs MARHTA (transposition)",
			a:        "MARTHA",
			b:        "MARHTA",
			expected: 0.944, // Standard Jaro value
			delta:    0.01,
		},
		{
			name:     "DWAYNE vs DUANE",
			a:        "DWAYNE",
			b:        "DUANE",
			expected: 0.82, // Approximate standard Jaro value
			delta:    0.05,
		},
		{
			name:     "DIXON vs DICKSONX",
			a:        "DIXON",
			b:        "DICKSONX",
			expected: 0.767, // Approximate standard Jaro value
			delta:    0.05,
		},
		{
			name:     "Simple similar strings",
			a:        "test",
			b:        "text",
			expected: 0.867, // Approximate
			delta:    0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaroSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("JaroSimilarity(%q, %q) = %f, want %f (delta %f)", tt.a, tt.b, got, tt.expected, tt.delta)
			}
		})
	}
}

// TestJaroWinklerSimilarity tests the Jaro-Winkler similarity function.
func TestJaroWinklerSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected float64
		delta    float64
	}{
		{
			name:     "Identical strings",
			a:        "hello",
			b:        "hello",
			expected: 1.0,
			delta:    0.0,
		},
		{
			name:     "Empty strings",
			a:        "",
			b:        "",
			expected: 1.0, // Identical strings return 1.0
			delta:    0.0,
		},
		{
			name:     "MARTHA vs MARHTA",
			a:        "MARTHA",
			b:        "MARHTA",
			expected: 0.961, // Standard Jaro-Winkler value (higher than Jaro)
			delta:    0.01,
		},
		{
			name:     "DWAYNE vs DUANE",
			a:        "DWAYNE",
			b:        "DUANE",
			expected: 0.84, // Standard Jaro-Winkler value
			delta:    0.05,
		},
		{
			name:     "DIXON vs DICKSONX",
			a:        "DIXON",
			b:        "DICKSONX",
			expected: 0.813, // Standard Jaro-Winkler value
			delta:    0.05,
		},
		{
			name:     "Common prefix bonus",
			a:        "testing",
			b:        "test",
			expected: 0.911, // Approximate
			delta:    0.05,
		},
		{
			name:     "Low Jaro score gets no bonus",
			a:        "abc",
			b:        "xyz",
			expected: 0.0,
			delta:    0.0,
		},
		{
			name:     "Four character prefix",
			a:        "precise",
			b:        "precis",
			expected: 0.967, // Approximate
			delta:    0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaroWinklerSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("JaroWinklerSimilarity(%q, %q) = %f, want %f (delta %f)", tt.a, tt.b, got, tt.expected, tt.delta)
			}
		})
	}
}

// TestFindSimilarTerms tests the FindSimilarTerms function.
func TestFindSimilarTerms(t *testing.T) {
	vocabulary := []string{"apple", "application", "apply", "ape", "orange", "banana", "applet"}

	tests := []struct {
		name      string
		query     string
		vocab     []string
		threshold float64
		n         int
		wantCount int
		wantFirst string // First (highest similarity) term
	}{
		{
			name:      "Find similar to 'apple'",
			query:     "apple",
			vocab:     vocabulary,
			threshold: 0.5,
			n:         5,
			wantCount: 4, // apple, applet, apply, ape
			wantFirst: "apple",
		},
		{
			name:      "Find similar to 'appl'",
			query:     "appl",
			vocab:     vocabulary,
			threshold: 0.5,
			n:         5,
			wantCount: 4, // apple, application, apply, applet
			wantFirst: "apple",
		},
		{
			name:      "High threshold",
			query:     "apple",
			vocab:     vocabulary,
			threshold: 0.9,
			n:         5,
			wantCount: 1, // Only exact match (apple)
			wantFirst: "apple",
		},
		{
			name:      "Low threshold",
			query:     "apple",
			vocab:     vocabulary,
			threshold: 0.1,
			n:         10,
			wantCount: 7, // All terms are somewhat similar
			wantFirst: "apple",
		},
		{
			name:      "Limit results with n",
			query:     "appl",
			vocab:     vocabulary,
			threshold: 0.3,
			n:         2,
			wantCount: 2, // Limited by n
			wantFirst: "apple",
		},
		{
			name:      "No matches",
			query:     "xyz",
			vocab:     vocabulary,
			threshold: 0.5,
			n:         5,
			wantCount: 0,
			wantFirst: "",
		},
		{
			name:      "Empty vocabulary",
			query:     "apple",
			vocab:     []string{},
			threshold: 0.5,
			n:         5,
			wantCount: 0,
			wantFirst: "",
		},
		{
			name:      "Single character query",
			query:     "a",
			vocab:     vocabulary,
			threshold: 0.1,
			n:         5,
			wantCount: 5, // First 5 alphabetically or by similarity
			wantFirst: "ape",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSimilarTerms(tt.query, tt.vocab, tt.threshold, tt.n)

			if len(got) != tt.wantCount {
				t.Errorf("FindSimilarTerms(%q, vocab, %f, %d) returned %d terms, want %d", tt.query, tt.threshold, tt.n, len(got), tt.wantCount)
			}

			if len(got) > 0 {
				if got[0].Term != tt.wantFirst {
					t.Errorf("FindSimilarTerms(%q, vocab, %f, %d) first term = %q, want %q", tt.query, tt.threshold, tt.n, got[0].Term, tt.wantFirst)
				}

				// Verify results are sorted by similarity (descending)
				for i := 1; i < len(got); i++ {
					if got[i-1].Similarity < got[i].Similarity {
						t.Errorf("Results not sorted by similarity: got[%d-1].Similarity = %f, got[%d].Similarity = %f", i, got[i-1].Similarity, i, got[i].Similarity)
					}
				}

				// Verify all results meet threshold
				for _, term := range got {
					if term.Similarity < tt.threshold {
						t.Errorf("Term %q has similarity %f below threshold %f", term.Term, term.Similarity, tt.threshold)
					}
				}
			}
		})
	}
}

// TestSimilarTerm tests the SimilarTerm struct fields.
func TestSimilarTerm(t *testing.T) {
	vocabulary := []string{"test", "tent", "best", "rest"}
	results := FindSimilarTerms("test", vocabulary, 0.0, 10)

	if len(results) == 0 {
		t.Fatal("FindSimilarTerms returned no results")
	}

	// Check that the first result is the exact match
	first := results[0]
	if first.Term != "test" {
		t.Errorf("First term should be 'test', got %q", first.Term)
	}
	if first.Similarity != 1.0 {
		t.Errorf("Exact match should have similarity 1.0, got %f", first.Similarity)
	}
	if first.Distance != 0 {
		t.Errorf("Exact match should have distance 0, got %d", first.Distance)
	}

	// Check that a similar term has reasonable values
	if len(results) > 1 {
		second := results[1]
		if second.Term != "tent" && second.Term != "rest" && second.Term != "best" {
			t.Logf("Second term is %q (distance: %d, similarity: %f)", second.Term, second.Distance, second.Similarity)
		}
	}
}

// TestFuzzEdgeCases tests edge cases for fuzzy matching functions.
func TestFuzzEdgeCases(t *testing.T) {
	t.Run("Unicode emoji", func(t *testing.T) {
		dist := LevenshteinDistance("😀", "😁")
		if dist < 0 {
			t.Errorf("Negative distance for emoji: %d", dist)
		}
	})

	t.Run("Very long strings", func(t *testing.T) {
		longA := string(make([]byte, 1000))
		longB := string(make([]byte, 1000))
		dist := LevenshteinDistance(longA, longB)
		if dist != 0 {
			t.Errorf("Identical long strings should have distance 0, got %d", dist)
		}
	})

	t.Run("Strings with special characters", func(t *testing.T) {
		dist := LevenshteinDistance("hello!", "hello?")
		if dist != 1 {
			t.Errorf("Expected distance 1 for 'hello!' vs 'hello?', got %d", dist)
		}
	})

	t.Run("Similarity is bounded", func(t *testing.T) {
		sim := Similarity("", "hello")
		if sim < 0 || sim > 1 {
			t.Errorf("Similarity out of bounds [0,1]: %f", sim)
		}
	})
}

// Benchmark tests for performance measurement.
func BenchmarkLevenshteinDistance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LevenshteinDistance("kitten", "sitting")
	}
}

func BenchmarkDamerauLevenshteinDistance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DamerauLevenshteinDistance("kitten", "sitting")
	}
}

func BenchmarkJaroSimilarity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		JaroSimilarity("MARTHA", "MARHTA")
	}
}

func BenchmarkJaroWinklerSimilarity(b *testing.B) {
	for i := 0; i < b.N; i++ {
		JaroWinklerSimilarity("MARTHA", "MARHTA")
	}
}

func BenchmarkFindSimilarTerms(b *testing.B) {
	vocab := make([]string, 1000)
	for i := range vocab {
		vocab[i] = "term" + string(rune('0'+i%10))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FindSimilarTerms("term", vocab, 0.5, 10)
	}
}

// TestHelperFunctions tests the internal helper functions.
func TestHelperFunctions(t *testing.T) {
	t.Run("min function", func(t *testing.T) {
		if min(5, 3) != 3 {
			t.Error("min(5, 3) should be 3")
		}
		if min(1, 10) != 1 {
			t.Error("min(1, 10) should be 1")
		}
		if min(5, 5) != 5 {
			t.Error("min(5, 5) should be 5")
		}
	})

	t.Run("max function", func(t *testing.T) {
		if max(5, 3) != 5 {
			t.Error("max(5, 3) should be 5")
		}
		if max(1, 10) != 10 {
			t.Error("max(1, 10) should be 10")
		}
		if max(5, 5) != 5 {
			t.Error("max(5, 5) should be 5")
		}
	})

	t.Run("minInt function", func(t *testing.T) {
		if minInt(5, 3, 7) != 3 {
			t.Error("minInt(5, 3, 7) should be 3")
		}
		if minInt(10, 20, 5) != 5 {
			t.Error("minInt(10, 20, 5) should be 5")
		}
		if minInt(1, 1, 1) != 1 {
			t.Error("minInt(1, 1, 1) should be 1")
		}
	})
}

// TestFuzzyDistanceProperties tests mathematical properties of distance functions.
func TestFuzzyDistanceProperties(t *testing.T) {
	t.Run("Levenshtein symmetry", func(t *testing.T) {
		pairs := [][2]string{
			{"hello", "world"},
			{"test", "tent"},
			{"kitten", "sitting"},
		}
		for _, pair := range pairs {
			d1 := LevenshteinDistance(pair[0], pair[1])
			d2 := LevenshteinDistance(pair[1], pair[0])
			if d1 != d2 {
				t.Errorf("LevenshteinDistance not symmetric: d(%q,%q)=%d, d(%q,%q)=%d", pair[0], pair[1], d1, pair[1], pair[0], d2)
			}
		}
	})

	t.Run("Levenshtein triangle inequality", func(t *testing.T) {
		a, b, c := "test", "tent", "rent"
		dab := LevenshteinDistance(a, b)
		dbc := LevenshteinDistance(b, c)
		dac := LevenshteinDistance(a, c)
		if dac > dab+dbc {
			t.Errorf("Triangle inequality violated: d(a,c)=%d > d(a,b)+d(b,c)=%d+%d=%d", dac, dab, dbc, dab+dbc)
		}
	})

	t.Run("Levenshtein non-negativity", func(t *testing.T) {
		pairs := [][2]string{
			{"", ""},
			{"a", "b"},
			{"hello", "hello"},
			{"longstring", "anotherlong"},
		}
		for _, pair := range pairs {
			d := LevenshteinDistance(pair[0], pair[1])
			if d < 0 {
				t.Errorf("LevenshteinDistance negative: d(%q,%q)=%d", pair[0], pair[1], d)
			}
		}
	})

	t.Run("Levenshtein identity", func(t *testing.T) {
		strings := []string{"", "a", "hello", "test123"}
		for _, s := range strings {
			d := LevenshteinDistance(s, s)
			if d != 0 {
				t.Errorf("LevenshteinDistance(%q,%q) should be 0, got %d", s, s, d)
			}
		}
	})
}

// TestFindSimilarTermsSorting tests that results are properly sorted.
func TestFindSimilarTermsSorting(t *testing.T) {
	vocabulary := []string{"aaaa", "aaab", "aabb", "abbb", "bbbb"}
	results := FindSimilarTerms("aaaa", vocabulary, 0.0, 10)

	// Check if results are sorted by similarity (descending)
	for i := 1; i < len(results); i++ {
		if results[i-1].Similarity < results[i].Similarity {
			t.Errorf("Results not sorted by similarity descending: [%d]=%f, [%d]=%f",
				i-1, results[i-1].Similarity, i, results[i].Similarity)
		}
	}
}

// TestSimilarityRange tests that similarity always returns values in [0, 1].
func TestSimilarityRange(t *testing.T) {
	pairs := [][2]string{
		{"", ""},
		{"", "a"},
		{"a", "a"},
		{"a", "b"},
		{"same", "same"},
		{"completely", "different"},
		{"café", "cafe"},
		{"hello world", "hello world!"},
		{"a", "abcdefghijklmnopqrstuvwxyz"},
	}

	for _, pair := range pairs {
		sim := Similarity(pair[0], pair[1])
		if sim < 0 || sim > 1 {
			t.Errorf("Similarity(%q, %q) = %f, out of range [0, 1]", pair[0], pair[1], sim)
		}
	}
}

// TestJaroWinklerPrefixBonus tests that prefix matching increases Jaro-Winkler score.
func TestJaroWinklerPrefixBonus(t *testing.T) {
	a, b := "testing", "test"

	jaro := JaroSimilarity(a, b)
	jaroWinkler := JaroWinklerSimilarity(a, b)

	// Jaro-Winkler should be >= Jaro for strings with common prefix
	if jaroWinkler < jaro {
		t.Errorf("JaroWinklerSimilarity(%q, %q) = %f < JaroSimilarity = %f", a, b, jaroWinkler, jaro)
	}
}

// TestEmptyStringHandling tests behavior with empty strings.
func TestEmptyStringHandling(t *testing.T) {
	t.Run("Levenshtein with empty", func(t *testing.T) {
		if LevenshteinDistance("", "") != 0 {
			t.Error("LevenshteinDistance('', '') should be 0")
		}
		if LevenshteinDistance("", "abc") != 3 {
			t.Error("LevenshteinDistance('', 'abc') should be 3")
		}
		if LevenshteinDistance("abc", "") != 3 {
			t.Error("LevenshteinDistance('abc', '') should be 3")
		}
	})

	t.Run("Similarity with empty", func(t *testing.T) {
		if Similarity("", "") != 1.0 {
			t.Error("Similarity('', '') should be 1.0")
		}
		if Similarity("", "abc") != 0.0 {
			t.Error("Similarity('', 'abc') should be 0.0")
		}
	})

	t.Run("Jaro with empty", func(t *testing.T) {
		if JaroSimilarity("", "") != 1.0 {
			t.Error("JaroSimilarity('', '') should be 1.0 for identical strings")
		}
		if JaroSimilarity("", "abc") != 0 {
			t.Error("JaroSimilarity('', 'abc') should be 0")
		}
	})
}

// TestFindSimilarTermStructFields validates the SimilarTerm struct.
func TestFindSimilarTermStructFields(t *testing.T) {
	vocab := []string{"test", "tent", "text"}
	results := FindSimilarTerms("test", vocab, 0.0, 10)

	for i, r := range results {
		if r.Term == "" {
			t.Errorf("Result %d has empty Term field", i)
		}
		if r.Similarity < 0 || r.Similarity > 1 {
			t.Errorf("Result %d has invalid Similarity: %f", i, r.Similarity)
		}
		if r.Distance < 0 {
			t.Errorf("Result %d has negative Distance: %d", i, r.Distance)
		}
	}
}

// TestSortingStability tests that sorting is deterministic.
func TestSortingStability(t *testing.T) {
	vocab := []string{"test", "tent", "text", "tint"}
	results1 := FindSimilarTerms("test", vocab, 0.0, 10)
	results2 := FindSimilarTerms("test", vocab, 0.0, 10)

	if len(results1) != len(results2) {
		t.Fatalf("Different result counts: %d vs %d", len(results1), len(results2))
	}

	for i := range results1 {
		if results1[i].Term != results2[i].Term {
			t.Errorf("Results not stable at index %d: %q vs %q", i, results1[i].Term, results2[i].Term)
		}
	}
}
