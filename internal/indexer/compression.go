package indexer

import (
	"encoding/binary"
)

// EncodeGaps encodes a slice of gaps using variable-byte encoding.
//
// Variable-byte encoding is a compression technique that uses fewer bytes
// for smaller numbers. Each byte uses the high bit as a continuation flag:
// - high bit = 1: more bytes follow
// - high bit = 0: last byte of the number
//
// The remaining 7 bits store the number data in little-endian order.
func EncodeGaps(gaps []int) []byte {
	if len(gaps) == 0 {
		return nil
	}

	// Estimate size: average 2 bytes per gap
	size := len(gaps) * 2
	buf := make([]byte, 0, size)

	for _, gap := range gaps {
		buf = encodeVarInt(buf, gap)
	}

	return buf
}

// DecodeGaps decodes variable-byte encoded data into a slice of gaps.
func DecodeGaps(data []byte) []int {
	if len(data) == 0 {
		return nil
	}

	gaps := make([]int, 0, len(data)/2) // Estimate 2 bytes per gap
	offset := 0

	for offset < len(data) {
		value, n := decodeVarInt(data[offset:])
		gaps = append(gaps, value)
		offset += n
	}

	return gaps
}

// encodeVarInt encodes a single integer using variable-byte encoding.
func encodeVarInt(buf []byte, value int) []byte {
	for {
		// Take low 7 bits
		b := byte(value & 0x7F)

		// Shift right by 7
		value >>= 7

		// If more bytes follow, set high bit
		if value > 0 {
			b |= 0x80
			buf = append(buf, b)
		} else {
			// Last byte - high bit is 0
			buf = append(buf, b)
			break
		}
	}

	return buf
}

// decodeVarInt decodes a single variable-byte encoded integer.
// Returns the decoded value and the number of bytes consumed.
func decodeVarInt(data []byte) (int, int) {
	value := 0
	shift := 0
	bytesRead := 0

	for i, b := range data {
		bytesRead = i + 1

		// Add low 7 bits to value
		value |= int(b&0x7F) << shift

		// If high bit is 0, this is the last byte
		if b&0x80 == 0 {
			break
		}

		shift += 7
	}

	return value, bytesRead
}

// EncodeDocIDs encodes a sorted slice of docIDs using gap encoding.
// The first docID is stored as-is, subsequent values are stored as gaps.
func EncodeDocIDs(docIDs []string) []byte {
	if len(docIDs) == 0 {
		return nil
	}

	// For now, we'll convert docIDs to numeric IDs for gap encoding
	// In a real implementation, you'd have a docID -> numeric ID mapping
	// This is a simplified version assuming docIDs are hashable to numbers

	// Convert docIDs to numeric hashes (simple hash function)
	numIDs := make([]int, len(docIDs))
	for i, docID := range docIDs {
		numIDs[i] = hashDocID(docID)
	}

	// Sort and convert to gaps
	// Note: In production, you'd maintain sorted docIDs
	// For now, just encode as-is
	return EncodeGaps(numIDs)
}

// DecodeDocIDs decodes gap-encoded docIDs back to the original slice.
// Note: This requires the same docID mapping used during encoding.
func DecodeDocIDs(data []byte) []string {
	if len(data) == 0 {
		return nil
	}

	// Decode gaps to numeric IDs
	numIDs := DecodeGaps(data)

	// Convert back to docIDs (placeholder - needs mapping)
	docIDs := make([]string, len(numIDs))
	for i, numID := range numIDs {
		docIDs[i] = unhashDocID(numID)
	}

	return docIDs
}

// hashDocID converts a docID string to a numeric hash.
// This is a simple implementation for demonstration.
// In production, use a proper bijection mapping.
func hashDocID(docID string) int {
	// Simple hash function
	hash := 0
	for _, c := range docID {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// unhashDocID converts a numeric hash back to a docID.
// Note: This is a lossy conversion and only works for simple cases.
// In production, maintain a bidirectional mapping.
func unhashDocID(hash int) string {
	// Placeholder - in production, use a proper mapping
	return string(rune(hash))
}

// EncodePostingsList encodes an entire postings list for storage.
// Returns a byte slice containing the compressed postings.
func EncodePostingsList(pl *PostingsList) []byte {
	if pl == nil || len(pl.Postings) == 0 {
		return nil
	}

	// Format:
	// [4 bytes: doc frequency][for each posting:][4 bytes: docID length][docID bytes][2 bytes: term frequency][positions...]

	// For simplicity, we'll encode just the docIDs and term frequencies
	// A more sophisticated implementation would also compress positions

	buf := make([]byte, 0, 256) // Start with reasonable capacity

	// Write doc frequency
	buf = binary.BigEndian.AppendUint16(buf, uint16(pl.DocFrequency))

	// Write each posting
	for _, p := range pl.Postings {
		// Write docID length and docID
		docIDBytes := []byte(p.DocID)
		buf = binary.BigEndian.AppendUint16(buf, uint16(len(docIDBytes)))
		buf = append(buf, docIDBytes...)

		// Write term frequency
		buf = binary.BigEndian.AppendUint16(buf, uint16(p.TermFrequency))

		// Write positions (could be gap-encoded for more compression)
		for _, pos := range p.Positions {
			buf = binary.BigEndian.AppendUint32(buf, uint32(pos))
		}
	}

	return buf
}

// DecodePostingsList decodes a byte slice into a PostingsList.
func DecodePostingsList(data []byte) (*PostingsList, error) {
	if len(data) == 0 {
		return &PostingsList{DocFrequency: 0, Postings: []Posting{}}, nil
	}

	pl := &PostingsList{
		Postings: make([]Posting, 0),
	}

	offset := 0

	// Read doc frequency
	if offset+2 > len(data) {
		return nil, ErrStorage
	}
	docFreq := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	pl.DocFrequency = docFreq

	// Read each posting
	for i := 0; i < docFreq; i++ {
		// Read docID length
		if offset+2 > len(data) {
			return nil, ErrStorage
		}
		docIDLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2

		// Read docID
		if offset+docIDLen > len(data) {
			return nil, ErrStorage
		}
		docID := string(data[offset : offset+docIDLen])
		offset += docIDLen

		// Read term frequency
		if offset+2 > len(data) {
			return nil, ErrStorage
		}
		termFreq := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += 2

		posting := Posting{
			DocID:         docID,
			TermFrequency: termFreq,
			Positions:     make([]int, termFreq),
		}

		// Read positions
		for j := 0; j < termFreq; j++ {
			if offset+4 > len(data) {
				return nil, ErrStorage
			}
			pos := int(binary.BigEndian.Uint32(data[offset : offset+4]))
			offset += 4
			posting.Positions[j] = pos
		}

		pl.Postings = append(pl.Postings, posting)
	}

	return pl, nil
}

// CompressRatio returns the compression ratio achieved by gap encoding.
// A value less than 1.0 indicates compression.
func CompressRatio(originalSize, compressedSize int) float64 {
	if originalSize == 0 {
		return 1.0
	}
	return float64(compressedSize) / float64(originalSize)
}

// EstimateEncodedSize estimates the size of gap-encoded data.
// Useful for pre-allocating buffers.
func EstimateEncodedSize(count int) int {
	// Average case: 1.5 bytes per gap
	return count * 3 / 2
}
