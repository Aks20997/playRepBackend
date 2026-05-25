package redis

import (
	"context"
	"time"

	"FunRepBackend/config"
)

// SaveABToRedis saves AndarBahar winnings data to Redis
func SaveABToRedis(userId string, winningsAB float64, totalAB float64, isLocked bool, winningNumber int32) {
	key := "ab:" + userId
	ctx := context.Background()
	
	config.RedisClient.HSet(ctx, key, map[string]interface{}{
		"winningsAB":      winningsAB,
		"totalWinningsAB": totalAB,
		"isBetLockedAB":   boolToInt(isLocked),
		"winningNumberAB": winningNumber,
	})
	
	config.RedisClient.Expire(ctx, key, 30*time.Second)
}

