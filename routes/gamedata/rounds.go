package gamedata

import (
	"net/http"
	"time"

	"FunRepBackend/services/rounds"

	"github.com/gin-gonic/gin"
)

// GetCurrentRounds returns the current round states for all games
func GetCurrentRounds(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	allRounds := rounds.GetAllCurrentRounds()
	
	// Format response with game names as keys
	response := make(map[string]interface{})
	for game, round := range allRounds {
		response[game] = gin.H{
			"game":       round.Game,
			"roundId":    round.RoundID.String(),
			"startTs":    round.StartTS,
			"endTs":      round.EndTS,
			"durationMs": round.DurationMs,
			"timeLeft":   round.GetTimeLeft(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"rounds":  response,
	})
}

// GetCurrentRound returns the current round state for a specific game
func GetCurrentRound(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	game := c.Param("game")
	if game == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Game parameter required"})
		return
	}

	round, exists := rounds.GetCurrentRound(game)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Round not found for game: " + game})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"round": gin.H{
			"game":       round.Game,
			"roundId":    round.RoundID.String(),
			"startTs":    round.StartTS,
			"endTs":      round.EndTS,
			"durationMs": round.DurationMs,
			"timeLeft":   round.GetTimeLeft(),
		},
	})
}

// GetServerTime returns the server Unix time in seconds (single source of truth)
// Clients should calculate per-game remaining time using: 
// remaining = roundDuration - 1 - ((serverUnixTime + gameOffset) % roundDuration)
// Game offsets: roulette=0, funtarget=15, triplefun=30, andarbahar=45
// The remainingSeconds field is generic and should not be used for game-specific timers
func GetServerTime(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	serverUnixTime := time.Now().Unix()
	remainingSeconds := 59 - (serverUnixTime % 60)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"serverUnixTime":  serverUnixTime,
		"remainingSeconds": remainingSeconds,
	})
}

