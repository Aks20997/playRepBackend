package rounds

import (
	"context"
	"log"
	"sync"
	"time"

	"FunRepBackend/config"
	"FunRepBackend/ws"

	"go.mongodb.org/mongo-driver/mongo"
)

type RoundManager struct {
	db *mongo.Database
	mu sync.Mutex
	// Game logic callbacks - stored per game to trigger at specific times
	gameCallbacks map[string]*GameCallbacks
	// Track last triggered round index and event type to avoid duplicate triggers
	lastTriggeredRound map[string]int64 // round index when onUpdate was triggered
	lastTransferRound  map[string]int64 // round index when onEnd was triggered
}

type GameCallbacks struct {
	Column       string
	RoundDuration int // Round duration in seconds (e.g., 60, 120, 45)
	Offset       int  // Time offset in seconds to stagger games (e.g., 0, 15, 30, 45)
	OnUpdate     func()
	OnEnd        func()
}

func NewRoundManager(db *mongo.Database) *RoundManager {
	rm := &RoundManager{
		db:                  db,
		gameCallbacks:       make(map[string]*GameCallbacks),
		lastTriggeredRound:  make(map[string]int64),
		lastTransferRound:   make(map[string]int64),
	}
	
	// Register function to get game remaining times in ws package
	ws.SetGetGameRemainingTimesFunc(rm.getAllGameRemainingTimes)
	
	// Start a single goroutine that checks Unix time modulo 60 for game logic triggers
	// This is minimal and necessary for game functionality (onUpdate at 7s, onEnd at 0s)
	go rm.gameLogicTicker()
	
	// Start periodic server time broadcasts (every 10 seconds for client resync)
	go rm.serverTimeBroadcaster()
	
	return rm
}

// getAllGameRemainingTimes calculates remaining time for all registered games with their offsets
func (rm *RoundManager) getAllGameRemainingTimes() map[string]int64 {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	gameRemainingTimes := make(map[string]int64)
	for game, callbacks := range rm.gameCallbacks {
		roundDuration := callbacks.RoundDuration
		if roundDuration == 0 {
			roundDuration = 60
		}
		now := time.Now().Unix()
		offsetTime := now + int64(callbacks.Offset)
		remaining := int64(roundDuration) - 1 - (offsetTime % int64(roundDuration))
		gameRemainingTimes[game] = remaining
	}
	return gameRemainingTimes
}

// serverTimeBroadcaster broadcasts server time every 10 seconds for client resync
// Includes per-game remaining times calculated with offsets
func (rm *RoundManager) serverTimeBroadcaster() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		// Calculate remaining time for each game with their offsets
		gameRemainingTimes := rm.getAllGameRemainingTimes()
		ws.BroadcastServerTime(gameRemainingTimes)
	}
}

// gameLogicTicker runs every second and checks if we need to trigger game logic
// Computes remaining time directly from Unix time and triggers on remaining == 7 and remaining == 0
func (rm *RoundManager) gameLogicTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for now := range ticker.C {
		unixTime := now.Unix()
		
		rm.mu.Lock()
		for game, callbacks := range rm.gameCallbacks {
			roundDuration := callbacks.RoundDuration
			if roundDuration == 0 {
				roundDuration = 60 // Default to 60 seconds if not set
			}
			
			offset := callbacks.Offset
			// Apply offset to unixTime: offsetTime = unixTime + offset
			// This shifts the timer calculation by the offset amount
			offsetTime := unixTime + int64(offset)
			
			// Compute remaining time with offset: remaining = roundDuration - 1 - (offsetTime % roundDuration)
			// This gives us the countdown from maxCounter (e.g., 59) down to 0, offset by the game's offset
			remaining := roundDuration - 1 - int(offsetTime%int64(roundDuration))
			
			// Track current round index to avoid duplicate triggers (using offset time)
			currentRoundIndex := offsetTime / int64(roundDuration)
			lastTriggeredRound := rm.lastTriggeredRound[game]
			lastTransferRound := rm.lastTransferRound[game]
			
			// Debug logging when remaining is near 0
			if remaining <= 2 {
				log.Printf("🔍 [%s] Near end: remaining=%d, roundIndex=%d, lastTriggeredRound=%d, lastTransferRound=%d", 
					game, remaining, currentRoundIndex, lastTriggeredRound, lastTransferRound)
			}
			
			// Trigger onUpdate when remaining == 7 (frontend shows 7 seconds)
			// This calculates TempWinnings for all users
			// Only trigger once per round
			if remaining == 7 && currentRoundIndex != lastTriggeredRound {
				log.Printf("⏰ [%s] Remaining at 7 seconds - Calculating TempWinnings... (remaining=%d, roundIndex=%d, lastTriggeredRound=%d)", 
					game, remaining, currentRoundIndex, lastTriggeredRound)
				// Update lastTriggeredRound to prevent duplicate triggers
				rm.lastTriggeredRound[game] = currentRoundIndex
				if callbacks.OnUpdate != nil {
					go func(cb func(), g string) {
						defer func() {
							if r := recover(); r != nil {
								log.Printf("⚠️ Recovered from panic in onUpdate for %s: %v", g, r)
							}
						}()
						cb()
						log.Printf("✅ [%s] TempWinnings calculation completed", g)
					}(callbacks.OnUpdate, game)
				} else {
					log.Printf("❌ [%s] OnUpdate callback is nil!", game)
				}
			}
			
			// Trigger onEnd when remaining == 0 (frontend shows 0 seconds - round end)
			// This transfers TempWinnings to final winnings
			// Only trigger once per round (use separate tracking from onUpdate)
			if remaining == 0 {
				log.Printf("🔍 [%s] remaining==0 detected: roundIndex=%d, lastTransferRound=%d, condition=%v", 
					game, currentRoundIndex, lastTransferRound, currentRoundIndex != lastTransferRound)
				if currentRoundIndex != lastTransferRound {
					log.Printf("⏰ [%s] Remaining at 0 seconds - Transferring winnings... (remaining=%d, roundIndex=%d, lastTransferRound=%d)", 
						game, remaining, currentRoundIndex, lastTransferRound)
					// Update lastTransferRound to prevent duplicate triggers
					rm.lastTransferRound[game] = currentRoundIndex
					
					if callbacks.OnEnd != nil {
						go func(cb func(), g string) {
							defer func() {
								if r := recover(); r != nil {
									log.Printf("⚠️ Recovered from panic in onEnd for %s: %v", g, r)
								}
							}()
							cb()
							log.Printf("✅ [%s] Winnings transfer completed", g)
						}(callbacks.OnEnd, game)
					} else {
						log.Printf("❌ [%s] OnEnd callback is nil!", game)
					}
				} else {
					log.Printf("⚠️ [%s] Skipping transfer: already triggered for roundIndex=%d", game, currentRoundIndex)
				}
			}
		}
		rm.mu.Unlock()
	}
}

