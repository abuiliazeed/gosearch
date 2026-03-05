package indexer

import (
	"reflect"
	"testing"
)

// TestEncodeGaps tests variable-byte encoding of gap sequences.
func TestEncodeGaps(t *testing.T) {
	tests := []struct {
		name    string
		gaps    []int
		wantNil bool
		minSize int
		maxSize int
	}{
		{
			name:    "Empty gaps",
			gaps:    []int{},
			wantNil: true,
		},
		{
			name:    "Nil gaps",
			gaps:    nil,
			wantNil: true,
		},
		{
			name:    "Single small value",
			gaps:    []int{5},
			minSize: 1,
			maxSize: 1,
		},
		{
			name:    "Single value 127 (fits in 1 byte)",
			gaps:    []int{127},
			minSize: 1,
			maxSize: 1,
		},
		{
			name:    "Single value 128 (needs 2 bytes)",
			gaps:    []int{128},
			minSize: 2,
			maxSize: 2,
		},
		{
			name:    "Single large value",
			gaps:    []int{16383},
			minSize: 2,
			maxSize: 2,
		},
		{
			name:    "Very large value (needs 3 bytes)",
			gaps:    []int{2097151},
			minSize: 3,
			maxSize: 3,
		},
		{
			name:    "Multiple small values",
			gaps:    []int{1, 2, 3, 4, 5},
			minSize: 5,
			maxSize: 5,
		},
		{
			name:    "Mixed size values",
			gaps:    []int{5, 128, 3, 16384, 1},
			minSize: 8,
			maxSize: 8,
		},
		{
			name:    "Document gaps (realistic)",
			gaps:    []int{1, 5, 12, 100, 500},
			minSize: 5,
			maxSize: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeGaps(tt.gaps)

			if tt.wantNil {
				if got != nil {
					t.Errorf("EncodeGaps() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("EncodeGaps() = nil, want non-nil")
				return
			}

			if tt.minSize > 0 && len(got) < tt.minSize {
				t.Errorf("EncodeGaps() size = %d, want >= %d", len(got), tt.minSize)
			}
			if tt.maxSize > 0 && len(got) > tt.maxSize {
				t.Errorf("EncodeGaps() size = %d, want <= %d", len(got), tt.maxSize)
			}
		})
	}
}

// TestDecodeGaps tests decoding of variable-byte encoded gaps.
func TestDecodeGaps(t *testing.T) {
	tests := []struct {
		name string
		gaps []int
	}{
		{
			name: "Empty gaps",
			gaps: []int{},
		},
		{
			name: "Nil gaps",
			gaps: nil,
		},
		{
			name: "Single small value",
			gaps: []int{5},
		},
		{
			name: "Single value 127",
			gaps: []int{127},
		},
		{
			name: "Single value 128",
			gaps: []int{128},
		},
		{
			name: "Single large value",
			gaps: []int{16383},
		},
		{
			name: "Multiple small values",
			gaps: []int{1, 2, 3, 4, 5},
		},
		{
			name: "Mixed size values",
			gaps: []int{5, 128, 3, 16384, 1},
		},
		{
			name: "Realistic doc gaps",
			gaps: []int{1, 5, 12, 100, 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode first
			encoded := EncodeGaps(tt.gaps)

			// Then decode
			got := DecodeGaps(encoded)

			// Handle empty slice vs nil slice equivalence
			if len(tt.gaps) == 0 && len(got) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.gaps) {
				t.Errorf("DecodeGaps(EncodeGaps(%v)) = %v, want %v", tt.gaps, got, tt.gaps)
			}
		})
	}
}

// TestEncodeGapsRoundTrip tests that encoding and decoding are symmetric.
func TestEncodeGapsRoundTrip(t *testing.T) {
	testCases := [][]int{
		{1},
		{1, 2, 3, 4, 5},
		{100, 200, 300},
		{127, 128, 129},
		{16383, 16384, 16385},
		{1, 1000, 100000, 10000000},
		{},
		nil,
	}

	for _, gaps := range testCases {
		encoded := EncodeGaps(gaps)
		decoded := DecodeGaps(encoded)

		// Handle empty slice vs nil slice equivalence
		if len(gaps) == 0 && len(decoded) == 0 {
			continue
		}
		if !reflect.DeepEqual(decoded, gaps) {
			t.Errorf("Round trip failed: original=%v, decoded=%v", gaps, decoded)
		}
	}
}

