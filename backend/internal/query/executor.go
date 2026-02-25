package query

import (
	"sort"
	"time"

	"github.com/logpulse/backend/internal/index"
	"github.com/logpulse/backend/internal/models"
	"github.com/logpulse/backend/internal/storage"
)

// Executor handles query execution
type Executor struct {
	index  *index.Index
	reader *storage.Reader
}

// NewExecutor creates a new query executor
func NewExecutor(idx *index.Index, reader *storage.Reader) *Executor {
	return &Executor{
		index:  idx,
		reader: reader,
	}
}

// QueryResult contains query results and stats
type QueryResult struct {
	Logs        []LogResponse        `json:"logs"`
	Stats       QueryStats           `json:"stats"`
	Aggregation *AggregationResult   `json:"aggregation,omitempty"`
	Total       int                  `json:"total"`       // Total matching logs
	HasMore     bool                 `json:"hasMore"`     // Whether more logs exist
	Offset      int                  `json:"offset"`      // Current offset
}

type LogResponse struct {
	ID        string            `json:"id"`
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Labels    map[string]string `json:"labels"`
}

type QueryStats struct {
	QueriedChunks int `json:"queriedChunks"`
	ScannedLines  int `json:"scannedLines"`
	MatchedLines  int `json:"matchedLines"`
	ExecutionTime int `json:"executionTime"` // milliseconds
}

// AggregationResult contains aggregation computation results
type AggregationResult struct {
	Type   string                   `json:"type"`
	Value  float64                  `json:"value,omitempty"`
	Series []AggregationSeriesPoint `json:"series,omitempty"`
	Groups []AggregationGroup       `json:"groups,omitempty"`
}

type AggregationSeriesPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type AggregationGroup struct {
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
}

// Execute runs a query and returns matching logs with pagination support
func (e *Executor) Execute(queryStr string, startTime, endTime time.Time, limit, offset int) (*QueryResult, error) {
	startExec := time.Now()

	// Parse query with advanced features
	parsed, err := ParseAdvancedQuery(queryStr)
	if err != nil {
		return nil, err
	}

	// Get simple labels for chunk lookup (exact matches only)
	simpleLabels := make(map[string]string)
	for _, m := range parsed.LabelMatchers {
		if m.Operator == MatchEqual {
			simpleLabels[m.Name] = m.Value
		}
	}

	// Find matching chunks
	chunkIDs := e.index.FindChunks(simpleLabels, startTime, endTime)

	stats := QueryStats{
		QueriedChunks: len(chunkIDs),
	}

	var allLogs []models.LogEntry

	// Optimization: For non-aggregation queries with pagination, we don't need all logs
	// Stop collecting once we have offset + limit + some buffer for sorting
	maxLogsToCollect := offset + limit
	if maxLogsToCollect <= 0 {
		maxLogsToCollect = 1000 // Reasonable default
	}

	// Read logs from each chunk
	for _, chunkID := range chunkIDs {
		meta := e.index.GetChunkMeta(chunkID)
		if meta == nil {
			continue
		}

		entries, scanned, err := e.reader.ReadChunkFiltered(meta.Labels, chunkID, startTime, endTime)
		if err != nil {
			continue
		}

		stats.ScannedLines += scanned

		// Apply advanced filters
		for _, entry := range entries {
			// Check label matchers (including regex)
			if !parsed.MatchLabels(entry.Labels) {
				continue
			}

			// Check line filters
			if !parsed.MatchLine(entry.Line) {
				continue
			}

			allLogs = append(allLogs, entry)

			// Early exit optimization for pagination queries
			// (but still read all for accurate total count in aggregations)
			if parsed.Aggregation == nil && len(allLogs) > maxLogsToCollect*2 {
				// We have enough to satisfy pagination + some buffer for sorting
				break
			}
		}

		// If we have enough logs, stop reading more chunks
		if parsed.Aggregation == nil && len(allLogs) > maxLogsToCollect*2 {
			break
		}
	}

	stats.MatchedLines = len(allLogs)

	// Sort by timestamp descending (newest first)
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp.After(allLogs[j].Timestamp)
	})

	// Handle aggregations
	var aggResult *AggregationResult
	if parsed.Aggregation != nil {
		aggResult = e.computeAggregation(parsed.Aggregation, allLogs, startTime, endTime)
	}

	// Calculate total before pagination
	totalLogs := len(allLogs)

	// Apply pagination (only for non-aggregation queries)
	var paginatedLogs []models.LogEntry
	hasMore := false

	if parsed.Aggregation == nil {
		// Calculate offsets for pagination
		if offset < 0 {
			offset = 0
		}
		if limit <= 0 {
			limit = 100 // Default limit
		}

		// Check if there are more logs after this page
		if offset+limit < totalLogs {
			hasMore = true
		}

		// Apply offset and limit
		end := offset + limit
		if end > totalLogs {
			end = totalLogs
		}

		if offset < totalLogs {
			paginatedLogs = allLogs[offset:end]
		}
	} else {
		// For aggregations, return all logs without pagination
		paginatedLogs = allLogs
	}

	// Convert to response format
	logs := make([]LogResponse, len(paginatedLogs))
	for i, entry := range paginatedLogs {
		level := "info"
		if l, ok := entry.Labels["level"]; ok {
			level = l
		}

		logs[i] = LogResponse{
			ID:        entry.ID,
			Timestamp: entry.Timestamp.Format(time.RFC3339Nano),
			Level:     level,
			Message:   entry.Line,
			Labels:    entry.Labels,
		}
	}

	stats.ExecutionTime = int(time.Since(startExec).Milliseconds())

	return &QueryResult{
		Logs:        logs,
		Stats:       stats,
		Aggregation: aggResult,
		Total:       totalLogs,
		HasMore:     hasMore,
		Offset:      offset,
	}, nil
}

