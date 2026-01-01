package ingest

import (
	"errors"
	"strings"

	"github.com/logpulse/backend/internal/models"
)

var (
	ErrEmptyLabels  = errors.New("labels cannot be empty")
	ErrEmptyEntries = errors.New("entries cannot be empty")
	ErrInvalidLabel = errors.New("invalid label key or value")
)

// ValidateStream validates a log stream
func ValidateStream(stream *models.Stream) error {
	if len(stream.Labels) == 0 {
		return ErrEmptyLabels
	}

	if len(stream.Entries) == 0 {
		return ErrEmptyEntries
	}

	for key, value := range stream.Labels {
		if err := validateLabelKey(key); err != nil {
			return err
		}
		if err := validateLabelValue(value); err != nil {
			return err
		}
	}

	return nil
}

// validateLabelKey checks if a label key is valid
func validateLabelKey(key string) error {
	if len(key) == 0 || len(key) > 128 {
		return ErrInvalidLabel
	}

	// Label keys must start with letter or underscore
	first := key[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return ErrInvalidLabel
	}

	// Rest can be alphanumeric or underscore
	for _, c := range key[1:] {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return ErrInvalidLabel
		}
	}

	return nil
}

// validateLabelValue checks if a label value is valid
func validateLabelValue(value string) error {
	if len(value) == 0 || len(value) > 2048 {
		return ErrInvalidLabel
	}

	// Value cannot contain newlines
	if strings.Contains(value, "\n") {
		return ErrInvalidLabel
	}

	return nil
}

// ValidateIngestRequest validates the entire ingest request
func ValidateIngestRequest(req *models.IngestRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	if len(req.Streams) == 0 {
		return errors.New("streams cannot be empty")
	}

	for _, stream := range req.Streams {
		if err := ValidateStream(&stream); err != nil {
			return err
		}
	}

	return nil
}
