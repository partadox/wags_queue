package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// User represents a user in the system
type User struct {
	Username string `json:"username"`
	Key string `json:"-"` // Never return API key in JSON responses
}

// MessageStatus represents the possible statuses of a message
type MessageStatus string

// BulkMessageStatus represents the possible statuses of a bulk message
type BulkMessageStatus string

const (
	// Message statuses
	StatusPending    MessageStatus = "PENDING"
	StatusSent       MessageStatus = "SENT"
	StatusFailed     MessageStatus = "FAILED"
	StatusProcessing MessageStatus = "PROCESSING"

	// Bulk message statuses
	BulkStatusProcess BulkMessageStatus = "PROCESS"
	BulkStatusDone    BulkMessageStatus = "DONE"
	BulkStatusFailed  BulkMessageStatus = "FAILED"
)

// Message represents an individual message
type Message struct {
	ID                 int           `json:"id"`
	Sender             string        `json:"sender"`
	Recipient          string        `json:"recipient"`
	Status             MessageStatus `json:"status"`
	Type               string        `json:"type"` // Reference to bulk message ID if part of a bulk
	DTStore            time.Time     `json:"dt_store"`
	DTQueue            time.Time     `json:"dt_queue"`
	DTSend             sql.NullTime  `json:"dt_send,omitempty"`
	MessageContent     string        `json:"message"`
	ExternalAPIResponse sql.NullString `json:"external_api_response,omitempty"`
}

// MessageBulk represents a bulk message
type MessageBulk struct {
	ID        int              `json:"id"`
	Sender    string           `json:"sender"`
	Status    BulkMessageStatus `json:"status"`
	DTStore   time.Time        `json:"dt_store"`
	DTConvert sql.NullTime     `json:"dt_convert,omitempty"`
	Bulk      json.RawMessage  `json:"bulk"` // JSON data representing the bulk message
}

// MessageView is used for UI display of messages
type MessageView struct {
	ID              int     `json:"id"`
	Recipient       string  `json:"recipient"`
	Status          string  `json:"status"`
	BroadcastMessage string  `json:"broadcast_message"` // "YES" or "NO"
	DTStore         string  `json:"dt_store"`
	DTQueue         string  `json:"dt_queue"`
	DTSend          *string `json:"dt_send,omitempty"`
	Message         string  `json:"message"`
}

// MessageBulkView is used for UI display of bulk messages
type MessageBulkView struct {
	ID        int     `json:"id"`
	Sender    string  `json:"sender"`
	Status    string  `json:"status"`
	DTStore   string  `json:"dt_store"`
	DTConvert *string `json:"dt_convert,omitempty"`
}

// SingleMessageRequest represents a request to send a single message
type SingleMessageRequest struct {
	Recipient string    `json:"recipient"`
	Sender    string    `json:"sender"`
	Message   string    `json:"message"`
	DTStore   time.Time `json:"dt_store"`
}

// SingleMessageResponse represents a response to a single message request
type SingleMessageResponse struct {
	MessageID int           `json:"message_id"`
	Status    MessageStatus `json:"status"`
	DTQueue   time.Time     `json:"dt_queue"`
	Info      string        `json:"info"`
}

// BulkMessageRequest represents a request to send a bulk message
type BulkMessageRequest struct {
	Sender     string    `json:"sender"`
	Recipients []string  `json:"recipients"`
	Message    string    `json:"message"`
	DTStore    time.Time `json:"dt_store"`
}

// BulkMessageResponse represents a response to a bulk message request
type BulkMessageResponse struct {
	BulkMessageID int              `json:"bulk_message_id"`
	Status        BulkMessageStatus `json:"status"`
	Info          string           `json:"info"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Key string `json:"key"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Username string `json:"username"`
	Key      string `json:"key"`
	Message  string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
