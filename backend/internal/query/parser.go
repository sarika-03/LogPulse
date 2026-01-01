package query

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidQuery     = errors.New("invalid query syntax")
	ErrInvalidRegex     = errors.New("invalid regex pattern")
	ErrInvalidTimeRange = errors.New("invalid time range in aggregation")
)

// MatchOperator defines the type of label matching
type MatchOperator int

const (
	MatchEqual    MatchOperator = iota // =
	MatchNotEqual                      // !=
	MatchRegex                         // =~
	MatchNotRegex                      // !~
)

// LabelMatcher represents a single label match condition
type LabelMatcher struct {
	Name     string
	Value    string
	Operator MatchOperator
	Regex    *regexp.Regexp // Compiled regex for =~ and !~
}

// LineFilterOperator defines the type of line filtering
type LineFilterOperator int

const (
	LineContains    LineFilterOperator = iota // |=
	LineNotContains                           // !=
	LineRegex                                 // |~
	LineNotRegex                              // !~
)

// LineFilter represents a filter on log content
type LineFilter struct {
	Pattern  string
	Operator LineFilterOperator
	Regex    *regexp.Regexp // Compiled regex for |~ and !~
}

// AggregationType defines the type of aggregation
type AggregationType int

const (
	AggNone AggregationType = iota
	AggCountOverTime
	AggRate
	AggBytesOverTime
	AggBytesRate
	AggSum
	AggAvg
	AggMin
	AggMax
)

// Aggregation represents an aggregation function
type Aggregation struct {
	Type     AggregationType
	Duration int64 // Duration in seconds for range functions
	GroupBy  []string
}

// ParsedQuery represents a fully parsed LogQL query
type ParsedQuery struct {
	LabelMatchers []LabelMatcher
	LineFilters   []LineFilter
	Aggregation   *Aggregation
	RawQuery      string
}

var (
	// Matches {key="value", key2=~"regex.*"}
	queryRegex = regexp.MustCompile(`\{([^}]*)\}`)
	// Matches different operators: =, !=, =~, !~
	labelRegex = regexp.MustCompile(`(\w+)\s*(=~|!~|!=|=)\s*"([^"]*)"`)
	// Matches line filters: |= "text", != "text", |~ "regex", !~ "regex"
	lineFilterRegex = regexp.MustCompile(`(\|=|\|~|!=|!~)\s*"([^"]*)"`)
	// Matches aggregation functions: count_over_time({...}[5m])
	aggFuncRegex = regexp.MustCompile(`^(count_over_time|rate|bytes_over_time|bytes_rate|sum|avg|min|max)\s*\(`)
	// Matches time range: [5m], [1h], [30s]
	timeRangeRegex = regexp.MustCompile(`\[(\d+)([smhd])\]`)
	// Matches group by: by (label1, label2)
	groupByRegex = regexp.MustCompile(`by\s*\(([^)]+)\)`)
)

// ParseAdvancedQuery parses a LogQL query with full feature support
func ParseAdvancedQuery(query string) (*ParsedQuery, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &ParsedQuery{
			LabelMatchers: []LabelMatcher{},
			LineFilters:   []LineFilter{},
			RawQuery:      query,
		}, nil
	}

	parsed := &ParsedQuery{
		RawQuery: query,
	}

	// Check for aggregation function
	aggMatch := aggFuncRegex.FindStringSubmatch(query)
	if len(aggMatch) > 0 {
		agg, innerQuery, err := parseAggregation(query, aggMatch[1])
		if err != nil {
			return nil, err
		}
		parsed.Aggregation = agg
		query = innerQuery
	}

	// Extract label selectors
	labelMatchers, err := parseLabelMatchers(query)
	if err != nil {
		return nil, err
	}
	parsed.LabelMatchers = labelMatchers

	// Extract line filters (after the label selector)
	lineFilters, err := parseLineFilters(query)
	if err != nil {
		return nil, err
	}
	parsed.LineFilters = lineFilters

	return parsed, nil
}

// parseLabelMatchers extracts label matchers from query
func parseLabelMatchers(query string) ([]LabelMatcher, error) {
	matches := queryRegex.FindStringSubmatch(query)
	if len(matches) < 2 {
		// No label selector found, return empty
		return []LabelMatcher{}, nil
	}

	labelContent := matches[1]
	if strings.TrimSpace(labelContent) == "" {
		return []LabelMatcher{}, nil
	}

	var matchers []LabelMatcher
	labelMatches := labelRegex.FindAllStringSubmatch(labelContent, -1)

	for _, match := range labelMatches {
		if len(match) != 4 {
			continue
		}

		name := strings.TrimSpace(match[1])
		opStr := match[2]
		value := match[3]

		var op MatchOperator
		var regex *regexp.Regexp
		var err error

		switch opStr {
		case "=":
			op = MatchEqual
		case "!=":
			op = MatchNotEqual
		case "=~":
			op = MatchRegex
			regex, err = regexp.Compile(value)
			if err != nil {
				return nil, ErrInvalidRegex
			}
		case "!~":
			op = MatchNotRegex
			regex, err = regexp.Compile(value)
			if err != nil {
				return nil, ErrInvalidRegex
			}
		}

		matchers = append(matchers, LabelMatcher{
			Name:     name,
			Value:    value,
			Operator: op,
			Regex:    regex,
		})
	}

	return matchers, nil
}

