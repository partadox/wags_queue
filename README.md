# WAGS Queue System

WAGS (WhatsApp Gateway Service) Queue System is a reliable message queueing system designed for delivering messages to recipients through an external API. The system supports both individual and bulk message sending, with a user-friendly web interface for monitoring message status.

## Features

- **Authentication System**: Direct API key authentication
- **Single Message Sending**: Send individual messages to recipients
- **Bulk Message Sending**: Send the same message to multiple recipients at once
- **Message Queuing**: Messages are stored and queued for reliable delivery
- **Worker System**: Background workers process message delivery
- **Dashboard**: Monitor message statistics
- **Message History**: View and filter message history
- **Broadcast History**: Track bulk message broadcasts

## Tech Stack

- **Backend**: Go with standard library and minimal dependencies
- **Database**: MySQL
- **Frontend**: HTML, CSS, JavaScript with Bootstrap UI
- **Authentication**: Direct API key authentication
- **Container**: Docker support for easy deployment

## Setup Instructions

### Prerequisites

- Go 1.21+
- MySQL 8.0+
- Docker (optional, for containerized deployment)

### Database Setup

1. Create a MySQL database:

```sql
CREATE DATABASE db_wags CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. Run the schema.sql file to create the necessary tables:

```bash
mysql -u yourusername -p db_wags < schema.sql
```

### Configuration

1. Copy the .env.example file to .env and update the values:

```bash
cp .env.example .env
```

2. Update the environment variables in the .env file:

```
# Server configuration
SERVER_PORT=8080
SERVER_READ_TIMEOUT=15
SERVER_WRITE_TIMEOUT=15
SERVER_IDLE_TIMEOUT=60

# Database configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=yourusername
DB_PASSWORD=yourpassword
DB_NAME=db_wags

# JWT configuration
# Note: These settings are maintained for backward compatibility but not used in the current implementation
JWT_SECRET=your-secure-jwt-secret-key
JWT_EXPIRES=24

# External API configuration
EXTERNAL_API_URL=https://wag.artakusuma.com/api/clients
EXTERNAL_API_KEY=your-api-key
```

### Running the Application

#### Method 1: Direct Go Build

1. Build and run the application:

```bash
go build -o wags_queue.exe ./cmd/server
./wags_queue.exe
```

2. Access the application at http://localhost:8080

#### Method 2: Using Docker

1. Build the Docker image:

```bash
docker build -t wags_queue .
```

2. Run the container:

```bash
docker run -d -p 8080:8080 --env-file .env --name wags_queue wags_queue
```

3. Access the application at http://localhost:8080

## Architecture

The system is designed with the following components:

1. **API Server**: Handles HTTP requests, authentication, and database operations
2. **Message Worker**: Processes messages from the queue and sends them to the external API
3. **Bulk Processor**: Converts bulk messages into individual messages
4. **Database**: Stores users, messages, and bulk messages

## Authentication

The system uses API key-based authentication:

1. **Login**: Users provide their username and API key to the `/api/auth/login` endpoint.
2. **Authentication**: The server verifies the credentials and returns the username and API key.
3. **Subsequent Requests**: For all subsequent API requests, clients must include the API key in the `X-Api-Key` header.
4. **Authorization**: The server middleware checks the API key for each request, retrieves the associated username, and authorizes access to the requested resources.

This direct API key authentication approach is simple and efficient for this application's use case.

## API Endpoints

### Authentication

- `POST /api/auth/login`: Authenticate user and receive API key

### Message Operations

- `POST /api/messages/send`: Send a single message
- `POST /api/messages/send-bulk`: Send a bulk message

### UI Data

- `GET /api/ui/messages`: Get list of messages
- `GET /api/ui/broadcasts`: Get list of bulk messages
- `GET /api/ui/broadcasts/{bulk_id}/details`: Get details of a bulk message

## User Interface

The web interface is accessible at the root URL and includes:

- Login screen
- Dashboard with message statistics
- Message sending form
- Bulk message sending form
- Message history with filtering
- Broadcast history with filtering

## Default User

The system is pre-configured with a default user:

- Username: `telkomsel`
- API Key: (Set in the schema.sql file)

## License

This project is proprietary and confidential.

## Author

Arta Kusuma H.

## Integration with Other Systems

### PHP Integration Example

Here's how to integrate with the WAGS Queue System from a PHP application:

#### Authentication

```php
<?php
// Authentication and getting API key
function getApiKey($username, $apiKey) {
    $url = 'http://localhost:8080/api/auth/login';
    
    $data = array(
        'username' => $username,
        'key' => $apiKey
    );
    
    $options = array(
        'http' => array(
            'header'  => "Content-type: application/json\r\n",
            'method'  => 'POST',
            'content' => json_encode($data)
        )
    );
    
    $context  = stream_context_create($options);
    $result = file_get_contents($url, false, $context);
    
    if ($result === FALSE) { 
        throw new Exception('Authentication failed'); 
    }
    
    $response = json_decode($result, true);
    return $response;
}

