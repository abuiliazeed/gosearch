package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// IndexStore handles index metadata storage using BoltDB.
// It stores term information, index metadata, and postings list locations.
type IndexStore struct {
	db     *bolt.DB
	isOpen bool
}

// Bucket names
var (
	MetaBucket      = []byte("meta")
	TermsBucket     = []byte("terms")
	DocumentsBucket = []byte("documents")
	PostingsBucket  = []byte("postings")
	DocInfoBucket   = []byte("docinfo")
)

// NewIndexStore creates a new IndexStore with the given file path.
func NewIndexStore(filePath string) (*IndexStore, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create index store directory: %w", err)
	}

	// Open database
	db, err := bolt.Open(filePath, 0600, &bolt.Options{Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(MetaBucket); err != nil {
			return fmt.Errorf("failed to create meta bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(TermsBucket); err != nil {
			return fmt.Errorf("failed to create terms bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(DocumentsBucket); err != nil {
			return fmt.Errorf("failed to create documents bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(PostingsBucket); err != nil {
			return fmt.Errorf("failed to create postings bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(DocInfoBucket); err != nil {
			return fmt.Errorf("failed to create docinfo bucket: %w", err)
		}
		return nil
	})

	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &IndexStore{db: db, isOpen: true}, nil
}

// Close closes the database connection.
func (is *IndexStore) Close() error {
	if is.db != nil {
		return is.db.Close()
	}
	return nil
}

// SaveMeta saves index metadata.
func (is *IndexStore) SaveMeta(meta *IndexMeta) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(MetaBucket)
		data, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("failed to marshal meta: %w", err)
		}
		return b.Put([]byte("index_meta"), data)
	})
}

// GetMeta retrieves index metadata.
func (is *IndexStore) GetMeta() (*IndexMeta, error) {
	var meta IndexMeta
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(MetaBucket)
		data := b.Get([]byte("index_meta"))
		if data == nil {
			meta = IndexMeta{
				TotalDocuments: 0,
				TotalTerms:     0,
				LastUpdated:    time.Now(),
			}
			return nil
		}
		return json.Unmarshal(data, &meta)
	})

	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// CheckLock verifies the database is not locked before starting operations.
func (is *IndexStore) CheckLock() error {
	// Try to get info with read-only transaction
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(MetaBucket)
		_ = b.Get([]byte("index_meta"))
		return nil
	})
	if err != nil {
		return fmt.Errorf("database check failed: %w", err)
	}
	return nil
}

// SaveTermInfo saves information about a term in the index.
func (is *IndexStore) SaveTermInfo(term string, info *TermInfo) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(TermsBucket)
		data, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("failed to marshal term info: %w", err)
		}
		return b.Put([]byte(term), data)
	})
}

// GetTermInfo retrieves information about a term.
func (is *IndexStore) GetTermInfo(term string) (*TermInfo, error) {
	var info TermInfo
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(TermsBucket)
		data := b.Get([]byte(term))
		if data == nil {
			return fmt.Errorf("term not found: %s", term)
		}
		return json.Unmarshal(data, &info)
	})

	if err != nil {
		return nil, err
	}
	return &info, nil
}

// DeleteTerm removes a term from the index.
func (is *IndexStore) DeleteTerm(term string) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(TermsBucket)
		return b.Delete([]byte(term))
	})
}

// ListTerms returns all terms in the index.
func (is *IndexStore) ListTerms() ([]string, error) {
	var terms []string
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(TermsBucket)
		return b.ForEach(func(k, _ []byte) error {
			terms = append(terms, string(k))
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return terms, nil
}

// UpdateDocumentCount updates the total document count in the metadata.
func (is *IndexStore) UpdateDocumentCount(delta int) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(MetaBucket)
		data := b.Get([]byte("index_meta"))

		var meta IndexMeta
		if data != nil {
			if err := json.Unmarshal(data, &meta); err != nil {
				return err
			}
		}

		meta.TotalDocuments += delta
		meta.LastUpdated = time.Now()

		newData, err := json.Marshal(meta)
		if err != nil {
			return err
		}

		return b.Put([]byte("index_meta"), newData)
	})
}

// GetDocumentCount returns the total number of documents in the index.
func (is *IndexStore) GetDocumentCount() (int, error) {
	meta, err := is.GetMeta()
	if err != nil {
		return 0, err
	}
	return meta.TotalDocuments, nil
}

// AddDocument adds a document to the documents bucket.
func (is *IndexStore) AddDocument(docID string, url string) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(DocumentsBucket)
		docInfo := map[string]interface{}{
			"url":   url,
			"added": time.Now().Unix(),
		}
		data, err := json.Marshal(docInfo)
		if err != nil {
			return err
		}
		return b.Put([]byte(docID), data)
	})
}

