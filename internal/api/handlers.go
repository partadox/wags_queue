package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/partadox/wags_queue/internal/auth"
	"github.com/partadox/wags_queue/internal/models"
)

// sendJSONResponse sends a JSON response
func sendJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// sendErrorResponse sends an error response
func sendErrorResponse(w http.ResponseWriter, statusCode int, message string, details string) {
	errorResp := models.ErrorResponse{
		Error:   message,
		Details: details,
	}
	sendJSONResponse(w, statusCode, errorResp)
}

// handleLogin handles user authentication
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var loginReq models.LoginRequest
	
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body", "")
		return
	}
	
	// For the updated direct API key approach, we'll return the username and key directly
	var apiKey string
	err := s.db.QueryRow("SELECT `key` FROM user WHERE username = ?", loginReq.Username).Scan(&apiKey)
	
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(w, http.StatusUnauthorized, "Invalid credentials", "")
		} else {
			sendErrorResponse(w, http.StatusInternalServerError, "Authentication error", "")
		}
		return
	}
	
	// Compare API keys
	if apiKey != loginReq.Key {
		sendErrorResponse(w, http.StatusUnauthorized, "Invalid credentials", "")
		return
	}
	
	loginResp := models.LoginResponse{
		Username: loginReq.Username,
		Key:      apiKey,
		Message:  "Login successful",
	}
	sendJSONResponse(w, http.StatusOK, loginResp)
}

// handleSendMessage handles single message sending
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var msgReq models.SingleMessageRequest
	
	if err := json.NewDecoder(r.Body).Decode(&msgReq); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body", "")
		return
	}
	
	// Get username from context (set by auth middleware)
	username, ok := auth.GetUsername(r.Context())
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}
	
	// Validate request
	if msgReq.Recipient == "" || msgReq.Message == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Missing required fields", "Recipient and message are required")
		return
	}
	
	// Override sender with authenticated username
	msgReq.Sender = username
	
	// Create queue time (now)
	dtQueue := time.Now()
	
	// Insert message into the database
	var messageID int
	err := s.db.QueryRow(`
		INSERT INTO message (
			sender, recipient, status, dt_store, dt_queue, message
		) VALUES (
			?, ?, ?, ?, ?, ?
		) RETURNING id
	`,
		msgReq.Sender,
		msgReq.Recipient,
		models.StatusPending,
		msgReq.DTStore,
		dtQueue,
		msgReq.Message,
	).Scan(&messageID)
	
	// If database doesn't support RETURNING, use this alternative:
	if err != nil {
		res, err := s.db.Exec(`
			INSERT INTO message (
				sender, recipient, status, dt_store, dt_queue, message
			) VALUES (
				?, ?, ?, ?, ?, ?
			)
		`,
			msgReq.Sender,
			msgReq.Recipient,
			models.StatusPending,
			msgReq.DTStore,
			dtQueue,
			msgReq.Message,
		)
		
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error inserting message: %v", err))
			return
		}
		
		lastID, err := res.LastInsertId()
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, "Database error", "Error retrieving message ID")
			return
		}
		
		messageID = int(lastID)
	}
	
	// Prepare response
	msgResp := models.SingleMessageResponse{
		MessageID: messageID,
		Status:    models.StatusPending,
		DTQueue:   dtQueue,
		Info:      "Message queued successfully",
	}
	
	sendJSONResponse(w, http.StatusCreated, msgResp)
}

// handleSendBulkMessage handles bulk message sending
func (s *Server) handleSendBulkMessage(w http.ResponseWriter, r *http.Request) {
	var bulkReq models.BulkMessageRequest
	
	if err := json.NewDecoder(r.Body).Decode(&bulkReq); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid request body", "")
		return
	}
	
	// Get username from context (set by auth middleware)
	username, ok := auth.GetUsername(r.Context())
	if !ok {
		sendErrorResponse(w, http.StatusUnauthorized, "Authentication required", "")
		return
	}
	
	// Validate request
	if len(bulkReq.Recipients) == 0 || bulkReq.Message == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Missing required fields", "Recipients and message are required")
		return
	}
	
	// Override sender with authenticated username
	bulkReq.Sender = username
	
	// Convert bulk data to JSON
	bulkJSON, err := json.Marshal(map[string]interface{}{
		"recipients": bulkReq.Recipients,
		"message":    bulkReq.Message,
	})
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Error processing request", "")
		return
	}
	
	// Insert bulk message into the database
	var bulkID int
	err = s.db.QueryRow(`
		INSERT INTO message_bulk (
			sender, status, dt_store, bulk
		) VALUES (
			?, ?, ?, ?
		) RETURNING id
	`,
		bulkReq.Sender,
		models.BulkStatusProcess,
		bulkReq.DTStore,
		bulkJSON,
	).Scan(&bulkID)
	
	// If database doesn't support RETURNING, use this alternative:
	if err != nil {
		res, err := s.db.Exec(`
			INSERT INTO message_bulk (
				sender, status, dt_store, bulk
			) VALUES (
				?, ?, ?, ?
			)
		`,
			bulkReq.Sender,
			models.BulkStatusProcess,
			bulkReq.DTStore,
			bulkJSON,
		)
		
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error inserting bulk message: %v", err))
			return
		}
		
		lastID, err := res.LastInsertId()
		if err != nil {
			sendErrorResponse(w, http.StatusInternalServerError, "Database error", "Error retrieving bulk message ID")
			return
		}
		
		bulkID = int(lastID)
	}
	
	// Prepare response
	bulkResp := models.BulkMessageResponse{
		BulkMessageID: bulkID,
		Status:        models.BulkStatusProcess,
		Info:          "Bulk message received and is being processed",
	}
	
	sendJSONResponse(w, http.StatusAccepted, bulkResp)
}