// Example usage
try {
    $authResponse = getApiKey('telkomsel', 'your-api-key');
    $username = $authResponse['username'];
    $apiKey = $authResponse['key'];
    echo "Authentication successful. API Key: " . $apiKey . "\n";
} catch (Exception $e) {
    echo 'Error: ' . $e->getMessage() . "\n";
}
?>
```

#### Sending a Single Message

```php
<?php
function sendMessage($apiKey, $recipient, $message) {
    $url = 'http://localhost:8080/api/messages/send';
    
    $data = array(
        'recipient' => $recipient,
        'message' => $message,
        'dt_store' => date('Y-m-d H:i:s')
    );
    
    $options = array(
        'http' => array(
            'header'  => "Content-type: application/json\r\n" .
                         "X-Api-Key: " . $apiKey . "\r\n",
            'method'  => 'POST',
            'content' => json_encode($data)
        )
    );
    
    $context = stream_context_create($options);
    $result = file_get_contents($url, false, $context);
    
    if ($result === FALSE) { 
        throw new Exception('Failed to send message'); 
    }
    
    return json_decode($result, true);
}

// Example usage
try {
    $messageResponse = sendMessage($apiKey, '628123456789', 'Hello from PHP integration!');
    echo "Message sent successfully. ID: " . $messageResponse['message_id'] . "\n";
} catch (Exception $e) {
    echo 'Error: ' . $e->getMessage() . "\n";
}
?>
```

#### Sending a Bulk Message

```php
<?php
function sendBulkMessage($apiKey, $recipients, $message) {
    $url = 'http://localhost:8080/api/messages/send-bulk';
    
    $data = array(
        'recipients' => $recipients,
        'message' => $message,
        'dt_store' => date('Y-m-d H:i:s')
    );
    
    $options = array(
        'http' => array(
            'header'  => "Content-type: application/json\r\n" .
                         "X-Api-Key: " . $apiKey . "\r\n",
            'method'  => 'POST',
            'content' => json_encode($data)
        )
    );
    
    $context = stream_context_create($options);
    $result = file_get_contents($url, false, $context);
    
    if ($result === FALSE) { 
        throw new Exception('Failed to send bulk message'); 
    }
    
    return json_decode($result, true);
}

// Example usage
try {
    $recipients = array('628123456789', '628987654321');
    $bulkResponse = sendBulkMessage($apiKey, $recipients, 'Bulk message from PHP integration!');
    echo "Bulk message sent successfully. ID: " . $bulkResponse['bulk_message_id'] . "\n";
} catch (Exception $e) {
    echo 'Error: ' . $e->getMessage() . "\n";
}
?>
```

### Using with cURL (Alternative)

If you prefer using cURL for API requests, here's an alternative approach:

```php
<?php
function sendMessageWithCurl($apiKey, $recipient, $message) {
    $url = 'http://localhost:8080/api/messages/send';
    
    $data = array(
        'recipient' => $recipient,
        'message' => $message,
        'dt_store' => date('Y-m-d H:i:s')
    );
    
    $ch = curl_init($url);
    
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_POST, true);
    curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
    curl_setopt($ch, CURLOPT_HTTPHEADER, array(
        'Content-Type: application/json',
        'X-Api-Key: ' . $apiKey
    ));
    
    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    
    if (curl_errno($ch) || $httpCode >= 400) {
        throw new Exception('API request failed: ' . curl_error($ch) . ' HTTP code: ' . $httpCode);
    }
    
    curl_close($ch);
    return json_decode($response, true);
}
?>
```