// RegisterGameCallbacks registers callbacks for a game that will be triggered based on round duration
// offset is the time offset in seconds to stagger this game's timer from others (e.g., 0, 15, 30, 45)
func (rm *RoundManager) RegisterGameCallbacks(game string, column string, roundDuration int, offset int, onUpdate func(), onEnd func()) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if roundDuration == 0 {
		roundDuration = 60 // Default to 60 seconds if not specified
	}
	
	rm.gameCallbacks[game] = &GameCallbacks{
		Column:        column,
		RoundDuration: roundDuration,
		Offset:        offset,
		OnUpdate:      onUpdate,
		OnEnd:         onEnd,
	}
	
	// Initialize last triggered rounds for this game
	// Set to a value that will definitely allow the first trigger
	now := time.Now().Unix()
	offsetTime := now + int64(offset)
	currentRoundIndex := offsetTime / int64(roundDuration)
	// Initialize to -1 to ensure first triggers work regardless of when server starts
	rm.lastTriggeredRound[game] = -1
	rm.lastTransferRound[game] = -1
	log.Printf("🔧 [%s] Initialized: roundDuration=%d, offset=%d, currentRoundIndex=%d, lastTriggeredRound=%d, lastTransferRound=%d", 
		game, roundDuration, offset, currentRoundIndex, rm.lastTriggeredRound[game], rm.lastTransferRound[game])
}

// GetRemainingSeconds calculates remaining seconds in the current 60-second round
// Formula: remaining = 59 - (currentUnixTime % 60)
func GetRemainingSeconds() int64 {
	now := time.Now().Unix()
	return 59 - (now % 60)
}

// GetRemainingSecondsForGame calculates remaining seconds for a specific game with its offset
// Returns -1 if game is not registered
func (rm *RoundManager) GetRemainingSecondsForGame(game string) int64 {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	callbacks, exists := rm.gameCallbacks[game]
	if !exists {
		return -1
	}
	
	roundDuration := callbacks.RoundDuration
	if roundDuration == 0 {
		roundDuration = 60
	}
	
	now := time.Now().Unix()
	offsetTime := now + int64(callbacks.Offset)
	remaining := int64(roundDuration) - 1 - (offsetTime % int64(roundDuration))
	
	return remaining
}

// GetServerUnixTime returns the current server Unix time in seconds
func GetServerUnixTime() int64 {
	return time.Now().Unix()
}

// FinalizeRoundHistory updates the round history with the final winning value
// If nextRoundEndTs is provided (> 0), it also creates a new "NOT OPEN" entry for the next round
func (rm *RoundManager) FinalizeRoundHistory(column string, endTs int64, value interface{}, nextRoundEndTs int64) error {
	ctx := context.Background()
	return config.FinalizeRoundHistory(ctx, rm.db, column, endTs, value, nextRoundEndTs)
}