// handleGetMessages handles retrieving messages for UI
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	queryValues := r.URL.Query()
	year := queryValues.Get("year")
	month := queryValues.Get("month")
	senderFilter := queryValues.Get("sender_filter")
	
	// Validate year parameter
	if year == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Missing year parameter", "")
		return
	}
	
	// Get username from context (set by auth middleware)
	username, _ := auth.GetUsername(r.Context())
	
	// Build query based on parameters
	query := `
		SELECT 
			id, recipient, status, 
			CASE WHEN type IS NULL OR type = '' THEN 'NO' ELSE 'YES' END AS broadcast_message,
			DATE_FORMAT(dt_store, '%d-%m-%y %H:%i:%s') AS dt_store_fmt,
			DATE_FORMAT(dt_queue, '%d-%m-%y %H:%i:%s') AS dt_queue_fmt,
			CASE WHEN dt_send IS NULL THEN NULL ELSE DATE_FORMAT(dt_send, '%d-%m-%y %H:%i:%s') END AS dt_send_fmt,
			message
		FROM message
		WHERE sender = ? AND YEAR(dt_store) = ?
	`
	
	args := []interface{}{username, year}
	
	// Add month filter if specified
	if month != "" && month != "all" {
		query += " AND MONTH(dt_store) = ?"
		args = append(args, month)
	}
	
	// Add sender filter if specified (for admin users)
	if senderFilter != "" {
		// Replace username with senderFilter in the args slice
		args[0] = senderFilter
	}
	
	// Add order by clause
	query += " ORDER BY dt_store DESC LIMIT 1000"
	
	// Execute query
	rows, err := s.db.Query(query, args...)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error querying messages: %v", err))
		return
	}
	defer rows.Close()
	
	// Build response
	var messages []*models.MessageView
	for rows.Next() {
		var msg models.MessageView
		var dtSendFmt sql.NullString
		
		err := rows.Scan(
			&msg.ID,
			&msg.Recipient,
			&msg.Status,
			&msg.BroadcastMessage,
			&msg.DTStore,
			&msg.DTQueue,
			&dtSendFmt,
			&msg.Message,
		)
		
		if err != nil {
			continue // Skip this row and continue with the next
		}
		
		if dtSendFmt.Valid {
			msg.DTSend = &dtSendFmt.String
		}
		
		messages = append(messages, &msg)
	}
	
	if err := rows.Err(); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error iterating messages: %v", err))
		return
	}
	
	sendJSONResponse(w, http.StatusOK, messages)
}

// handleGetBroadcasts handles retrieving bulk messages for UI
func (s *Server) handleGetBroadcasts(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	queryValues := r.URL.Query()
	year := queryValues.Get("year")
	month := queryValues.Get("month")
	
	// Validate year parameter
	if year == "" {
		sendErrorResponse(w, http.StatusBadRequest, "Missing year parameter", "")
		return
	}
	
	// Get username from context (set by auth middleware)
	username, _ := auth.GetUsername(r.Context())
	
	// Build query based on parameters
	query := `
		SELECT 
			id, sender, status, 
			DATE_FORMAT(dt_store, '%d-%m-%y %H:%i:%s') AS dt_store_fmt,
			CASE WHEN dt_convert IS NULL THEN NULL ELSE DATE_FORMAT(dt_convert, '%d-%m-%y %H:%i:%s') END AS dt_convert_fmt
		FROM message_bulk
		WHERE sender = ? AND YEAR(dt_store) = ?
	`
	
	args := []interface{}{username, year}
	
	// Add month filter if specified
	if month != "" && month != "all" {
		query += " AND MONTH(dt_store) = ?"
		args = append(args, month)
	}
	
	// Add order by clause
	query += " ORDER BY dt_store DESC LIMIT 1000"
	
	// Execute query
	rows, err := s.db.Query(query, args...)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error querying broadcasts: %v", err))
		return
	}
	defer rows.Close()
	
	// Build response
	var broadcasts []*models.MessageBulkView
	for rows.Next() {
		var bulk models.MessageBulkView
		var dtConvertFmt sql.NullString
		
		err := rows.Scan(
			&bulk.ID,
			&bulk.Sender,
			&bulk.Status,
			&bulk.DTStore,
			&dtConvertFmt,
		)
		
		if err != nil {
			continue // Skip this row and continue with the next
		}
		
		if dtConvertFmt.Valid {
			bulk.DTConvert = &dtConvertFmt.String
		}
		
		broadcasts = append(broadcasts, &bulk)
	}
	
	if err := rows.Err(); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error iterating broadcasts: %v", err))
		return
	}
	
	sendJSONResponse(w, http.StatusOK, broadcasts)
}

