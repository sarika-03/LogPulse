package api

import (
	"encoding/json"
	"net/http"

	"github.com/logpulse/backend/internal/query"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Query-related errors
	ErrorCodeBadQuery         ErrorCode = "BAD_QUERY"
	ErrorCodeInvalidRegex     ErrorCode = "INVALID_REGEX"
	ErrorCodeInvalidTimeRange ErrorCode = "INVALID_TIME_RANGE"

	// Input validation errors
	ErrorCodeInvalidJSON     ErrorCode = "INVALID_JSON"
	ErrorCodeValidationError ErrorCode = "VALIDATION_ERROR"
	ErrorCodeMissingField    ErrorCode = "MISSING_FIELD"

	// Server errors
	ErrorCodeInternalError  ErrorCode = "INTERNAL_ERROR"
	ErrorCodeIngestionError ErrorCode = "INGESTION_ERROR"

	// Connection errors
	ErrorCodeConnectionError ErrorCode = "CONNECTION_ERROR"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string    `json:"error"`
	Code    ErrorCode `json:"code"`
	Details string    `json:"details,omitempty"`
}

// WriteErrorResponse writes a structured error response to the HTTP response writer
func WriteErrorResponse(w http.ResponseWriter, statusCode int, code ErrorCode, message string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	json.NewEncoder(w).Encode(errorResp)
}

// WriteQueryError writes a query-specific error response
func WriteQueryError(w http.ResponseWriter, err error, details string) {
	var code ErrorCode
	var message string
	var errorDetails string

	// Check if it's a detailed QueryError
	if queryErr, ok := err.(*query.QueryError); ok {
		switch queryErr.Type {
		case "syntax":
			code = ErrorCodeBadQuery
			message = "Invalid query syntax"
			errorDetails = queryErr.Details
		case "regex":
			code = ErrorCodeInvalidRegex
			message = "Invalid regex pattern"
			errorDetails = queryErr.Details
		default:
			code = ErrorCodeBadQuery
			message = queryErr.Message
			errorDetails = queryErr.Details
		}
	} else {
		// Handle legacy errors
		switch err.Error() {
		case "invalid query syntax":
			code = ErrorCodeBadQuery
			message = "Invalid query syntax"
		case "invalid regex pattern":
			code = ErrorCodeInvalidRegex
			message = "Invalid regex pattern"
		case "invalid time range in aggregation":
			code = ErrorCodeInvalidTimeRange
			message = "Invalid time range in aggregation"
		default:
			code = ErrorCodeBadQuery
			message = "Query error"
			if details == "" {
				errorDetails = err.Error()
			}
		}
	}

	if errorDetails == "" && details != "" {
		errorDetails = details
	}

	WriteErrorResponse(w, http.StatusBadRequest, code, message, errorDetails)
}

// WriteValidationError writes a validation error response
func WriteValidationError(w http.ResponseWriter, field string, message string) {
	details := ""
	if field != "" {
		details = "Field: " + field
	}
	WriteErrorResponse(w, http.StatusBadRequest, ErrorCodeValidationError, message, details)
}

// WriteJSONError writes a JSON parsing error response
func WriteJSONError(w http.ResponseWriter, err error) {
	WriteErrorResponse(w, http.StatusBadRequest, ErrorCodeInvalidJSON, "Invalid JSON format", err.Error())
}

// WriteInternalError writes an internal server error response
func WriteInternalError(w http.ResponseWriter, message string, details string) {
	WriteErrorResponse(w, http.StatusInternalServerError, ErrorCodeInternalError, message, details)
}
