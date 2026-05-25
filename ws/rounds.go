package ws

import (
	"time"
)

// GetGameRemainingTimesFunc is a function type that returns per-game remaining times
type GetGameRemainingTimesFunc func() map[string]int64

var getGameRemainingTimes GetGameRemainingTimesFunc

// SetGetGameRemainingTimesFunc sets the function to get per-game remaining times
func SetGetGameRemainingTimesFunc(fn GetGameRemainingTimesFunc) {
	getGameRemainingTimes = fn
}

// ServerTimePayload represents a server time broadcast (stateless timer)
type ServerTimePayload struct {
	Type            string           `json:"type"`
	ServerUnixTime  int64            `json:"serverUnixTime"`  // Unix time in seconds
	RemainingSeconds int64           `json:"remainingSeconds"` // Generic remaining time (deprecated, use gameRemainingTimes)
	GameRemainingTimes map[string]int64 `json:"gameRemainingTimes"` // Per-game remaining times with offsets
}

// BroadcastServerTime broadcasts server time to all connected clients with per-game remaining times
// gameRemainingTimes is a map of game name -> remaining seconds (calculated with offsets)
func BroadcastServerTime(gameRemainingTimes map[string]int64) {
	serverUnixTime := time.Now().Unix()
	// Keep generic remainingSeconds for backward compatibility, but clients should use gameRemainingTimes
	remainingSeconds := 59 - (serverUnixTime % 60)
	
	payload := ServerTimePayload{
		Type:                "server_time",
		ServerUnixTime:      serverUnixTime,
		RemainingSeconds:    remainingSeconds,
		GameRemainingTimes:  gameRemainingTimes,
	}

	mutex.Lock()
	clients := make([]*Client, 0, len(userClients))
	for _, c := range userClients {
		clients = append(clients, c)
	}
	mutex.Unlock()

	for _, c := range clients {
		c.Mu.Lock()
		if err := c.Conn.WriteJSON(payload); err != nil {
			// Log error but continue with other clients
		}
		c.Mu.Unlock()
	}
}

// SendServerTime sends server time to a specific client with per-game remaining times
// If gameRemainingTimes is nil, it will try to get them from the registered function
func SendServerTime(client *Client, gameRemainingTimes map[string]int64) {
	serverUnixTime := time.Now().Unix()
	remainingSeconds := 59 - (serverUnixTime % 60)
	
	// If gameRemainingTimes is not provided, try to get it from registered function
	if gameRemainingTimes == nil && getGameRemainingTimes != nil {
		gameRemainingTimes = getGameRemainingTimes()
	}
	// If still nil, use empty map
	if gameRemainingTimes == nil {
		gameRemainingTimes = make(map[string]int64)
	}
	
	payload := ServerTimePayload{
		Type:                "server_time",
		ServerUnixTime:      serverUnixTime,
		RemainingSeconds:    remainingSeconds,
		GameRemainingTimes:  gameRemainingTimes,
	}

	client.Mu.Lock()
	defer client.Mu.Unlock()
	if err := client.Conn.WriteJSON(payload); err != nil {
		// Log error
	}
}
