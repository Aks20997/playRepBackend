package rounds

import (
	"sync"

	"FunRepBackend/models"
)

var (
	currentRounds = make(map[string]models.RoundState)
	roundsMutex   sync.RWMutex
)

// SetCurrentRound stores the current round state for a game
func SetCurrentRound(game string, round models.RoundState) {
	roundsMutex.Lock()
	defer roundsMutex.Unlock()
	currentRounds[game] = round
}

// GetCurrentRound retrieves the current round state for a game
func GetCurrentRound(game string) (models.RoundState, bool) {
	roundsMutex.RLock()
	defer roundsMutex.RUnlock()
	round, exists := currentRounds[game]
	return round, exists
}

// GetAllCurrentRounds retrieves all current round states
func GetAllCurrentRounds() map[string]models.RoundState {
	roundsMutex.RLock()
	defer roundsMutex.RUnlock()
	
	result := make(map[string]models.RoundState)
	for k, v := range currentRounds {
		result[k] = v
	}
	return result
}

