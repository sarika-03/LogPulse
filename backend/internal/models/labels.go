package models

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Labels is a map of label key-value pairs
type Labels map[string]string

// Hash generates a unique hash for a label set
func (l Labels) Hash() string {
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(l[k])
		sb.WriteString(",")
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:8])
}

// ToPath converts labels to a directory path
func (l Labels) ToPath() string {
	keys := make([]string, 0, len(l))
	for k := range l {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(l))
	for _, k := range keys {
		parts = append(parts, k+"="+l[k])
	}

	return strings.Join(parts, "_")
}

// Match checks if labels match a query
func (l Labels) Match(query Labels) bool {
	for k, v := range query {
		if l[k] != v {
			return false
		}
	}
	return true
}
