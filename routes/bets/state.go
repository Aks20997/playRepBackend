package bets

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func GetBetState(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		BetState map[string]float64 `bson:"betState"`
	}

	if err := UserCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"betState": result.BetState})
}

func GetBetsFT(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		BetsFT              map[string]float64 `bson:"betsFT"`
		IsBetLockedFunTarget bool              `bson:"isBetLockedFunTarget"`
	}

	if err := UserCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"betsFT":              result.BetsFT,
		"isBetLockedFunTarget": result.IsBetLockedFunTarget,
	})
}

func GetBetStateTF(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		BetsTripleFun        map[string]float64 `bson:"betsTripleFun"`
		IsBetLockedTripleFun bool               `bson:"isBetLockedTripleFun"`
	}

	if err := UserCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"betsTripleFun":        result.BetsTripleFun,
		"isBetLockedTripleFun": result.IsBetLockedTripleFun,
	})
}

func GetBetStateAB(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		BetsAB        map[string]float64 `bson:"betsAB"`
		IsBetLockedAB bool               `bson:"isBetLockedAB"`
	}

	if err := UserCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"betsAB":        result.BetsAB,
		"isBetLockedAB": result.IsBetLockedAB,
	})
}

