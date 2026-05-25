package winnings

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	UserCollection *mongo.Collection
	RedisClient    *redis.Client
)

func ClaimRouletteWinnings(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var userDoc struct {
		Points           float64 `bson:"points"`
		WinningsRoulette float64 `bson:"winningsRoulette"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points + userDoc.WinningsRoulette

	update := bson.M{
		"$set": bson.M{
			"points":           newPoints,
			"winningsRoulette": 0,
			"isBetLocked":      false,
		},
		"$unset": bson.M{
			"betState": "",
		},
	}

	_, err := UserCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user document"})
		return
	}

	redisKey := "roulette:" + userId.(string)
	RedisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Winnings claimed successfully",
	})
}

func ClaimFTWinnings(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var userDoc struct {
		Points     float64 `bson:"points"`
		WinningsFT float64 `bson:"winningsFunTarget"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points + userDoc.WinningsFT

	update := bson.M{
		"$set": bson.M{
			"points":               newPoints,
			"winningsFunTarget":    0,
			"isBetLockedFunTarget": false,
		},
		"$unset": bson.M{
			"betsFT": "",
		},
	}

	_, err := UserCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user document"})
		return
	}

	redisKey := "ft:" + userId.(string)
	RedisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Winnings claimed successfully",
	})
}

func ClaimTripleFunWinnings(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var userDoc struct {
		Points            float64 `bson:"points"`
		WinningsTripleFun float64 `bson:"winningsTripleFun"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points + userDoc.WinningsTripleFun

	update := bson.M{
		"$set": bson.M{
			"points":               newPoints,
			"winningsTripleFun":    0,
			"isBetLockedTripleFun": false,
		},
		"$unset": bson.M{
			"betsTripleFun": "",
		},
	}

	_, err := UserCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user document"})
		return
	}

	redisKey := "tf:" + userId.(string)
	RedisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "TripleFun winnings claimed successfully",
	})
}

func ClaimAndarBaharWinnings(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var userDoc struct {
		Points     float64 `bson:"points"`
		WinningsAB float64 `bson:"totalWinningsAB"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points + userDoc.WinningsAB

	update := bson.M{
		"$set": bson.M{
			"points":              newPoints,
			"isBetLockedAB":       false,
			"totalWinningsAB":     0,
			"winningsAB":          0,
			"TempWinningsAB":      float64(0),
			"TempTotalWinningsAB": float64(0),
		},
		"$unset": bson.M{
			"betsAB": "",
		},
	}

	if _, err := UserCollection.UpdateOne(ctx, filter, update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user document"})
		return
	}

	redisKey := "ab:" + userId.(string)
	RedisClient.Del(ctx, redisKey)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "AndarBahar winnings claimed successfully",
	})
}

func GetRouletteWinnings(c *gin.Context) {
	userId := c.GetString("userId")
	ctx := context.Background()
	key := "roulette:" + userId

	data, err := RedisClient.HGetAll(ctx, key).Result()

	if err == nil && data["winningsRoulette"] != "" {
		log.Println("♻️ Redis → Roulette winnings")

		wVal, _ := strconv.ParseFloat(data["winningsRoulette"], 64)
		nVal, _ := strconv.Atoi(data["winningNumberRoulette"])
		locked := data["isBetLocked"] == "1"

		c.JSON(http.StatusOK, gin.H{
			"winningsRoulette":      wVal,
			"winningNumberRoulette": nVal,
			"isBetLocked":           locked,
		})
		return
	}

	log.Println("🐢 Mongo → Roulette winnings")

	var result struct {
		WinningsRoulette      float64 `bson:"winningsRoulette"`
		WinningNumberRoulette int32   `bson:"winningNumberRoulette"`
		IsBetLocked           bool    `bson:"isBetLocked"`
	}

	ctxMongo, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := UserCollection.FindOne(ctxMongo, bson.M{"Id": userId}).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"winningsRoulette":      result.WinningsRoulette,
		"winningNumberRoulette": result.WinningNumberRoulette,
		"isBetLocked":           result.IsBetLocked,
	})
}

