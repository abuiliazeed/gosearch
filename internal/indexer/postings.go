package indexer

import (
	"sort"
)

// AddPosting adds a posting to the postings list.
// If a posting for the document already exists, the position is added
// and the term frequency is updated.
func (pl *PostingsList) AddPosting(docID string, position int) {
	// Check if posting for this docID already exists
	for i := range pl.Postings {
		if pl.Postings[i].DocID == docID {
			pl.Postings[i].Positions = append(pl.Postings[i].Positions, position)
			pl.Postings[i].TermFrequency++
			return
		}
	}

	// Add new posting
	pl.Postings = append(pl.Postings, Posting{
		DocID:         docID,
		Positions:     []int{position},
		TermFrequency: 1,
	})
	pl.DocFrequency++
}

// Sort sorts postings by DocID for efficient merging and compression.
func (pl *PostingsList) Sort() {
	sort.Slice(pl.Postings, func(i, j int) bool {
		return pl.Postings[i].DocID < pl.Postings[j].DocID
	})
}

// Merge merges another postings list into this one.
// Both lists must be sorted by DocID before merging.
// The result is a sorted union of both lists with combined positions.
func (pl *PostingsList) Merge(other *PostingsList) {
	if other == nil || len(other.Postings) == 0 {
		return
	}

	result := make([]Posting, 0, pl.DocFrequency+other.DocFrequency)
	i, j := 0, 0

	for i < len(pl.Postings) && j < len(other.Postings) {
		p1 := &pl.Postings[i]
		p2 := &other.Postings[j]

		if p1.DocID == p2.DocID {
			// Same document - merge positions
			merged := Posting{
				DocID:         p1.DocID,
				Positions:     append(p1.Positions, p2.Positions...),
				TermFrequency: p1.TermFrequency + p2.TermFrequency,
			}
			sort.Ints(merged.Positions)
			result = append(result, merged)
			i++
			j++
		} else if p1.DocID < p2.DocID {
			result = append(result, *p1)
			i++
		} else {
			result = append(result, *p2)
			j++
		}
	}

	// Add remaining postings from first list
	for i < len(pl.Postings) {
		result = append(result, pl.Postings[i])
		i++
	}

	// Add remaining postings from second list
	for j < len(other.Postings) {
		result = append(result, other.Postings[j])
		j++
	}

	pl.Postings = result
	pl.DocFrequency = len(result)
}

// Intersect performs an AND operation with another postings list.
// Returns a new postings list containing only documents present in both lists.
// Both lists must be sorted by DocID.
func (pl *PostingsList) Intersect(other *PostingsList) *PostingsList {
	if pl == nil || other == nil || len(pl.Postings) == 0 || len(other.Postings) == 0 {
		return &PostingsList{DocFrequency: 0, Postings: []Posting{}}
	}

	result := &PostingsList{
		Postings: make([]Posting, 0, min(pl.DocFrequency, other.DocFrequency)),
	}

	i, j := 0, 0
	for i < len(pl.Postings) && j < len(other.Postings) {
		p1 := &pl.Postings[i]
		p2 := &other.Postings[j]

		if p1.DocID == p2.DocID {
			// Document in both lists - add to result
			result.Postings = append(result.Postings, *p1)
			result.DocFrequency++
			i++
			j++
		} else if p1.DocID < p2.DocID {
			i++
		} else {
			j++
		}
	}

	return result
}

// Union performs an OR operation with another postings list.
// Returns a new postings list containing documents from either list.
// Both lists must be sorted by DocID.
func (pl *PostingsList) Union(other *PostingsList) *PostingsList {
	if pl == nil || len(pl.Postings) == 0 {
		return other
	}
	if other == nil || len(other.Postings) == 0 {
		return pl
	}

	result := &PostingsList{
		Postings: make([]Posting, 0, pl.DocFrequency+other.DocFrequency),
	}

	i, j := 0, 0
	for i < len(pl.Postings) && j < len(other.Postings) {
		p1 := &pl.Postings[i]
		p2 := &other.Postings[j]

		if p1.DocID == p2.DocID {
			// Same document - add once
			result.Postings = append(result.Postings, *p1)
			result.DocFrequency++
			i++
			j++
		} else if p1.DocID < p2.DocID {
			result.Postings = append(result.Postings, *p1)
			result.DocFrequency++
			i++
		} else {
			result.Postings = append(result.Postings, *p2)
			result.DocFrequency++
			j++
		}
	}

	// Add remaining postings
	for i < len(pl.Postings) {
		result.Postings = append(result.Postings, pl.Postings[i])
		result.DocFrequency++
		i++
	}
	for j < len(other.Postings) {
		result.Postings = append(result.Postings, other.Postings[j])
		result.DocFrequency++
		j++
	}

	return result
}

