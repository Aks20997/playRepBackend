package location

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var UserCollection *mongo.Collection

func SaveUserLocation(c *gin.Context) {
	userIdValue, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId, ok := userIdValue.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	location := bson.M{
		"type":        "Point",
		"coordinates": []float64{req.Longitude, req.Latitude},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := UserCollection.UpdateOne(ctx,
		bson.M{"Id": userId},
		bson.M{"$set": bson.M{"location": location}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Location saved"})
}

func GetAllowedLocation(c *gin.Context) {
	userIdValue, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userId := userIdValue.(string)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user struct {
		Type   string `bson:"Type"`
		Parent string `bson:"parent"`
		Id     string `bson:"Id"`
	}
	err := UserCollection.FindOne(ctx, bson.M{"Id": userId}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	targetId := user.Id
	if user.Type == "Customer" {
		if user.Parent == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parent not set for customer"})
			return
		}
		targetId = user.Parent
	}

	var locationData struct {
		AllowedLatitude  float64 `bson:"allowedLatitude"`
		AllowedLongitude float64 `bson:"allowedLongitude"`
		AllowedRadiusKm  float64 `bson:"allowedRadiusKm"`
	}
	err = UserCollection.FindOne(ctx, bson.M{"Id": targetId}).Decode(&locationData)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Location data not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"latitude":  locationData.AllowedLatitude,
		"longitude": locationData.AllowedLongitude,
		"radiusKm":  locationData.AllowedRadiusKm,
	})
}

