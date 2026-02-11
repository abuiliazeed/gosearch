// Package search provides query processing and search functionality for gosearch.
//
// It includes query parsing, boolean operations, fuzzy matching,
// phrase queries, and result ranking with caching.
package search

import ()

// LevenshteinDistance computes the Levenshtein distance between two strings.
// The Levenshtein distance is the minimum number of single-character edits
// (insertions, deletions, or substitutions) required to change one string into the other.
//
// Uses the Wagner-Fisher algorithm with dynamic programming.
// Time complexity: O(m*n) where m and n are the lengths of the strings.
// Space complexity: O(min(m,n)) - optimized to use only two rows.
func LevenshteinDistance(a, b string) int {
	// Ensure a is the shorter string for space optimization
	if len(a) > len(b) {
		a, b = b, a
	}

	// Empty string cases
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use rune slice for proper Unicode handling
	aRunes := []rune(a)
	bRunes := []rune(b)

	lenA := len(aRunes)
	lenB := len(bRunes)

	// Previous row of distances
	prevRow := make([]int, lenA+1)

	// Initialize the first row
	for i := 0; i <= lenA; i++ {
		prevRow[i] = i
	}

	// Fill in the rest of the matrix
	for j := 1; j <= lenB; j++ {
		// Current row starts with the distance for deleting all characters up to j
		currRow := []int{j}

		for i := 1; i <= lenA; i++ {
			cost := 0
			if aRunes[i-1] != bRunes[j-1] {
				cost = 1
			}

			// Minimum of three operations:
			// 1. Deletion (from prevRow[i])
			// 2. Insertion (from currRow[i-1])
			// 3. Substitution (from prevRow[i-1] + cost)
			minDist := prevRow[i] + 1 // Deletion
			if ins := currRow[i-1] + 1; ins < minDist {
				minDist = ins // Insertion
			}
			if sub := prevRow[i-1] + cost; sub < minDist {
				minDist = sub // Substitution
			}

			currRow = append(currRow, minDist)
		}

		prevRow = currRow
	}

	return prevRow[lenA]
}

// DamerauLevenshteinDistance computes the Damerau-Levenshtein distance between two strings.
// This is like Levenshtein distance but also allows transposition of adjacent characters.
// For example, "ab" vs "ba" has distance 1 (transposition) instead of 2 (substitution + substitution).
//
// This implementation uses a simplified version that doesn't handle repeated characters optimally.
func DamerauLevenshteinDistance(a, b string) int {
	// Ensure a is the shorter string
	if len(a) > len(b) {
		a, b = b, a
	}

	// Empty string cases
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	aRunes := []rune(a)
	bRunes := []rune(b)

	lenA := len(aRunes)
	lenB := len(bRunes)

	// Create the distance matrix
	d := make([][]int, lenA+1)
	for i := range d {
		d[i] = make([]int, lenB+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			cost := 0
			if aRunes[i-1] != bRunes[j-1] {
				cost = 1
			}

			d[i][j] = minInt(
				d[i-1][j]+1,      // Deletion
				d[i][j-1]+1,      // Insertion
				d[i-1][j-1]+cost, // Substitution
			)

			// Transposition
			if i > 1 && j > 1 && aRunes[i-1] == bRunes[j-2] && aRunes[i-2] == bRunes[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}

	return d[lenA][lenB]
}

// Similarity computes a similarity score between two strings based on Levenshtein distance.
// Returns a value in [0, 1] where 1 means identical and 0 means completely different.
// The formula is: 1 - (distance / max(len(a), len(b)))
func Similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1.0
	}

	distance := LevenshteinDistance(a, b)
	similarity := 1.0 - (float64(distance) / float64(maxLen))

	if similarity < 0 {
		return 0
	}

	return similarity
}

// JaroSimilarity computes the Jaro similarity between two strings.
// The Jaro similarity is a measure of similarity between two strings.
// Returns a value in [0, 1] where 1 means identical.
func JaroSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}

	aRunes := []rune(a)
	bRunes := []rune(b)

	lenA := len(aRunes)
	lenB := len(bRunes)

	if lenA == 0 || lenB == 0 {
		return 0
	}

	// Match distance
	matchDistance := max(lenA, lenB)/2 - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	// Find matches
	aMatches := make([]bool, lenA)
	bMatches := make([]bool, lenB)
	matches := 0
	transpositions := 0

	for i := 0; i < lenA; i++ {
		start := max(0, i-matchDistance)
		end := min(lenB, i+matchDistance+1)

		for j := start; j < end; j++ {
			if bMatches[j] || aRunes[i] != bRunes[j] {
				continue
			}

			aMatches[i] = true
			bMatches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0
	}

	// Count transpositions
	k := 0
	for i := 0; i < lenA; i++ {
		if !aMatches[i] {
			continue
		}

		for !bMatches[k] {
			k++
		}

		if aRunes[i] != bRunes[k] {
			transpositions++
		}

		k++
	}

	// Compute Jaro similarity
	return (float64(matches)/float64(lenA) +
		float64(matches)/float64(lenB) +
		float64(matches-transpositions/2)/float64(matches)) / 3.0
}

// JaroWinklerSimilarity computes the Jaro-Winkler similarity between two strings.
// This is an extension of Jaro similarity that gives more weight to prefix matches.
// The prefix scale factor is typically 0.1 and the prefix length is limited to 4 characters.
func JaroWinklerSimilarity(a, b string) float64 {
	jaro := JaroSimilarity(a, b)

	if jaro < 0.7 {
		return jaro
	}

	aRunes := []rune(a)
	bRunes := []rune(b)

	// Find prefix length
	prefix := 0
	maxPrefix := min(min(len(aRunes), len(bRunes)), 4)

	for prefix < maxPrefix && aRunes[prefix] == bRunes[prefix] {
		prefix++
	}

	// Jaro-Winkler similarity
	return jaro + float64(prefix)*0.1*(1.0-jaro)
}

// FindSimilarTerms finds terms in the vocabulary that are similar to the query term.
// Returns up to n terms that have similarity above the threshold.
// Terms are sorted by similarity in descending order.
func FindSimilarTerms(query string, vocabulary []string, threshold float64, n int) []SimilarTerm {
	similar := make([]SimilarTerm, 0)

	for _, term := range vocabulary {
		sim := Similarity(query, term)
		if sim >= threshold {
			similar = append(similar, SimilarTerm{
				Term:       term,
				Similarity: sim,
				Distance:   LevenshteinDistance(query, term),
			})
		}
	}

	// Sort by similarity (descending)
	// Simple bubble sort
	for i := 0; i < len(similar)-1; i++ {
		for j := i + 1; j < len(similar); j++ {
			if similar[i].Similarity < similar[j].Similarity {
				similar[i], similar[j] = similar[j], similar[i]
			}
		}
	}

	// Return top n
	if n > len(similar) {
		n = len(similar)
	}
	return similar[:n]
}

// SimilarTerm represents a term with its similarity score.
type SimilarTerm struct {
	Term       string
	Similarity float64
	Distance   int
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b, c int) int {
	minVal := a
	if b < minVal {
		minVal = b
	}
	if c < minVal {
		minVal = c
	}
	return minVal
}