// handleGetBroadcastDetails handles retrieving details of a bulk message
func (s *Server) handleGetBroadcastDetails(w http.ResponseWriter, r *http.Request) {
	// Get bulk_id from URL parameters
	vars := mux.Vars(r)
	bulkIDStr := vars["bulk_id"]
	
	bulkID, err := strconv.Atoi(bulkIDStr)
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid bulk_id parameter", "")
		return
	}
	
	// Get username from context (set by auth middleware)
	username, _ := auth.GetUsername(r.Context())
	
	// First, check if the bulk message belongs to the user
	var bulkSender string
	err = s.db.QueryRow("SELECT sender FROM message_bulk WHERE id = ?", bulkID).Scan(&bulkSender)
	if err != nil {
		if err == sql.ErrNoRows {
			sendErrorResponse(w, http.StatusNotFound, "Bulk message not found", "")
		} else {
			sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error checking bulk message: %v", err))
		}
		return
	}
	
	// Only allow the owner to see the details (or just check if user is admin in a real system)
	if bulkSender != username {
		sendErrorResponse(w, http.StatusForbidden, "Access denied", "You can only view your own bulk messages")
		return
	}
	
	// Get all messages that belong to this bulk message
	query := `
		SELECT 
			id, recipient, status, 
			'YES' AS broadcast_message,
			DATE_FORMAT(dt_store, '%d-%m-%y %H:%i:%s') AS dt_store_fmt,
			DATE_FORMAT(dt_queue, '%d-%m-%y %H:%i:%s') AS dt_queue_fmt,
			CASE WHEN dt_send IS NULL THEN NULL ELSE DATE_FORMAT(dt_send, '%d-%m-%y %H:%i:%s') END AS dt_send_fmt,
			message
		FROM message
		WHERE type = ?
		ORDER BY id
	`
	
	// Execute query
	rows, err := s.db.Query(query, bulkIDStr)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error querying broadcast details: %v", err))
		return
	}
	defer rows.Close()
	
	// Build response
	var messages []*models.MessageView
	for rows.Next() {
		var msg models.MessageView
		var dtSendFmt sql.NullString
		
		err := rows.Scan(
			&msg.ID,
			&msg.Recipient,
			&msg.Status,
			&msg.BroadcastMessage,
			&msg.DTStore,
			&msg.DTQueue,
			&dtSendFmt,
			&msg.Message,
		)
		
		if err != nil {
			continue // Skip this row and continue with the next
		}
		
		if dtSendFmt.Valid {
			msg.DTSend = &dtSendFmt.String
		}
		
		messages = append(messages, &msg)
	}
	
	if err := rows.Err(); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error iterating messages: %v", err))
		return
	}
	
	sendJSONResponse(w, http.StatusOK, messages)
}

// handleGetAvailableYears retrieves all unique years from messages and bulk messages
func (s *Server) handleGetAvailableYears(w http.ResponseWriter, r *http.Request) {
	// Get username from context (set by auth middleware)
	username, _ := auth.GetUsername(r.Context())
	
	// Query for unique years from both message and message_bulk tables
	query := `
		SELECT DISTINCT years FROM (
			SELECT YEAR(dt_store) as years FROM message WHERE sender = ?
			UNION
			SELECT YEAR(dt_store) as years FROM message_bulk WHERE sender = ?
		) as all_years
		ORDER BY years DESC
	`
	
	// Execute query
	rows, err := s.db.Query(query, username, username)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error querying years: %v", err))
		return
	}
	defer rows.Close()
	
	// Build response
	years := []int{}
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			continue // Skip this row and continue with the next
		}
		years = append(years, year)
	}
	
	// If no years found, include current year
	if len(years) == 0 {
		years = append(years, time.Now().Year())
	}
	
	if err := rows.Err(); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, "Database error", fmt.Sprintf("Error iterating years: %v", err))
		return
	}
	
	sendJSONResponse(w, http.StatusOK, years)
}