// TestEncodeDocIDs tests encoding of document IDs.
func TestEncodeDocIDs(t *testing.T) {
	tests := []struct {
		name    string
		docIDs  []string
		wantNil bool
	}{
		{
			name:    "Empty docIDs",
			docIDs:  []string{},
			wantNil: true,
		},
		{
			name:    "Nil docIDs",
			docIDs:  nil,
			wantNil: true,
		},
		{
			name:    "Single docID",
			docIDs:  []string{"doc1"},
			wantNil: false,
		},
		{
			name:    "Multiple docIDs",
			docIDs:  []string{"doc1", "doc2", "doc3"},
			wantNil: false,
		},
		{
			name:    "Long docIDs",
			docIDs:  []string{"very_long_document_id_12345", "another_long_id_67890"},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeDocIDs(tt.docIDs)

			if tt.wantNil {
				if got != nil {
					t.Errorf("EncodeDocIDs() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("EncodeDocIDs() = nil, want non-nil")
			}
		})
	}
}

// TestDecodeDocIDs tests decoding of document IDs.
func TestDecodeDocIDs(t *testing.T) {
	tests := []struct {
		name   string
		docIDs []string
	}{
		{
			name:   "Empty docIDs",
			docIDs: []string{},
		},
		{
			name:   "Single docID",
			docIDs: []string{"doc1"},
		},
		{
			name:   "Multiple docIDs",
			docIDs: []string{"doc1", "doc2", "doc3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeDocIDs(tt.docIDs)
			decoded := DecodeDocIDs(encoded)

			// Note: Due to the lossy hash/unhash implementation,
			// we only check that we get the same number of results
			if len(decoded) != len(tt.docIDs) {
				t.Errorf("DecodeDocIDs() length = %d, want %d", len(decoded), len(tt.docIDs))
			}
		})
	}
}

// TestEncodePostingsList tests encoding of a full postings list.
func TestEncodePostingsList(t *testing.T) {
	tests := []struct {
		name    string
		pl      *PostingsList
		wantNil bool
	}{
		{
			name:    "Nil postings list",
			pl:      nil,
			wantNil: true,
		},
		{
			name:    "Empty postings list",
			pl:      &PostingsList{DocFrequency: 0, Postings: []Posting{}},
			wantNil: true,
		},
		{
			name: "Single posting, no positions",
			pl: &PostingsList{
				DocFrequency: 1,
				Postings: []Posting{
					{DocID: "doc1", Positions: []int{}, TermFrequency: 0},
				},
			},
			wantNil: false,
		},
		{
			name: "Single posting with positions",
			pl: &PostingsList{
				DocFrequency: 1,
				Postings: []Posting{
					{DocID: "doc1", Positions: []int{1, 5, 10}, TermFrequency: 3},
				},
			},
			wantNil: false,
		},
		{
			name: "Multiple postings",
			pl: &PostingsList{
				DocFrequency: 3,
				Postings: []Posting{
					{DocID: "doc1", Positions: []int{1, 5}, TermFrequency: 2},
					{DocID: "doc2", Positions: []int{3, 7, 12}, TermFrequency: 3},
					{DocID: "doc3", Positions: []int{8}, TermFrequency: 1},
				},
			},
			wantNil: false,
		},
		{
			name: "Large docID",
			pl: &PostingsList{
				DocFrequency: 1,
				Postings: []Posting{
					{DocID: "very_long_document_id_with_many_characters", Positions: []int{1, 2, 3}, TermFrequency: 3},
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodePostingsList(tt.pl)

			if tt.wantNil {
				if got != nil {
					t.Errorf("EncodePostingsList() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("EncodePostingsList() = nil, want non-nil")
				return
			}

			// Check that we can decode it back
			decoded, err := DecodePostingsList(got)
			if err != nil {
				t.Errorf("DecodePostingsList() returned error: %v", err)
				return
			}

			if decoded.DocFrequency != tt.pl.DocFrequency {
				t.Errorf("DocFrequency = %d, want %d", decoded.DocFrequency, tt.pl.DocFrequency)
			}

			if len(decoded.Postings) != len(tt.pl.Postings) {
				t.Errorf("Postings count = %d, want %d", len(decoded.Postings), len(tt.pl.Postings))
			}
		})
	}
}

// TestDecodePostingsList tests decoding of postings list data.
func TestDecodePostingsList(t *testing.T) {
	tests := []struct {
		name    string
		pl      *PostingsList
		wantErr bool
	}{
		{
			name:    "Empty data",
			pl:      nil,
			wantErr: false,
		},
		{
			name: "Single posting",
			pl: &PostingsList{
				DocFrequency: 1,
				Postings: []Posting{
					{DocID: "doc1", Positions: []int{1, 5, 10}, TermFrequency: 3},
				},
			},
			wantErr: false,
		},
		{
			name: "Multiple postings",
			pl: &PostingsList{
				DocFrequency: 2,
				Postings: []Posting{
					{DocID: "doc1", Positions: []int{1, 5}, TermFrequency: 2},
					{DocID: "doc2", Positions: []int{3, 7, 12}, TermFrequency: 3},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode first
			encoded := EncodePostingsList(tt.pl)

			// Then decode
			got, err := DecodePostingsList(encoded)

			if (err != nil) != tt.wantErr {
				t.Errorf("DecodePostingsList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.pl == nil {
				if got == nil || got.DocFrequency != 0 || len(got.Postings) != 0 {
					t.Errorf("DecodePostingsList() = %v, want empty PostingsList", got)
				}
				return
			}

			// Verify decoded data matches original
			if got.DocFrequency != tt.pl.DocFrequency {
				t.Errorf("DocFrequency = %d, want %d", got.DocFrequency, tt.pl.DocFrequency)
			}

			if len(got.Postings) != len(tt.pl.Postings) {
				t.Errorf("Postings count = %d, want %d", len(got.Postings), len(tt.pl.Postings))
				return
			}

			for i := range got.Postings {
				if got.Postings[i].DocID != tt.pl.Postings[i].DocID {
					t.Errorf("Posting[%d].DocID = %s, want %s", i, got.Postings[i].DocID, tt.pl.Postings[i].DocID)
				}
				if got.Postings[i].TermFrequency != tt.pl.Postings[i].TermFrequency {
					t.Errorf("Posting[%d].TermFrequency = %d, want %d", i, got.Postings[i].TermFrequency, tt.pl.Postings[i].TermFrequency)
				}
				if !reflect.DeepEqual(got.Postings[i].Positions, tt.pl.Postings[i].Positions) {
					t.Errorf("Posting[%d].Positions = %v, want %v", i, got.Postings[i].Positions, tt.pl.Postings[i].Positions)
				}
			}
		})
	}
}

// TestDecodePostingsListErrors tests error conditions for decoding.
func TestDecodePostingsListErrors(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "Empty data",
			data:    []byte{},
			wantErr: false, // Returns empty PostingsList
		},
		{
			name:    "Nil data",
			data:    nil,
			wantErr: false, // Returns empty PostingsList
		},
		{
			name:    "Truncated header (1 byte)",
			data:    []byte{0x01},
			wantErr: true, // ErrStorage
		},
		{
			name:    "Truncated during docID length (only header)",
			data:    []byte{0x00, 0x01}, // DocFreq=1, but nothing follows
			wantErr: true,               // ErrStorage
		},
		{
			name:    "Truncated during docID (header + 1 byte of length)",
			data:    []byte{0x00, 0x01, 0x04, 0x64}, // DocFreq=1, DocIDLen=4, only 'd'
			wantErr: true,                           // ErrStorage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePostingsList(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodePostingsList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEncodePostingsListRoundTrip tests full round-trip encoding/decoding.
func TestEncodePostingsListRoundTrip(t *testing.T) {
	testCases := []*PostingsList{
		nil,
		{DocFrequency: 0, Postings: []Posting{}},
		{
			DocFrequency: 1,
			Postings: []Posting{
				{DocID: "doc1", Positions: []int{1, 5, 10}, TermFrequency: 3},
			},
		},
		{
			DocFrequency: 3,
			Postings: []Posting{
				{DocID: "doc1", Positions: []int{1, 5}, TermFrequency: 2},
				{DocID: "doc2", Positions: []int{3, 7, 12}, TermFrequency: 3},
				{DocID: "doc3", Positions: []int{8}, TermFrequency: 1},
			},
		},
		{
			DocFrequency: 2,
			Postings: []Posting{
				{DocID: "document_with_long_id_123", Positions: []int{1, 2, 3, 4, 5}, TermFrequency: 5},
				{DocID: "d2", Positions: []int{100, 200, 300}, TermFrequency: 3},
			},
		},
	}

	for _, pl := range testCases {
		encoded := EncodePostingsList(pl)
		decoded, err := DecodePostingsList(encoded)

		if err != nil {
			t.Errorf("DecodePostingsList() error = %v", err)
			continue
		}

		if pl == nil || len(pl.Postings) == 0 {
			if decoded == nil || (decoded.DocFrequency != 0 || len(decoded.Postings) != 0) {
				t.Errorf("Round trip failed for empty: original=%v, decoded=%v", pl, decoded)
			}
			continue
		}

		if decoded.DocFrequency != pl.DocFrequency {
			t.Errorf("Round trip failed: DocFrequency original=%d, decoded=%d", pl.DocFrequency, decoded.DocFrequency)
		}

		if len(decoded.Postings) != len(pl.Postings) {
			t.Errorf("Round trip failed: Postings count original=%d, decoded=%d", len(pl.Postings), len(decoded.Postings))
			continue
		}

		for i := range decoded.Postings {
			if decoded.Postings[i].DocID != pl.Postings[i].DocID {
				t.Errorf("Round trip failed: Posting[%d].DocID original=%s, decoded=%s", i, pl.Postings[i].DocID, decoded.Postings[i].DocID)
			}
			if decoded.Postings[i].TermFrequency != pl.Postings[i].TermFrequency {
				t.Errorf("Round trip failed: Posting[%d].TermFrequency original=%d, decoded=%d", i, pl.Postings[i].TermFrequency, decoded.Postings[i].TermFrequency)
			}
			if !reflect.DeepEqual(decoded.Postings[i].Positions, pl.Postings[i].Positions) {
				t.Errorf("Round trip failed: Posting[%d].Positions original=%v, decoded=%v", i, pl.Postings[i].Positions, decoded.Postings[i].Positions)
			}
		}
	}
}

// TestCompressRatio tests the compression ratio calculation.
func TestCompressRatio(t *testing.T) {
	tests := []struct {
		name           string
		originalSize   int
		compressedSize int
		want           float64
	}{
		{
			name:           "Zero original size",
			originalSize:   0,
			compressedSize: 100,
			want:           1.0,
		},
		{
			name:           "Perfect compression (half size)",
			originalSize:   1000,
			compressedSize: 500,
			want:           0.5,
		},
		{
			name:           "No compression (same size)",
			originalSize:   1000,
			compressedSize: 1000,
			want:           1.0,
		},
		{
			name:           "Expansion (larger)",
			originalSize:   1000,
			compressedSize: 1200,
			want:           1.2,
		},
		{
			name:           "Good compression (quarter size)",
			originalSize:   4000,
			compressedSize: 1000,
			want:           0.25,
		},
		{
			name:           "Small file compression",
			originalSize:   100,
			compressedSize: 75,
			want:           0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompressRatio(tt.originalSize, tt.compressedSize)
			if got != tt.want {
				t.Errorf("CompressRatio() = %f, want %f", got, tt.want)
			}
		})
	}
}

// TestEstimateEncodedSize tests the size estimation utility.
func TestEstimateEncodedSize(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  int
	}{
		{
			name:  "Zero count",
			count: 0,
			want:  0,
		},
		{
			name:  "Single item",
			count: 1,
			want:  1, // 1 * 3 / 2 = 1 (integer division)
		},
		{
			name:  "Two items",
			count: 2,
			want:  3, // 2 * 3 / 2 = 3
		},
		{
			name:  "Ten items",
			count: 10,
			want:  15, // 10 * 3 / 2 = 15
		},
		{
			name:  "Hundred items",
			count: 100,
			want:  150, // 100 * 3 / 2 = 150
		},
		{
			name:  "Odd count",
			count: 7,
			want:  10, // 7 * 3 / 2 = 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateEncodedSize(tt.count)
			if got != tt.want {
				t.Errorf("EstimateEncodedSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestHashDocID tests the docID hashing function.
func TestHashDocID(t *testing.T) {
	tests := []struct {
		name  string
		docID string
		want  int
	}{
		{
			name:  "Empty string",
			docID: "",
			want:  0,
		},
		{
			name:  "Simple docID",
			docID: "doc1",
			want:  3088889, // hash: 0->100->3211->99640->3088889
		},
		{
			name:  "Another docID",
			docID: "doc2",
			want:  3088890, // hash: 0->100->3211->99640->3088890
		},
		{
			name:  "Different strings should hash differently",
			docID: "abc",
			want:  96354, // hash: 0->97->3105->96354
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hashDocID(tt.docID)
			if got != tt.want {
				t.Errorf("hashDocID() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestHashDocIDDeterministic tests that hashing is deterministic.
func TestHashDocIDDeterministic(t *testing.T) {
	docID := "test_document_123"

	// Hash the same docID multiple times
	results := make([]int, 10)
	for i := range results {
		results[i] = hashDocID(docID)
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Errorf("hashDocID() not deterministic: got %d, want %d", results[i], results[0])
		}
	}
}

// TestHashDocIDPositive tests that hashing always returns positive values.
func TestHashDocIDPositive(t *testing.T) {
	docIDs := []string{
		"doc1", "doc2", "test", "a", "Z", "123", "very_long_document_id",
		"special!@#$%", "unicode\u2603", "mixedCASE123",
	}

	for _, docID := range docIDs {
		got := hashDocID(docID)
		if got < 0 {
			t.Errorf("hashDocID(%q) = %d, want >= 0", docID, got)
		}
	}
}
