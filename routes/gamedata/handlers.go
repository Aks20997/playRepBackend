package gamedata

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	RealtimeDataCollection *mongo.Collection
	RoundsDataCollection    *mongo.Collection
)

func GetRouletteData(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		NextWinningNumber int `bson:"nextWinningNumber"`
		WinningNumber     int `bson:"winningNumber"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nextWinningNumber": result.NextWinningNumber,
		"winningNumber":     result.WinningNumber,
	})
}

func GetFtData(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		NextWinningNumberFT int `bson:"nextWinningNumberFT"`
		WinningNumberFT     int `bson:"winningNumberFT"`
		NextMultiplier      int `bson:"nextMultiplier"`
		Multiplier          int `bson:"multiplier"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nextWinningNumberFT": result.NextWinningNumberFT,
		"winningNumberFT":     result.WinningNumberFT,
		"multiplier":          result.Multiplier,
		"nextMultiplier":      result.NextMultiplier,
	})
}

func GetTfData(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		NextWinningNumberTF string `bson:"nextWinningNumberTF"`
		WinningNumberTF     string `bson:"winningNumberTF"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nextWinningNumberTF": result.NextWinningNumberTF,
		"winningNumberTF":     result.WinningNumberTF,
	})
}

func GetAbData(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		NextWinningNumberAB int   `bson:"nextWinningNumberAB"`
		WinningNumberAB     int   `bson:"winningNumberAB"`
		NextABArray         []int `bson:"nextABArray"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nextWinningNumberAB": result.NextWinningNumberAB,
		"winningNumberAB":     result.WinningNumberAB,
		"nextABArray":         result.NextABArray,
	})
}

func GetRouletteHistory(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		RouletteHistory []any `bson:"rouletteHistory"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rouletteHistory": result.RouletteHistory,
	})
}

func GetFtHistory(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		FunTargetHistory []any `bson:"funTargetHistory"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"funTargetHistory": result.FunTargetHistory,
	})
}

func GetTripleFunHistory(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		TripleFunHistory []any `bson:"tripleFunHistory"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tripleFunHistory": result.TripleFunHistory,
	})
}

func GetAndarBaharHistory(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		AndarBaharHistory []any `bson:"ABHistory"`
	}

	if err := RealtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ABHistory": result.AndarBaharHistory,
	})
}

func GetDrawDetails(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("69314e8483520b7bf43cc484")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		RoundsData1 map[string]any `bson:"roundsData1"`
		RoundsData2 map[string]any `bson:"roundsData2"`
		RoundsData3 map[string]any `bson:"roundsData3"`
		RoundsData4 map[string]any `bson:"roundsData4"`
	}

	if err := RoundsDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roundsData1": result.RoundsData1,
		"roundsData2": result.RoundsData2,
		"roundsData3": result.RoundsData3,
		"roundsData4": result.RoundsData4,
	})
}

