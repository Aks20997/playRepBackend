package user

import (
	"context"
	"net/http"
	"time"

	"FunRepBackend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var UserCollection *mongo.Collection

func GetUserById(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := c.Param("id")
	var user models.User
	err := UserCollection.FindOne(ctx, bson.M{"Id": id}).Decode(&user)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func GetUserPoints(c *gin.Context) {
	userIdValue, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	userId, ok := userIdValue.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"Id": userId}

	var result struct {
		Points float64 `bson:"points"`
	}

	err := UserCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"points":  result.Points,
	})
}

func GetUserChilds(c *gin.Context) {
	userIdValue, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	userId, ok := userIdValue.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user struct {
		Type   string   `bson:"Type"`
		Childs []string `bson:"Childs"`
	}

	err := UserCollection.FindOne(ctx, bson.M{"Id": userId}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.Type != "Dealer" && user.Type != "Master" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only Dealers or Master can have Childs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"childs":  user.Childs,
	})
}

var RealtimeDataCollection *mongo.Collection

func GetVersionInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex("6846a62ab832fbdec733c8e4")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid object ID"})
		return
	}

	filter := bson.M{"_id": objectID}

	var result struct {
		CurrentVersion int32  `bson:"currentVersion"`
		URL            string `bson:"url"`
	}

	err = RealtimeDataCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"currentVersion": result.CurrentVersion,
		"url":            result.URL,
	})
}