func GetFunTargetWinnings(c *gin.Context) {
	userId := c.GetString("userId")
	ctx := context.Background()
	key := "ft:" + userId

	data, err := RedisClient.HGetAll(ctx, key).Result()

	if err == nil && data["winningsFunTarget"] != "" {
		log.Println("♻️ Redis → FunTarget winnings")

		wVal, _ := strconv.ParseFloat(data["winningsFunTarget"], 64)
		nVal, _ := strconv.Atoi(data["winningNumberFunTarget"])
		locked := data["isBetLockedFunTarget"] == "1"

		c.JSON(http.StatusOK, gin.H{
			"winningsFunTarget":      wVal,
			"winningNumberFunTarget": nVal,
			"isBetLockedFunTarget":   locked,
		})
		return
	}

	log.Println("🐢 Mongo → FunTarget winnings")

	var result struct {
		WinningsFunTarget      float64 `bson:"winningsFunTarget"`
		WinningNumberFunTarget int32   `bson:"winningNumberFunTarget"`
		IsBetLockedFunTarget   bool    `bson:"isBetLockedFunTarget"`
	}

	ctxMongo, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := UserCollection.FindOne(ctxMongo, bson.M{"Id": userId}).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"winningsFunTarget":      result.WinningsFunTarget,
		"winningNumberFunTarget": result.WinningNumberFunTarget,
		"isBetLockedFunTarget":   result.IsBetLockedFunTarget,
	})
}

func GetTripleFunWinnings(c *gin.Context) {
	userId := c.GetString("userId")
	ctx := context.Background()
	key := "tf:" + userId

	data, err := RedisClient.HGetAll(ctx, key).Result()

	if err == nil && data["winningsTripleFun"] != "" {
		log.Println("♻️ Redis → TripleFun winnings")

		wVal, _ := strconv.ParseFloat(data["winningsTripleFun"], 64)
		locked := data["isBetLockedTripleFun"] == "1"

		c.JSON(http.StatusOK, gin.H{
			"winningsTripleFun":    wVal,
			"isBetLockedTripleFun": locked,
		})
		return
	}

	log.Println("🐢 Mongo → TripleFun winnings")

	var result struct {
		WinningsTripleFun    float64 `bson:"winningsTripleFun"`
		IsBetLockedTripleFun bool    `bson:"isBetLockedTripleFun"`
	}

	ctxMongo, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := UserCollection.FindOne(ctxMongo, bson.M{"Id": userId}).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"winningsTripleFun":    result.WinningsTripleFun,
		"isBetLockedTripleFun": result.IsBetLockedTripleFun,
	})
}

func GetAndarBaharWinnings(c *gin.Context) {
	userId := c.GetString("userId")
	ctx := context.Background()
	key := "ab:" + userId

	data, err := RedisClient.HGetAll(ctx, key).Result()

	if err == nil && data["winningsAB"] != "" {
		log.Println("♻️ Redis → AndarBahar winnings")

		wVal, _ := strconv.ParseFloat(data["winningsAB"], 64)
		tVal, _ := strconv.ParseFloat(data["totalWinningsAB"], 64)
		locked := data["isBetLockedAB"] == "1"

		c.JSON(http.StatusOK, gin.H{
			"winningsAB":      wVal,
			"totalWinningsAB": tVal,
			"isBetLockedAB":   locked,
		})
		return
	}

	log.Println("Mongo → AndarBahar winnings")

	var result struct {
		WinningsAB      float64 `bson:"winningsAB"`
		TotalWinningsAB float64 `bson:"totalWinningsAB"`
		IsBetLockedAB   bool    `bson:"isBetLockedAB"`
	}

	ctxMongo, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := UserCollection.FindOne(ctxMongo, bson.M{"Id": userId}).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"winningsAB":      result.WinningsAB,
		"totalWinningsAB": result.TotalWinningsAB,
		"isBetLockedAB":   result.IsBetLockedAB,
	})
}
