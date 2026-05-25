package redis

import (
	"context"
	"time"

	"FunRepBackend/config"
)

// SaveFTToRedis saves FunTarget winnings data to Redis
func SaveFTToRedis(
	userId string,
	temp float64,
	winnings float64,
	winningNumber int32,
	isLocked bool,
	nextWinningNumber int32,
) {
	key := "ft:" + userId
	ctx := context.Background()

	config.RedisClient.HSet(ctx, key, map[string]interface{}{
		"TempWinnings2":          temp,
		"winningsFunTarget":      winnings,
		"winningNumberFunTarget": winningNumber,
		"isBetLockedFunTarget":   boolToInt(isLocked),
		"nextWinningNumberFT":    nextWinningNumber,
	})

	config.RedisClient.Expire(ctx, key, 60*time.Second)
}

