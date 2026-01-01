package index

import (
	"sync"
	"time"

	"github.com/logpulse/backend/internal/models"
)

// Index is an in-memory label-to-chunk mapping
type Index struct {
	mu sync.RWMutex

	// labelIndex maps label hash -> list of chunk IDs
	labelIndex map[string][]string

	// chunkMeta stores chunk metadata by ID
	chunkMeta map[string]*models.ChunkMeta

	// labelKeys tracks all unique label keys
	labelKeys map[string]struct{}

	// labelValues tracks all values for each label key
	labelValues map[string]map[string]struct{}
}

// NewIndex creates a new in-memory index
func NewIndex() *Index {
	return &Index{
		labelIndex:  make(map[string][]string),
		chunkMeta:   make(map[string]*models.ChunkMeta),
		labelKeys:   make(map[string]struct{}),
		labelValues: make(map[string]map[string]struct{}),
	}
}

// AddChunk registers a new chunk in the index
func (idx *Index) AddChunk(chunkID string, labels map[string]string, startTime, endTime time.Time, entryCount int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Create label hash
	l := models.Labels(labels)
	hash := l.Hash()

	// Add to label index
	idx.labelIndex[hash] = append(idx.labelIndex[hash], chunkID)

	// Store chunk metadata
	idx.chunkMeta[chunkID] = &models.ChunkMeta{
		ID:         chunkID,
		Labels:     labels,
		StartTime:  startTime.Unix(),
		EndTime:    endTime.Unix(),
		EntryCount: entryCount,
	}

	// Track label keys and values
	for k, v := range labels {
		idx.labelKeys[k] = struct{}{}
		if idx.labelValues[k] == nil {
			idx.labelValues[k] = make(map[string]struct{})
		}
		idx.labelValues[k][v] = struct{}{}
	}
}

// FindChunks returns chunk IDs matching the query labels and time range
func (idx *Index) FindChunks(query map[string]string, startTime, endTime time.Time) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var matchingChunks []string
	startUnix := startTime.Unix()
	endUnix := endTime.Unix()

	// Iterate all chunks and check matches
	for chunkID, meta := range idx.chunkMeta {
		// Check time overlap
		if meta.EndTime < startUnix || meta.StartTime > endUnix {
			continue
		}

		// Check label match
		if models.Labels(meta.Labels).Match(models.Labels(query)) {
			matchingChunks = append(matchingChunks, chunkID)
		}
	}

	return matchingChunks
}

// GetChunkMeta returns metadata for a specific chunk
func (idx *Index) GetChunkMeta(chunkID string) *models.ChunkMeta {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.chunkMeta[chunkID]
}

// GetAllLabels returns all unique label keys
func (idx *Index) GetAllLabels() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	keys := make([]string, 0, len(idx.labelKeys))
	for k := range idx.labelKeys {
		keys = append(keys, k)
	}
	return keys
}

// GetLabelValues returns all values for a label key
func (idx *Index) GetLabelValues(labelKey string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	values := make([]string, 0)
	if valMap, ok := idx.labelValues[labelKey]; ok {
		for v := range valMap {
			values = append(values, v)
		}
	}
	return values
}

// RemoveChunk removes a chunk from the index
func (idx *Index) RemoveChunk(chunkID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	meta, exists := idx.chunkMeta[chunkID]
	if !exists {
		return
	}

	// Remove from label index
	hash := models.Labels(meta.Labels).Hash()
	chunks := idx.labelIndex[hash]
	for i, id := range chunks {
		if id == chunkID {
			idx.labelIndex[hash] = append(chunks[:i], chunks[i+1:]...)
			break
		}
	}

	// Remove chunk metadata
	delete(idx.chunkMeta, chunkID)
}

// Stats returns index statistics
func (idx *Index) Stats() (chunkCount int, labelCount int) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.chunkMeta), len(idx.labelKeys)
}
