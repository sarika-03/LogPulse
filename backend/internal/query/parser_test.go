package query

import (
	"testing"
)

func TestParseAdvancedQuery_ExactMatch(t *testing.T) {
	query := `{app="nginx", level="error"}`
	parsed, err := ParseAdvancedQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.LabelMatchers) != 2 {
		t.Fatalf("expected 2 matchers, got %d", len(parsed.LabelMatchers))
	}

	for _, m := range parsed.LabelMatchers {
		if m.Operator != MatchEqual {
			t.Errorf("expected MatchEqual operator, got %v", m.Operator)
		}
	}
}

func TestParseAdvancedQuery_RegexMatch(t *testing.T) {
	query := `{app=~"nginx.*", level!="debug"}`
	parsed, err := ParseAdvancedQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.LabelMatchers) != 2 {
		t.Fatalf("expected 2 matchers, got %d", len(parsed.LabelMatchers))
	}

	foundRegex := false
	foundNotEqual := false
	for _, m := range parsed.LabelMatchers {
		if m.Name == "app" && m.Operator == MatchRegex {
			foundRegex = true
			if m.Regex == nil {
				t.Error("expected compiled regex for app matcher")
			}
		}
		if m.Name == "level" && m.Operator == MatchNotEqual {
			foundNotEqual = true
		}
	}

	if !foundRegex {
		t.Error("regex matcher not found")
	}
	if !foundNotEqual {
		t.Error("not equal matcher not found")
	}
}

func TestParseAdvancedQuery_LineFilters(t *testing.T) {
	query := `{app="nginx"} |= "error" |~ "timeout.*"`
	parsed, err := ParseAdvancedQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.LineFilters) != 2 {
		t.Fatalf("expected 2 line filters, got %d", len(parsed.LineFilters))
	}

	if parsed.LineFilters[0].Operator != LineContains {
		t.Errorf("expected LineContains, got %v", parsed.LineFilters[0].Operator)
	}

	if parsed.LineFilters[1].Operator != LineRegex {
		t.Errorf("expected LineRegex, got %v", parsed.LineFilters[1].Operator)
	}
}

func TestParseAdvancedQuery_Aggregation(t *testing.T) {
	query := `count_over_time({app="nginx"}[5m])`
	parsed, err := ParseAdvancedQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Aggregation == nil {
		t.Fatal("expected aggregation to be parsed")
	}

	if parsed.Aggregation.Type != AggCountOverTime {
		t.Errorf("expected AggCountOverTime, got %v", parsed.Aggregation.Type)
	}

	if parsed.Aggregation.Duration != 300 { // 5 minutes in seconds
		t.Errorf("expected duration 300, got %d", parsed.Aggregation.Duration)
	}
}

func TestParseAdvancedQuery_RateAggregation(t *testing.T) {
	query := `rate({app="nginx", level="error"}[1h])`
	parsed, err := ParseAdvancedQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Aggregation == nil {
		t.Fatal("expected aggregation to be parsed")
	}

	if parsed.Aggregation.Type != AggRate {
		t.Errorf("expected AggRate, got %v", parsed.Aggregation.Type)
	}

	if parsed.Aggregation.Duration != 3600 { // 1 hour in seconds
		t.Errorf("expected duration 3600, got %d", parsed.Aggregation.Duration)
	}
}

func TestLabelMatcher_Match(t *testing.T) {
	tests := []struct {
		name     string
		matcher  LabelMatcher
		labels   map[string]string
		expected bool
	}{
		{
			name:     "exact match success",
			matcher:  LabelMatcher{Name: "app", Value: "nginx", Operator: MatchEqual},
			labels:   map[string]string{"app": "nginx"},
			expected: true,
		},
		{
			name:     "exact match failure",
			matcher:  LabelMatcher{Name: "app", Value: "nginx", Operator: MatchEqual},
			labels:   map[string]string{"app": "apache"},
			expected: false,
		},
		{
			name:     "not equal success",
			matcher:  LabelMatcher{Name: "level", Value: "debug", Operator: MatchNotEqual},
			labels:   map[string]string{"level": "error"},
			expected: true,
		},
		{
			name:     "not equal failure",
			matcher:  LabelMatcher{Name: "level", Value: "debug", Operator: MatchNotEqual},
			labels:   map[string]string{"level": "debug"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.matcher.Match(tt.labels)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestLineFilter_Match(t *testing.T) {
	tests := []struct {
		name     string
		filter   LineFilter
		line     string
		expected bool
	}{
		{
			name:     "contains success",
			filter:   LineFilter{Pattern: "error", Operator: LineContains},
			line:     "this is an error message",
			expected: true,
		},
		{
			name:     "contains failure",
			filter:   LineFilter{Pattern: "error", Operator: LineContains},
			line:     "this is a success message",
			expected: false,
		},
		{
			name:     "not contains success",
			filter:   LineFilter{Pattern: "debug", Operator: LineNotContains},
			line:     "this is an error message",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.Match(tt.line)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseQuery_BackwardCompatibility(t *testing.T) {
	query := `{app="nginx", level="error"}`
	labels, err := ParseQuery(query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if labels["app"] != "nginx" {
		t.Errorf("expected app=nginx, got %s", labels["app"])
	}
	if labels["level"] != "error" {
		t.Errorf("expected level=error, got %s", labels["level"])
	}
}

func TestParseAdvancedQuery_InvalidRegex(t *testing.T) {
	query := `{app=~"[invalid"}`
	_, err := ParseAdvancedQuery(query)
	if err != ErrInvalidRegex {
		t.Errorf("expected ErrInvalidRegex, got %v", err)
	}
}

func TestParseAdvancedQuery_EmptyQuery(t *testing.T) {
	parsed, err := ParseAdvancedQuery("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.LabelMatchers) != 0 {
		t.Error("expected empty matchers for empty query")
	}
}
