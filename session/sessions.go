package session

import "sync"

var (
    mutex       sync.Mutex
    validTokens = make(map[string]string)
)

func RegisterValidToken(userID string, token string) {
    mutex.Lock()
    validTokens[userID] = token
    mutex.Unlock()
}

func IsValidToken(userID string, token string) bool {
    mutex.Lock()
    defer mutex.Unlock()
    storedToken, ok := validTokens[userID]
    return ok && storedToken == token
}

// RemoveToken removes a token from the in-memory map
func RemoveToken(userID string) {
    mutex.Lock()
    delete(validTokens, userID)
    mutex.Unlock()
}