// Difference performs a NOT operation (this - other).
// Returns a new postings list containing documents in this list but not in other.
// Both lists must be sorted by DocID.
func (pl *PostingsList) Difference(other *PostingsList) *PostingsList {
	if pl == nil || len(pl.Postings) == 0 {
		return &PostingsList{DocFrequency: 0, Postings: []Posting{}}
	}
	if other == nil || len(other.Postings) == 0 {
		return pl
	}

	result := &PostingsList{
		Postings: make([]Posting, 0, pl.DocFrequency),
	}

	i, j := 0, 0
	for i < len(pl.Postings) && j < len(other.Postings) {
		p1 := &pl.Postings[i]
		p2 := &other.Postings[j]

		if p1.DocID == p2.DocID {
			// Skip this document
			i++
			j++
		} else if p1.DocID < p2.DocID {
			result.Postings = append(result.Postings, *p1)
			result.DocFrequency++
			i++
		} else {
			j++
		}
	}

	// Add remaining postings from first list
	for i < len(pl.Postings) {
		result.Postings = append(result.Postings, pl.Postings[i])
		result.DocFrequency++
		i++
	}

	return result
}

// GetPosting returns the posting for a specific document ID.
// Returns nil if the document is not in the postings list.
func (pl *PostingsList) GetPosting(docID string) *Posting {
	for i := range pl.Postings {
		if pl.Postings[i].DocID == docID {
			return &pl.Postings[i]
		}
	}
	return nil
}

// HasDocument returns true if the postings list contains the given document ID.
func (pl *PostingsList) HasDocument(docID string) bool {
	for i := range pl.Postings {
		if pl.Postings[i].DocID == docID {
			return true
		}
	}
	return false
}

// Clone creates a deep copy of the postings list.
func (pl *PostingsList) Clone() *PostingsList {
	if pl == nil {
		return nil
	}

	result := &PostingsList{
		DocFrequency: pl.DocFrequency,
		Postings:     make([]Posting, len(pl.Postings)),
	}

	for i, p := range pl.Postings {
		result.Postings[i] = Posting{
			DocID:         p.DocID,
			TermFrequency: p.TermFrequency,
			Positions:     make([]int, len(p.Positions)),
		}
		copy(result.Postings[i].Positions, p.Positions)
	}

	return result
}

// PositionalIntersect performs positional intersection for phrase queries.
// Returns documents where all terms appear within the specified distance.
// If distance is 0, terms must be adjacent (exact phrase).
func (pl *PostingsList) PositionalIntersect(other *PostingsList, distance int) *PostingsList {
	if pl == nil || other == nil || len(pl.Postings) == 0 || len(other.Postings) == 0 {
		return &PostingsList{DocFrequency: 0, Postings: []Posting{}}
	}

	result := &PostingsList{
		Postings: make([]Posting, 0),
	}

	i, j := 0, 0
	for i < len(pl.Postings) && j < len(other.Postings) {
		p1 := &pl.Postings[i]
		p2 := &other.Postings[j]

		if p1.DocID == p2.DocID {
			// Check for positional matches
			positions := pl.positionalMatch(p1.Positions, p2.Positions, distance)
			if len(positions) > 0 {
				result.Postings = append(result.Postings, Posting{
					DocID:         p1.DocID,
					Positions:     positions,
					TermFrequency: len(positions),
				})
				result.DocFrequency++
			}
			i++
			j++
		} else if p1.DocID < p2.DocID {
			i++
		} else {
			j++
		}
	}

	return result
}

// positionalMatch finds positions where two position lists are within distance.
func (pl *PostingsList) positionalMatch(pos1, pos2 []int, distance int) []int {
	var matches []int

	i, j := 0, 0
	for i < len(pos1) && j < len(pos2) {
		// For phrase queries, pos2 should come after pos1
		diff := pos2[j] - pos1[i]

		if diff >= 1 && diff <= distance+1 {
			// Match found - record the position of the first term
			matches = append(matches, pos1[i])
			// Move to next potential match
			if i+1 < len(pos1) && pos1[i+1] <= pos2[j] {
				i++
			} else {
				j++
			}
		} else if pos2[j] > pos1[i] {
			// Need to advance i
			i++
		} else {
			// Need to advance j
			j++
		}
	}

	return matches
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
