package worker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/partadox/wags_queue/internal/models"
)

// BulkProcessor handles the processing of bulk messages
type BulkProcessor struct {
	db   *sql.DB
	done chan struct{}
	wg   sync.WaitGroup
}

// NewBulkProcessor creates a new bulk message processor
func NewBulkProcessor(db *sql.DB) *BulkProcessor {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())
	
	return &BulkProcessor{
		db:   db,
		done: make(chan struct{}),
	}
}

// Run starts the bulk message processor
func (p *BulkProcessor) Run() {
	p.wg.Add(1)
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.processBulkMessages()
		case <-p.done:
			log.Println("Bulk processor is shutting down...")
			return
		}
	}
}

// Stop signals the processor to stop
func (p *BulkProcessor) Stop() {
	close(p.done)
	p.wg.Wait()
	log.Println("Bulk processor stopped")
}

// processBulkMessages processes pending bulk messages
func (p *BulkProcessor) processBulkMessages() {
	// Begin transaction
	tx, err := p.db.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			log.Printf("Panic in processBulkMessages: %v", err)
		}
	}()

	// Get a batch of bulk messages to process
	rows, err := tx.Query(`
		SELECT id, sender, bulk, dt_store 
		FROM message_bulk 
		WHERE status = ? 
		LIMIT 5
	`, models.BulkStatusProcess)

	if err != nil {
		tx.Rollback()
		log.Printf("Error querying bulk messages: %v", err)
		return
	}
	defer rows.Close()

	bulksToProcess := make([]models.MessageBulk, 0)
	for rows.Next() {
		var bulk models.MessageBulk
		if err := rows.Scan(&bulk.ID, &bulk.Sender, &bulk.Bulk, &bulk.DTStore); err != nil {
			log.Printf("Error scanning bulk message row: %v", err)
			continue
		}
		bulksToProcess = append(bulksToProcess, bulk)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		log.Printf("Error iterating bulk message rows: %v", err)
		return
	}

	// If no bulk messages to process, commit empty transaction and return
	if len(bulksToProcess) == 0 {
		if err := tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
		}
		return
	}

	// Commit transaction to release locks before processing
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return
	}

	// Process each bulk message
	for _, bulk := range bulksToProcess {
		go p.processBulkMessage(bulk) // Process in goroutine for non-blocking operation
	}
}

// processBulkMessage processes a single bulk message
func (p *BulkProcessor) processBulkMessage(bulk models.MessageBulk) {
	// Parse the bulk message data
	var bulkData struct {
		Recipients []string `json:"recipients"`
		Message    string   `json:"message"`
	}

	if err := json.Unmarshal(bulk.Bulk, &bulkData); err != nil {
		log.Printf("Error unmarshalling bulk data (ID: %d): %v", bulk.ID, err)
		p.updateBulkStatus(bulk.ID, models.BulkStatusFailed)
		return
	}

	// Insert individual messages for each recipient
	// Calculate time window based on bulk size and max rate to make it look natural
	totalRecipients := len(bulkData.Recipients)
	maxRatePerMinute := 100 // Maximum allowed rate
	
	// Determine minimum total time needed for all messages (in seconds)
	// If recipients count is under max rate, we'll still spread over at least 30 seconds
	minTotalTimeSeconds := 30
	if totalRecipients > maxRatePerMinute {
		minTotalTimeSeconds = (totalRecipients * 60) / maxRatePerMinute
	}
	
	// Base delay for each message
	baseDelay := time.Duration(minTotalTimeSeconds) * time.Second / time.Duration(totalRecipients)
	
	// Store time for the first message
	baseTime := bulk.DTStore
	
	for i, recipient := range bulkData.Recipients {
		// Add random variance to queue time (±50% of base delay)
		randomFactor := 0.5 + rand.Float64()
		messageDelay := time.Duration(float64(baseDelay) * randomFactor)
		
		// Calculate queue time by adding progressive delay to base time
		queueTime := baseTime.Add(time.Duration(i) * baseDelay + messageDelay)
		
		// For first few messages, apply smaller delays to appear natural
		if i < 3 {
			// First message: 1-3 seconds delay
			// Second message: 2-5 seconds delay
			// Third message: 3-8 seconds delay
			randomSeconds := rand.Intn(3) + i + 1
			queueTime = baseTime.Add(time.Duration(randomSeconds) * time.Second)
		}
		
		_, err := p.db.Exec(`
			INSERT INTO message (
				sender, recipient, status, type, dt_store, dt_queue, message
			) VALUES (
				?, ?, ?, ?, ?, ?, ?
			)
		`,
			bulk.Sender,
			recipient,
			models.StatusPending,
			fmt.Sprintf("%d", bulk.ID), // Store bulk ID as type
			bulk.DTStore,               // Use the same dt_store as the bulk message
			queueTime,                  // Set calculated queue time with natural delay
			bulkData.Message,
		)

		if err != nil {
			log.Printf("Error inserting individual message for recipient %s (Bulk ID: %d): %v", 
				recipient, bulk.ID, err)
			// Continue with other recipients
		}
	}

	// Update bulk message status to DONE
	p.updateBulkStatus(bulk.ID, models.BulkStatusDone)
	log.Printf("Bulk message processed successfully (ID: %d), created %d individual messages", 
		bulk.ID, len(bulkData.Recipients))
}

// updateBulkStatus updates the status of a bulk message
func (p *BulkProcessor) updateBulkStatus(bulkID int, status models.BulkMessageStatus) {
	_, err := p.db.Exec(`
		UPDATE message_bulk 
		SET status = ?, 
			dt_convert = ? 
		WHERE id = ?
	`, status, time.Now(), bulkID)

	if err != nil {
		log.Printf("Error updating bulk message status (ID: %d): %v", bulkID, err)
	}
}
