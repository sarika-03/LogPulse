package index

import (
	"encoding/json"
	"github.com/boltdb/bolt"
	"github.com/logpulse/backend/internal/models"
)

// NOTE: BoltDB persistence will be added later
// For now, the index is purely in-memory

// PersistIndex saves the index to BoltDB
func (idx *Index) PersistIndex(dbPath string) error {
		// Implement BoltDB persistence
		db, err := bolt.Open(dbPath, 0600, nil)
		if err != nil {
			return err
		}
		defer db.Close()

		err = db.Update(func(tx *bolt.Tx) error {
			chunks, err := tx.CreateBucketIfNotExists([]byte("chunks"))
			if err != nil {
				return err
			}
			labels, err := tx.CreateBucketIfNotExists([]byte("labels"))
			if err != nil {
				return err
			}
			// Store chunk metadata
			for id, meta := range idx.chunkMeta {
				buf, err := json.Marshal(meta)
				if err != nil {
					return err
				}
				if err := chunks.Put([]byte(id), buf); err != nil {
					return err
				}
			}
			// Store label key-value mappings
			for key, values := range idx.labelValues {
				vlist := make([]string, 0, len(values))
				for v := range values {
					vlist = append(vlist, v)
				}
				buf, err := json.Marshal(vlist)
				if err != nil {
					return err
				}
				if err := labels.Put([]byte(key), buf); err != nil {
					return err
				}
			}
			return nil
		})
		return err
}

// LoadIndex loads the index from BoltDB
func LoadIndex(dbPath string) (*Index, error) {
		// Implement BoltDB loading
		db, err := bolt.Open(dbPath, 0600, nil)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		idx := NewIndex()
		err = db.View(func(tx *bolt.Tx) error {
			chunks := tx.Bucket([]byte("chunks"))
			if chunks != nil {
				err := chunks.ForEach(func(k, v []byte) error {
					var meta models.ChunkMeta
					if err := json.Unmarshal(v, &meta); err != nil {
						return err
					}
					idx.chunkMeta[string(k)] = &meta
					return nil
				})
				if err != nil {
					return err
				}
			}
			labels := tx.Bucket([]byte("labels"))
			if labels != nil {
				err := labels.ForEach(func(k, v []byte) error {
					var vlist []string
					if err := json.Unmarshal(v, &vlist); err != nil {
						return err
					}
					m := make(map[string]struct{})
					for _, val := range vlist {
						m[val] = struct{}{}
					}
					idx.labelValues[string(k)] = m
					return nil
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
		return idx, err
}