// GetDocumentURL retrieves a document URL by ID.
func (is *IndexStore) GetDocumentURL(docID string) (string, error) {
	var url string
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(DocumentsBucket)
		data := b.Get([]byte(docID))
		if data == nil {
			return fmt.Errorf("document not found: %s", docID)
		}
		var docInfo map[string]interface{}
		if err := json.Unmarshal(data, &docInfo); err != nil {
			return err
		}
		url = docInfo["url"].(string)
		return nil
	})

	if err != nil {
		return "", err
	}
	return url, nil
}

// Backup creates a backup of the index store to the given destination.
func (is *IndexStore) Backup(dest string) error {
	return is.db.View(func(tx *bolt.Tx) error {
		return tx.CopyFile(dest, 0600)
	})
}

// Stats returns statistics about the index store.
func (is *IndexStore) Stats() (map[string]int64, error) {
	stats := make(map[string]int64)
	err := is.db.View(func(tx *bolt.Tx) error {
		// Stats for each bucket
		for _, name := range [][]byte{MetaBucket, TermsBucket, DocumentsBucket, PostingsBucket, DocInfoBucket} {
			b := tx.Bucket(name)
			if b == nil {
				continue
			}
			stats[string(name)] = int64(b.Stats().KeyN)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return stats, nil
}

// SavePostings saves a complete postings list for a term to BoltDB.
func (is *IndexStore) SavePostings(plist *PersistedPostingsList) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(PostingsBucket)
		data, err := json.Marshal(plist)
		if err != nil {
			return fmt.Errorf("failed to marshal postings list: %w", err)
		}
		return b.Put([]byte(plist.Term), data)
	})
}

// SavePostingsBatch saves multiple postings lists in a single transaction.
func (is *IndexStore) SavePostingsBatch(lists []*PersistedPostingsList) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(PostingsBucket)
		for _, plist := range lists {
			data, err := json.Marshal(plist)
			if err != nil {
				return fmt.Errorf("failed to marshal postings list for term %s: %w", plist.Term, err)
			}
			if err := b.Put([]byte(plist.Term), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// LoadPostings loads a complete postings list for a term from BoltDB.
// Returns nil if the term is not found.
func (is *IndexStore) LoadPostings(term string) (*PersistedPostingsList, error) {
	var plist PersistedPostingsList
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(PostingsBucket)
		data := b.Get([]byte(term))
		if data == nil {
			return fmt.Errorf("postings not found for term: %s", term)
		}
		return json.Unmarshal(data, &plist)
	})

	if err != nil {
		return nil, err
	}
	return &plist, nil
}

// ListAllPostings returns all terms that have persisted postings.
func (is *IndexStore) ListAllPostings() ([]string, error) {
	var terms []string
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(PostingsBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			terms = append(terms, string(k))
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return terms, nil
}

// DeletePostings removes a postings list for a term from BoltDB.
func (is *IndexStore) DeletePostings(term string) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(PostingsBucket)
		return b.Delete([]byte(term))
	})
}

// SaveDocInfo saves document metadata to BoltDB.
func (is *IndexStore) SaveDocInfo(docInfo *PersistedDocInfo) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(DocInfoBucket)
		data, err := json.Marshal(docInfo)
		if err != nil {
			return fmt.Errorf("failed to marshal doc info: %w", err)
		}
		return b.Put([]byte(docInfo.DocID), data)
	})
}

// SaveDocInfoBatch saves multiple document metadata entries in a single transaction.
func (is *IndexStore) SaveDocInfoBatch(infos []*PersistedDocInfo) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(DocInfoBucket)
		for _, info := range infos {
			data, err := json.Marshal(info)
			if err != nil {
				return fmt.Errorf("failed to marshal doc info for %s: %w", info.DocID, err)
			}
			if err := b.Put([]byte(info.DocID), data); err != nil {
				return err
			}
		}
		return nil
	})
}

// LoadDocInfo loads document metadata from BoltDB.
// Returns nil if the document is not found.
func (is *IndexStore) LoadDocInfo(docID string) (*PersistedDocInfo, error) {
	var docInfo PersistedDocInfo
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(DocInfoBucket)
		data := b.Get([]byte(docID))
		if data == nil {
			return fmt.Errorf("doc info not found: %s", docID)
		}
		return json.Unmarshal(data, &docInfo)
	})

	if err != nil {
		return nil, err
	}
	return &docInfo, nil
}

// ListAllDocInfo returns all document IDs that have persisted metadata.
func (is *IndexStore) ListAllDocInfo() ([]string, error) {
	var docIDs []string
	err := is.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(DocInfoBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, _ []byte) error {
			docIDs = append(docIDs, string(k))
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	return docIDs, nil
}

// DeleteDocInfo removes document metadata from BoltDB.
func (is *IndexStore) DeleteDocInfo(docID string) error {
	return is.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(DocInfoBucket)
		return b.Delete([]byte(docID))
	})
}
