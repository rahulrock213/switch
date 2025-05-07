package miyagi

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	// IMPORTANT: Replace this with the actual path to your Miyagi socket
	defaultSocatAddress = "/var/run/miyagi.sock"
	defaultDialTimeout  = 2 * time.Second
	defaultIOTimeout    = 5 * time.Second
)

// SendRequest sends a request to the Miyagi agent via the specified socket path and returns the parsed response.
func SendRequest(socketPath string, request MiyagiRequest) (*MiyagiResponse, error) {
	conn, err := net.DialTimeout("unix", socketPath, defaultDialTimeout)
	if err != nil {
		log.Printf("MIYAGI_CLIENT: Failed to connect to Miyagi agent at %s: %v", socketPath, err)
		return nil, fmt.Errorf("failed to connect to Miyagi agent: %w", err)
	}
	defer conn.Close()

	// Set deadlines for read/write
	deadline := time.Now().Add(defaultIOTimeout)
	if err := conn.SetWriteDeadline(deadline); err != nil {
		log.Printf("MIYAGI_CLIENT: Failed to set write deadline: %v", err)
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		log.Printf("MIYAGI_CLIENT: Failed to send request: %v", err)
		return nil, fmt.Errorf("failed to send request to Miyagi agent: %w", err)
	}

	if err := conn.SetReadDeadline(deadline); err != nil {
		log.Printf("MIYAGI_CLIENT: Failed to set read deadline: %v", err)
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	var miyagiResp MiyagiResponse
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&miyagiResp); err != nil {
		log.Printf("MIYAGI_CLIENT: Failed to decode response: %v", err)
		// Consider reading raw response here for debugging if Decode fails often
		return nil, fmt.Errorf("failed to decode response from Miyagi agent: %w", err)
	}

	// The miyagiResp.Error will be checked by the caller.
	return &miyagiResp, nil
}