// computeAggregation computes the aggregation result
func (e *Executor) computeAggregation(agg *Aggregation, logs []models.LogEntry, startTime, endTime time.Time) *AggregationResult {
	result := &AggregationResult{}

	switch agg.Type {
	case AggCountOverTime:
		result.Type = "count_over_time"
		result.Value = float64(len(logs))
		result.Series = e.computeTimeSeries(logs, agg.Duration, startTime, endTime, countAggFunc)

	case AggRate:
		result.Type = "rate"
		duration := endTime.Sub(startTime).Seconds()
		if duration > 0 {
			result.Value = float64(len(logs)) / duration
		}
		result.Series = e.computeTimeSeries(logs, agg.Duration, startTime, endTime, rateAggFunc)

	case AggBytesOverTime:
		result.Type = "bytes_over_time"
		var totalBytes int64
		for _, log := range logs {
			totalBytes += int64(len(log.Line))
		}
		result.Value = float64(totalBytes)
		result.Series = e.computeBytesSeries(logs, agg.Duration, startTime, endTime, false)

	case AggBytesRate:
		result.Type = "bytes_rate"
		var totalBytes int64
		for _, log := range logs {
			totalBytes += int64(len(log.Line))
		}
		duration := endTime.Sub(startTime).Seconds()
		if duration > 0 {
			result.Value = float64(totalBytes) / duration
		}
		result.Series = e.computeBytesSeries(logs, agg.Duration, startTime, endTime, true)

	case AggSum, AggAvg, AggMin, AggMax:
		result.Type = aggTypeToString(agg.Type)
		if len(agg.GroupBy) > 0 {
			result.Groups = e.computeGroupedAggregation(agg, logs)
		} else {
			result.Value = float64(len(logs))
		}
	}

	return result
}

func aggTypeToString(t AggregationType) string {
	switch t {
	case AggSum:
		return "sum"
	case AggAvg:
		return "avg"
	case AggMin:
		return "min"
	case AggMax:
		return "max"
	default:
		return "unknown"
	}
}

type aggFunc func(count int, duration float64) float64

func countAggFunc(count int, _ float64) float64 {
	return float64(count)
}

func rateAggFunc(count int, duration float64) float64 {
	if duration > 0 {
		return float64(count) / duration
	}
	return 0
}

// computeTimeSeries computes time series data points
func (e *Executor) computeTimeSeries(logs []models.LogEntry, stepSeconds int64, startTime, endTime time.Time, fn aggFunc) []AggregationSeriesPoint {
	if stepSeconds <= 0 {
		stepSeconds = 60 // Default to 1 minute
	}

	step := time.Duration(stepSeconds) * time.Second
	var series []AggregationSeriesPoint

	// Create buckets
	for t := startTime; t.Before(endTime); t = t.Add(step) {
		bucketEnd := t.Add(step)
		if bucketEnd.After(endTime) {
			bucketEnd = endTime
		}

		count := 0
		for _, log := range logs {
			if !log.Timestamp.Before(t) && log.Timestamp.Before(bucketEnd) {
				count++
			}
		}

		duration := bucketEnd.Sub(t).Seconds()
		series = append(series, AggregationSeriesPoint{
			Timestamp: t.Format(time.RFC3339),
			Value:     fn(count, duration),
		})
	}

	return series
}

// computeBytesSeries computes bytes time series
func (e *Executor) computeBytesSeries(logs []models.LogEntry, stepSeconds int64, startTime, endTime time.Time, asRate bool) []AggregationSeriesPoint {
	if stepSeconds <= 0 {
		stepSeconds = 60
	}

	step := time.Duration(stepSeconds) * time.Second
	var series []AggregationSeriesPoint

	for t := startTime; t.Before(endTime); t = t.Add(step) {
		bucketEnd := t.Add(step)
		if bucketEnd.After(endTime) {
			bucketEnd = endTime
		}

		var bytes int64
		for _, log := range logs {
			if !log.Timestamp.Before(t) && log.Timestamp.Before(bucketEnd) {
				bytes += int64(len(log.Line))
			}
		}

		value := float64(bytes)
		if asRate {
			duration := bucketEnd.Sub(t).Seconds()
			if duration > 0 {
				value = value / duration
			}
		}

		series = append(series, AggregationSeriesPoint{
			Timestamp: t.Format(time.RFC3339),
			Value:     value,
		})
	}

	return series
}

// computeGroupedAggregation computes aggregation grouped by labels
func (e *Executor) computeGroupedAggregation(agg *Aggregation, logs []models.LogEntry) []AggregationGroup {
	groups := make(map[string]*AggregationGroup)

	for _, log := range logs {
		// Build group key
		key := ""
		groupLabels := make(map[string]string)
		for _, label := range agg.GroupBy {
			if val, ok := log.Labels[label]; ok {
				key += label + "=" + val + ","
				groupLabels[label] = val
			}
		}

		if _, exists := groups[key]; !exists {
			groups[key] = &AggregationGroup{
				Labels: groupLabels,
				Value:  0,
			}
		}

		switch agg.Type {
		case AggSum, AggCountOverTime:
			groups[key].Value++
		case AggAvg:
			groups[key].Value++ // Will divide later
		}
	}

	// For avg, we need to track counts separately
	// For simplicity, just return counts for now

	result := make([]AggregationGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, *g)
	}

	return result
}
