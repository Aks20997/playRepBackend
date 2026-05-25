package redis

import (
	"context"
	"time"

	"FunRepBackend/config"
)

// SaveRouletteToRedis saves roulette winnings data to Redis
func SaveRouletteToRedis(
	userId string,
	temp float64,
	winnings float64,
	winningNumber int32,
	isLocked bool,
	nextWinningNumber int32,
) {
	key := "roulette:" + userId
	ctx := context.Background()

	config.RedisClient.HSet(ctx, key, map[string]interface{}{
		"TempWinnings1":         temp,
		"winningsRoulette":      winnings,
		"winningNumberRoulette": winningNumber,
		"isBetLocked":           boolToInt(isLocked),
		"nextWinningNumber":     nextWinningNumber,
	})

	config.RedisClient.Expire(ctx, key, 60*time.Second)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

