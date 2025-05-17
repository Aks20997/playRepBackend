package routes

import (
	"FunRepBackend/models"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var userCollection *mongo.Collection
var realtimeDataCollection *mongo.Collection

type BetRequest struct {
	Bets map[string]float64 `json:"bets"`
}

func InitRoutes(router *gin.Engine, db *mongo.Database) {
	userCollection = db.Collection("Users")
	realtimeDataCollection = db.Collection("RealtimeData")
	userRoutes := router.Group("/api")
	{
		userRoutes.GET("/user/:id", getUserById)
		userRoutes.GET("/getPoints/:id", getUserPoints)
		userRoutes.POST("/login", loginUser)
		userRoutes.POST("/rouletteBets/:id", updateBets)
		userRoutes.POST("/funTargetBets/:id", updateBetsFT)
		userRoutes.GET("/claimRouletteWinnings/:id", claimRouletteWinnings)
		userRoutes.GET("/claimFtWinnings/:id", claimFTWinnings)
		userRoutes.GET("/getRouletteWinnings/:id", getRouletteWinnings)
		userRoutes.GET("/getFTWinnings/:id", getFunTargetWinnings)
		userRoutes.GET("/getBetState/:id", getBetState)
		userRoutes.GET("/getBetsFT/:id", getBetsFT)
		userRoutes.GET("/getRouletteData", getRouletteData)
		userRoutes.GET("/getFtData", getFtData)
	}
}

func getUserById(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	var user models.User
	err := userCollection.FindOne(ctx, bson.M{"Id": id}).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func loginUser(c *gin.Context) {
	var loginData struct {
		Id       string `json:"Id"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid JSON",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := userCollection.FindOne(ctx, bson.M{"Id": loginData.Id}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "User not found",
		})
		return
	}

	if loginData.Password != user.PasswordHash {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid password",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"userId":  user.UserId,
		"points":  user.Points,
		"message": "Login Successful",
	})
}

func getUserPoints(c *gin.Context) {
	userId := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var result struct {
		Points float64 `bson:"points"`
	}

	err := userCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"points":  result.Points,
	})
}

func updateBets(c *gin.Context) {
	userId := c.Param("id")

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
	if err := userCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points - totalBet

	// Timestamp for betsHistory
	timestamp := time.Now().Format("2006-01-02T15:04:05")

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

	_, err := userCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
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

func updateBetsFT(c *gin.Context) {
	userId := c.Param("id")

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

	// Calculate total amount from ftBets
	var totalBet float64
	for _, amount := range req.NumberBets {
		totalBet += amount
	}

	// Fetch current user points
	var userDoc struct {
		Points float64 `bson:"points"`
	}
	if err := userCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	newPoints := userDoc.Points - totalBet

	// Current timestamp for betsHistoryFT
	timestamp := time.Now().Format("2006-01-02T15:04:05")

	update := bson.M{
		"points":               newPoints,
		"isBetLockedFunTarget": true,
	}
	if req.NumberBets != nil {
		update["betsFT"] = req.NumberBets
		update["betsHistoryFT."+timestamp] = req.NumberBets
	}

	_, err := userCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
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

func claimRouletteWinnings(c *gin.Context) {
	userId := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	// Step 1: Fetch current winningsRoulette and points
	var userDoc struct {
		Points           float64 `bson:"points"`
		WinningsRoulette float64 `bson:"winningsRoulette"`
	}
	if err := userCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Step 2: Calculate new points and update
	newPoints := userDoc.Points + userDoc.WinningsRoulette

	update := bson.M{
		"$set": bson.M{
			"points":           newPoints,
			"winningsRoulette": 0,
			"isBetLocked":      false,
		}, "$unset": bson.M{
			"betState": "",
		},
	}

	_, err := userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Winnings claimed successfully",
	})
}

func claimFTWinnings(c *gin.Context) {
	userId := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	// Step 1: Fetch current winningsRoulette and points
	var userDoc struct {
		Points     float64 `bson:"points"`
		WinningsFT float64 `bson:"winningsFunTarget"`
	}
	if err := userCollection.FindOne(ctx, filter).Decode(&userDoc); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Step 2: Calculate new points and update
	newPoints := userDoc.Points + userDoc.WinningsFT

	update := bson.M{
		"$set": bson.M{
			"points":               newPoints,
			"winningsFunTarget":    0,
			"isBetLockedFunTarget": false,
		}, "$unset": bson.M{
			"betsFT": "",
		},
	}

	_, err := userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Winnings claimed successfully",
	})
}

func getRouletteWinnings(c *gin.Context) {
	userId := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		WinningsRoulette      float64 `bson:"winningsRoulette"`
		WinningNumberRoulette int32   `bson:"winningNumberRoulette"`
		isBetLocked           bool    `bson:"isBetLocked"`
	}

	if err := userCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"winningsRoulette":      result.WinningsRoulette,
		"winningNumberRoulette": result.WinningNumberRoulette,
		"isBetLocked":           result.isBetLocked,
	})
}

func getFunTargetWinnings(c *gin.Context) {
	userId := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		WinningsFunTarget      float64 `bson:"winningsFunTarget"`
		WinningNumberFunTarget int32   `bson:"winningNumberFunTarget"`
		IsBetLockedFunTarget   bool    `bson:"isBetLockedFunTarget"`
	}

	if err := userCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"winningsFunTarget":      result.WinningsFunTarget,
		"winningNumberFunTarget": result.WinningNumberFunTarget,
		"isBetLockedFunTarget":   result.IsBetLockedFunTarget,
	})
}

func getBetState(c *gin.Context) {
	userId := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		BetState map[string]float64 `bson:"betState"`
	}

	if err := userCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"betState": result.BetState})
}

func getBetsFT(c *gin.Context) {
	userId := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}
	var result struct {
		BetsFT map[string]float64 `bson:"betsFT"`
	}

	if err := userCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"betsFT": result.BetsFT})
}

func getRouletteData(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		NextWinningNumber int   `bson:"nextWinningNumber"`
		WinningNumber     int   `bson:"winningNumber"`
		RouletteHistory   []any `bson:"rouletteHistory"`
	}

	if err := realtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nextWinningNumber": result.NextWinningNumber,
		"winningNumber":     result.WinningNumber,
		"rouletteHistory":   result.RouletteHistory,
	})
}

func getFtData(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		return
	}
	filter := bson.M{"_id": objectID}

	var result struct {
		NextWinningNumberFT int   `bson:"nextWinningNumberFT"`
		WinningNumberFT     int   `bson:"winningNumberFT"`
		NextMultiplier      int   `bson:"nextMultiplier"`
		Multiplier          int   `bson:"multiplier"`
		FunTargetHistory    []any `bson:"funTargetHistory"`
	}

	if err := realtimeDataCollection.FindOne(ctx, filter).Decode(&result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"nextWinningNumber": result.NextWinningNumberFT,
		"winningNumber":     result.WinningNumberFT,
		"rouletteHistory":   result.FunTargetHistory,
		"multiplier":        result.Multiplier,
		"nextMultiplier":    result.NextMultiplier,
	})
}

// func updateBets(c *gin.Context) {
// 	userId := c.Param("id")

// 	var req struct {
// 		NumberBets map[string]float64 `json:"numberBets"`
// 		BetState   map[string]float64 `json:"betState"`
// 	}

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	filter := bson.M{"Id": userId}

// 	// Current timestamp for betsHistory
// 	timestamp := time.Now().Format("2006-01-02T15:04:05")

// 	update := bson.M{}
// 	if req.NumberBets != nil {
// 		update["bets"] = req.NumberBets
// 	}
// 	if req.BetState != nil {
// 		update["betState"] = req.BetState
// 		update["betsHistory."+timestamp] = req.BetState
// 	}

// 	_, err := userCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bets"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Bets updated successfully"})
// }

// func updateBetsFT(c *gin.Context) {
// 	userId := c.Param("id")

// 	var req struct {
// 		NumberBets map[string]float64 `json:"ftBets"`
// 	}

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
// 		return
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	filter := bson.M{"Id": userId}

// 	// Current timestamp for betsHistory
// 	timestamp := time.Now().Format("2006-01-02T15:04:05")

// 	update := bson.M{}
// 	if req.NumberBets != nil {
// 		update["betsFT"] = req.NumberBets
// 		update["betsHistoryFT."+timestamp] = req.NumberBets
// 	}

// 	_, err := userCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bets"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Bets updated successfully"})
// }
