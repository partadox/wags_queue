package worker

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/partadox/wags_queue/internal/config"
	"github.com/partadox/wags_queue/internal/models"
)

// MessageWorker handles the processing of queued messages
type MessageWorker struct {
	db          *sql.DB
	externalAPI config.ExternalAPIConfig
	client      *http.Client
	done        chan struct{}
	wg          sync.WaitGroup
}

// NewMessageWorker creates a new message worker
func NewMessageWorker(db *sql.DB, externalAPI config.ExternalAPIConfig) *MessageWorker {
	return &MessageWorker{
		db:          db,
		externalAPI: externalAPI,
		client:      &http.Client{Timeout: 30 * time.Second},
		done:        make(chan struct{}),
	}
}

// Run starts the message worker
func (w *MessageWorker) Run() {
	w.wg.Add(1)
	defer w.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processMessages()
		case <-w.done:
			log.Println("Message worker is shutting down...")
			return
		}
	}
}

// Stop signals the worker to stop processing messages
func (w *MessageWorker) Stop() {
	close(w.done)
	w.wg.Wait()
	log.Println("Message worker stopped")
}

// processMessages processes pending messages
func (w *MessageWorker) processMessages() {
	// Begin transaction
	tx, err := w.db.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			log.Printf("Panic in processMessages: %v", err)
		}
	}()

	// Get a batch of messages to process
	rows, err := tx.Query(`
		SELECT id, sender, recipient, message 
		FROM message 
		WHERE status = ? 
		ORDER BY dt_queue ASC 
		LIMIT 10
	`, models.StatusPending)

	if err != nil {
		tx.Rollback()
		log.Printf("Error querying messages: %v", err)
		return
	}
	defer rows.Close()

	messagesToProcess := make([]models.Message, 0)
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.Sender, &msg.Recipient, &msg.MessageContent); err != nil {
			log.Printf("Error scanning message row: %v", err)
			continue
		}
		messagesToProcess = append(messagesToProcess, msg)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		log.Printf("Error iterating message rows: %v", err)
		return
	}

	// If no messages to process, commit empty transaction and return
	if len(messagesToProcess) == 0 {
		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
		}
		return
	}

	// Update messages to PROCESSING status
	for _, msg := range messagesToProcess {
		_, err := tx.Exec(`
			UPDATE message 
			SET status = ? 
			WHERE id = ?
		`, models.StatusProcessing, msg.ID)

		if err != nil {
			log.Printf("Error updating message status to PROCESSING (ID: %d): %v", msg.ID, err)
			// Continue with other messages
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}

	// Process each message
	for _, msg := range messagesToProcess {
		w.sendMessage(msg)
	}
}

// sendMessage sends a message to the external API
func (w *MessageWorker) sendMessage(msg models.Message) {
	// Prepare request to external API
	type ExternalAPIRequest struct {
		Recipient string `json:"recipient"`
		Message   string `json:"message"`
		// Sender    string `json:"from"`
	}

	reqBody := ExternalAPIRequest{
		Recipient: msg.Recipient,
		Message:   msg.MessageContent,
		// Sender:    msg.Sender,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Error marshalling message data (ID: %d): %v", msg.ID, err)
		w.updateMessageStatus(msg.ID, models.StatusFailed, fmt.Sprintf("Error preparing request: %v", err))
		return
	}

	// Create HTTP request with dynamic URL that includes sender
	fullURL := fmt.Sprintf("%s/%s/send", w.externalAPI.URL, msg.Sender)
	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating HTTP request (ID: %d): %v", msg.ID, err)
		w.updateMessageStatus(msg.ID, models.StatusFailed, fmt.Sprintf("Error creating request: %v", err))
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", w.externalAPI.Key)

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("Error sending message to external API (ID: %d): %v", msg.ID, err)
		w.updateMessageStatus(msg.ID, models.StatusFailed, fmt.Sprintf("Error sending to external API: %v", err))
		return
	}
	defer resp.Body.Close()

	// Process response
	var respBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		log.Printf("Error decoding API response (ID: %d): %v", msg.ID, err)
		w.updateMessageStatus(msg.ID, models.StatusFailed, "Error decoding API response")
		return
	}

	// Convert response to string for logging
	respJSON, _ := json.Marshal(respBody)
	respStr := string(respJSON)

	// Update message status based on API response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.updateMessageStatus(msg.ID, models.StatusSent, respStr)
		log.Printf("Message sent successfully (ID: %d)", msg.ID)
	} else {
		w.updateMessageStatus(msg.ID, models.StatusFailed, respStr)
		log.Printf("Message sending failed (ID: %d): %s", msg.ID, respStr)
	}
}

// updateMessageStatus updates the status of a message
func (w *MessageWorker) updateMessageStatus(messageID int, status models.MessageStatus, apiResponse string) {
	_, err := w.db.Exec(`
		UPDATE message 
		SET status = ?, 
			dt_send = ?, 
			external_api_response = ? 
		WHERE id = ?
	`, status, time.Now(), apiResponse, messageID)

	if err != nil {
		log.Printf("Error updating message status (ID: %d): %v", messageID, err)
	}
}