// parseLineFilters extracts line filters from query
func parseLineFilters(query string) ([]LineFilter, error) {
	// Find everything after the label selector
	braceEnd := strings.LastIndex(query, "}")
	if braceEnd == -1 {
		return []LineFilter{}, nil
	}

	filterPart := query[braceEnd+1:]
	
	// Remove time range if present (for aggregations)
	if idx := strings.Index(filterPart, "["); idx != -1 {
		endIdx := strings.Index(filterPart, "]")
		if endIdx > idx {
			filterPart = filterPart[:idx] + filterPart[endIdx+1:]
		}
	}

	// Remove closing paren from aggregation if present
	filterPart = strings.TrimSuffix(strings.TrimSpace(filterPart), ")")

	var filters []LineFilter
	filterMatches := lineFilterRegex.FindAllStringSubmatch(filterPart, -1)

	for _, match := range filterMatches {
		if len(match) != 3 {
			continue
		}

		opStr := match[1]
		pattern := match[2]

		var op LineFilterOperator
		var regex *regexp.Regexp
		var err error

		switch opStr {
		case "|=":
			op = LineContains
		case "!=":
			op = LineNotContains
		case "|~":
			op = LineRegex
			regex, err = regexp.Compile(pattern)
			if err != nil {
				return nil, ErrInvalidRegex
			}
		case "!~":
			op = LineNotRegex
			regex, err = regexp.Compile(pattern)
			if err != nil {
				return nil, ErrInvalidRegex
			}
		}

		filters = append(filters, LineFilter{
			Pattern:  pattern,
			Operator: op,
			Regex:    regex,
		})
	}

	return filters, nil
}

// parseAggregation extracts aggregation function and returns inner query
func parseAggregation(query string, funcName string) (*Aggregation, string, error) {
	agg := &Aggregation{}

	switch funcName {
	case "count_over_time":
		agg.Type = AggCountOverTime
	case "rate":
		agg.Type = AggRate
	case "bytes_over_time":
		agg.Type = AggBytesOverTime
	case "bytes_rate":
		agg.Type = AggBytesRate
	case "sum":
		agg.Type = AggSum
	case "avg":
		agg.Type = AggAvg
	case "min":
		agg.Type = AggMin
	case "max":
		agg.Type = AggMax
	}

	// Extract time range [5m], [1h], etc.
	timeMatch := timeRangeRegex.FindStringSubmatch(query)
	if len(timeMatch) == 3 {
		value, _ := strconv.ParseInt(timeMatch[1], 10, 64)
		unit := timeMatch[2]

		switch unit {
		case "s":
			agg.Duration = value
		case "m":
			agg.Duration = value * 60
		case "h":
			agg.Duration = value * 3600
		case "d":
			agg.Duration = value * 86400
		}
	}

	// Extract group by labels
	groupByMatch := groupByRegex.FindStringSubmatch(query)
	if len(groupByMatch) == 2 {
		labels := strings.Split(groupByMatch[1], ",")
		for _, l := range labels {
			agg.GroupBy = append(agg.GroupBy, strings.TrimSpace(l))
		}
	}

	// Extract inner query (content within the aggregation function)
	// Find the label selector within
	innerQuery := query
	if idx := strings.Index(query, "{"); idx != -1 {
		// Find matching closing brace
		braceCount := 0
		endIdx := idx
		for i := idx; i < len(query); i++ {
			if query[i] == '{' {
				braceCount++
			} else if query[i] == '}' {
				braceCount--
				if braceCount == 0 {
					endIdx = i
					break
				}
			}
		}
		innerQuery = query[idx : endIdx+1]
		
		// Also capture line filters if present
		afterBrace := query[endIdx+1:]
		if filterIdx := strings.Index(afterBrace, "|"); filterIdx != -1 {
			// Find end of line filter
			endFilter := strings.Index(afterBrace, "[")
			if endFilter == -1 {
				endFilter = strings.Index(afterBrace, ")")
			}
			if endFilter > filterIdx {
				innerQuery += afterBrace[:endFilter]
			}
		}
	}

	return agg, innerQuery, nil
}

// ParseQuery parses a simple LogQL-style query string into label matchers (backwards compatible)
func ParseQuery(query string) (map[string]string, error) {
	parsed, err := ParseAdvancedQuery(query)
	if err != nil {
		return nil, err
	}

	labels := make(map[string]string)
	for _, m := range parsed.LabelMatchers {
		// For backward compatibility, only return exact matches
		if m.Operator == MatchEqual {
			labels[m.Name] = m.Value
		}
	}

	return labels, nil
}

// BuildQuery creates a query string from labels
func BuildQuery(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+`="`+v+`"`)
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

// Match checks if a set of labels matches the given matchers
func (m *LabelMatcher) Match(labels map[string]string) bool {
	value, exists := labels[m.Name]

	switch m.Operator {
	case MatchEqual:
		return exists && value == m.Value
	case MatchNotEqual:
		return !exists || value != m.Value
	case MatchRegex:
		return exists && m.Regex != nil && m.Regex.MatchString(value)
	case MatchNotRegex:
		return !exists || (m.Regex != nil && !m.Regex.MatchString(value))
	}

	return false
}

// Match checks if a log line matches the filter
func (f *LineFilter) Match(line string) bool {
	switch f.Operator {
	case LineContains:
		return strings.Contains(line, f.Pattern)
	case LineNotContains:
		return !strings.Contains(line, f.Pattern)
	case LineRegex:
		return f.Regex != nil && f.Regex.MatchString(line)
	case LineNotRegex:
		return f.Regex == nil || !f.Regex.MatchString(line)
	}

	return true
}

// MatchLabels checks if all matchers match the given labels
func (p *ParsedQuery) MatchLabels(labels map[string]string) bool {
	for _, m := range p.LabelMatchers {
		if !m.Match(labels) {
			return false
		}
	}
	return true
}

// MatchLine checks if all line filters match the given line
func (p *ParsedQuery) MatchLine(line string) bool {
	for _, f := range p.LineFilters {
		if !f.Match(line) {
			return false
		}
	}
	return true
}
