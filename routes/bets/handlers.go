package bets

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var UserCollection *mongo.Collection

func UpdateBets(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		NumberBets map[string]float64 `json:"numberBets"`
		BetState   map[string]float64 `json:"betState"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	// Calculate total bet amount
	var totalBet float64
	for _, amount := range req.BetState {
		totalBet += amount
	}

	// Fetch current points
	var userDoc struct {
		Points float64 `bson:"points"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points - totalBet

	// Timestamp for betsHistory
	ist := time.FixedZone("IST", 5.5*3600)
	timestamp := time.Now().In(ist).Format("2006-01-02T15:04:05")

	update := bson.M{
		"points":      newPoints,
		"isBetLocked": true,
	}
	if req.NumberBets != nil {
		update["bets"] = req.NumberBets
	}
	if req.BetState != nil {
		update["betState"] = req.BetState
		update["betsHistory."+timestamp] = req.BetState
	}

	_, err := UserCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Bets updated, points deducted, and bet locked",
		"newPoints": newPoints,
	})
}

func UpdateBetsFT(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		NumberBets map[string]float64 `json:"ftBets"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var totalBet float64
	for _, amount := range req.NumberBets {
		totalBet += amount
	}

	var userDoc struct {
		Points float64 `bson:"points"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points - totalBet

	ist := time.FixedZone("IST", 5.5*3600)
	timestamp := time.Now().In(ist).Format("2006-01-02T15:04:05")

	update := bson.M{
		"points":               newPoints,
		"isBetLockedFunTarget": true,
	}
	if req.NumberBets != nil {
		update["betsFT"] = req.NumberBets
		update["betsHistoryFT."+timestamp] = req.NumberBets
	}

	_, err := UserCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "FT Bets updated, points deducted, and bet locked",
		"newPoints": newPoints,
	})
}

func UpdateBetsTripleFun(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		NumberBets map[string]float64 `json:"tripleFunBets"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var totalBet float64
	for _, amount := range req.NumberBets {
		totalBet += amount
	}

	var userDoc struct {
		Points float64 `bson:"points"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points - totalBet

	ist := time.FixedZone("IST", 5.5*3600)
	timestamp := time.Now().In(ist).Format("2006-01-02T15:04:05")

	update := bson.M{
		"points":               newPoints,
		"isBetLockedTripleFun": true,
	}
	if req.NumberBets != nil {
		update["betsTripleFun"] = req.NumberBets
		update["betsHistoryTripleFun."+timestamp] = req.NumberBets
	}

	_, err := UserCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Triple Fun Bets updated, points deducted, and bet locked",
		"newPoints": newPoints,
	})
}

func UpdateBetsAndarBahar(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Bets map[string]float64 `json:"betsAB"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var totalBet float64
	for _, amount := range req.Bets {
		totalBet += amount
	}

	var userDoc struct {
		Points float64 `bson:"points"`
	}
	if err := UserCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points - totalBet

	ist := time.FixedZone("IST", 19800)
	timestamp := time.Now().In(ist).Format("2006-01-02T15:04:05")

	update := bson.M{
		"points":        newPoints,
		"isBetLockedAB": true,
	}
	if req.Bets != nil {
		update["betsAB"] = req.Bets
		update["betsHistoryAB."+timestamp] = req.Bets
	}

	_, err := UserCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "AndarBahar bets updated, points deducted, and bet locked",
		"newPoints": newPoints,
	})
}

