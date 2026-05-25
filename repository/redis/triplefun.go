package redis

import (
	"context"
	"time"

	"FunRepBackend/config"
)

// SaveTFToRedis saves TripleFun winnings data to Redis
func SaveTFToRedis(
	userId string,
	temp float64,
	winnings float64,
	isLocked bool,
) {
	key := "tf:" + userId
	ctx := context.Background()

	config.RedisClient.HSet(ctx, key, map[string]interface{}{
		"TempWinnings3":        temp,
		"winningsTripleFun":    winnings,
		"isBetLockedTripleFun": boolToInt(isLocked),
	})

	config.RedisClient.Expire(ctx, key, 60*time.Second)
}

