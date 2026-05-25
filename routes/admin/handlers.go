package admin

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var UserCollection *mongo.Collection

func AdminDBHandler(c *gin.Context) {
	if UserCollection == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database not initialized"})
		return
	}

	var req struct {
		Action     string `json:"action"`
		Collection string `json:"collection"`
		Filter     bson.M `json:"filter"`
		Update     bson.M `json:"update"`
		Data       bson.M `json:"data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Filter == nil {
		req.Filter = bson.M{}
	}

	if idStr, ok := req.Filter["_id"].(string); ok {
		if objID, err := primitive.ObjectIDFromHex(idStr); err == nil {
			req.Filter["_id"] = objID
		}
	}

	defaultExcludes := bson.M{
		"passwordHash":           0,
		"TempWinnings1":          0,
		"TempWinnings2":          0,
		"winningsRoulette":       0,
		"winningsFunTarget":      0,
		"isBetLocked":            0,
		"isBetLockedFunTarget":   0,
		"winningNumberFunTarget": 0,
		"winningNumberRoulette":  0,
		"isActive":               0,
		"betsHistory":            0,
		"parent":                 0,
		"pin":                    0,
		"betsHistoryFT":          0,
	}

	coll := UserCollection.Database().Collection(req.Collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch req.Action {
	case "get":
		opts := options.FindOne().SetProjection(defaultExcludes)
		var result bson.M
		err := coll.FindOne(ctx, req.Filter, opts).Decode(&result)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found", "details": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"data": result})
		}

	case "update":
		result, err := coll.UpdateOne(ctx, req.Filter, req.Update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed", "details": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"matched": result.MatchedCount, "modified": result.ModifiedCount})
		}

	case "insert":
		_, err := coll.InsertOne(ctx, req.Data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert failed", "details": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"success": true})
		}

	case "delete":
		result, err := coll.DeleteOne(ctx, req.Filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed", "details": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"deletedCount": result.DeletedCount})
		}

	case "find_all":
		opts := options.Find().SetProjection(defaultExcludes)
		cursor, err := coll.Find(ctx, req.Filter, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Find failed", "details": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var results []bson.M
		if err := cursor.All(ctx, &results); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cursor decoding failed", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": results})

	case "find_latest":
		opts := options.Find().
			SetProjection(defaultExcludes).
			SetSort(bson.D{{Key: "_id", Value: -1}}).
			SetLimit(1)
		cursor, err := coll.Find(ctx, req.Filter, opts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Find latest failed", "details": err.Error()})
			return
		}
		defer cursor.Close(ctx)

		var results []bson.M
		if err := cursor.All(ctx, &results); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cursor decoding failed", "details": err.Error()})
			return
		}
		if len(results) > 0 {
			c.JSON(http.StatusOK, gin.H{"data": results[0]})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "No documents found"})
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action"})
	}
}

func ResetPasswordByID(c *gin.Context) {
	userIDFromJWT, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var requestingUser struct {
		Type string `bson:"Type"`
	}
	err := UserCollection.FindOne(context.Background(), bson.M{"Id": userIDFromJWT}).Decode(&requestingUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user from JWT"})
		return
	}

	if requestingUser.Type != "Master" && requestingUser.Type != "Dealer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
		return
	}

	var req struct {
		UserID string `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	newPassword := string(b)

	filter := bson.M{"Id": req.UserID}
	update := bson.M{"$set": bson.M{"PasswordHash": newPassword}}

	res, err := UserCollection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"newPassword": newPassword})
}

func ResetPin(c *gin.Context) {
	userIDFromJWT, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx := context.Background()
	var requestingUser struct {
		Type string `bson:"Type"`
	}
	err := UserCollection.FindOne(ctx, bson.M{"Id": userIDFromJWT}).Decode(&requestingUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user from JWT"})
		return
	}

	if requestingUser.Type != "Master" && requestingUser.Type != "Dealer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only Master or dealer users can reset PIN"})
		return
	}

	var req struct {
		UserID string `json:"userId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	newPin := rand.Intn(9000) + 1000

	res, err := UserCollection.UpdateOne(ctx,
		bson.M{"Id": req.UserID},
		bson.M{"$set": bson.M{"pin": newPin}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pin"})
		return
	}
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"newPin": newPin})
}

func ChangePassword(c *gin.Context) {
	userIDFromJWT, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr, ok := userIDFromJWT.(string)
	if !ok {
		idStr = fmt.Sprintf("%v", userIDFromJWT)
	}

	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
		Pin         int    `json:"pin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var user struct {
		PasswordHash string `bson:"PasswordHash"`
		Pin          int    `bson:"pin"`
	}
	err := UserCollection.FindOne(context.TODO(), bson.M{"Id": idStr}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.PasswordHash != req.OldPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Old password is incorrect"})
		return
	}

	if user.Pin != req.Pin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid pin"})
		return
	}

	update := bson.M{"$set": bson.M{"PasswordHash": req.NewPassword}}
	_, err = UserCollection.UpdateOne(context.TODO(), bson.M{"Id": idStr}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

func ChangePin(c *gin.Context) {
	userIDFromJWT, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr, ok := userIDFromJWT.(string)
	if !ok {
		idStr = fmt.Sprintf("%v", userIDFromJWT)
	}

	var req struct {
		CurrentPin int    `json:"currentPin"`
		NewPin     int    `json:"newPin"`
		Password   string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var user struct {
		PasswordHash string `bson:"PasswordHash"`
		Pin          int    `bson:"pin"`
	}
	err := UserCollection.FindOne(context.TODO(), bson.M{"Id": idStr}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.PasswordHash != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password is incorrect"})
		return
	}

	if user.Pin != req.CurrentPin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current pin is incorrect"})
		return
	}

	update := bson.M{"$set": bson.M{"Pin": req.NewPin}}
	_, err = UserCollection.UpdateOne(context.TODO(), bson.M{"Id": idStr}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pin changed successfully"})
}

