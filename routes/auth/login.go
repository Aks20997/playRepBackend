package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"FunRepBackend/models"
	"FunRepBackend/session"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var UserCollection *mongo.Collection

func StartSessionCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("✅ Session cleanup service started - will check every minute for stale online users")

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			
			cursor, err := UserCollection.Find(ctx, bson.M{"isOnline": true})
			if err != nil {
				cancel()
				continue
			}

			cleanedCount := 0
			for cursor.Next(ctx) {
				var u models.User
				if err := cursor.Decode(&u); err != nil {
					continue
				}

				// Check if session exists in Redis
				exists, _ := RedisClient.Exists(ctx, u.UserId).Result()
				if exists == 0 {
					// Session expired - set user offline and clean up in-memory token
					_, _ = UserCollection.UpdateOne(ctx, bson.M{"Id": u.UserId}, bson.M{"$set": bson.M{"isOnline": false}})
					session.RemoveToken(u.UserId)
					cleanedCount++
				}
			}
			cursor.Close(ctx)
			cancel()

			if cleanedCount > 0 {
				log.Printf("Cleaned up %d stale online user(s)", cleanedCount)
			}
		}
	}
}

func LoginUser(c *gin.Context) {
	var loginData struct {
		Id       string `json:"Id"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid JSON"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := UserCollection.FindOne(ctx, bson.M{"Id": loginData.Id}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "User not found"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "ID is Blocked"})
		return
	}

	if loginData.Password != user.PasswordHash {
		key := "login_attempts:" + loginData.Id
		attempts, _ := RedisClient.Get(ctx, key).Int()
		attempts++

		if attempts >= 3 {
			// Block user in DB
			UserCollection.UpdateOne(ctx, bson.M{"Id": loginData.Id}, bson.M{"$set": bson.M{"isActive": false}})

			// Set Redis key to auto-unblock after 5 minutes
			RedisClient.Set(ctx, "auto_unblock:"+loginData.Id, 1, 5*time.Minute)

			// Clear login attempts
			RedisClient.Del(ctx, key)

			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "ID is Blocked"})
			return
		}

		// Set failed attempts with 2-minute TTL
		RedisClient.Set(ctx, key, attempts, 2*time.Minute)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid password"})
		return
	}

	// Reset failed attempts on successful login
	RedisClient.Del(ctx, "login_attempts:"+loginData.Id)

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.UserId,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Token generation failed"})
		return
	}

	// Invalidate existing session and clean up old token
	RedisClient.Del(ctx, user.UserId)
	session.RemoveToken(user.UserId)

	// Store new session with 5-minute TTL
	err = RedisClient.Set(ctx, user.UserId, tokenString, 5*time.Minute).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to store session"})
		return
	}

	session.RegisterValidToken(user.UserId, tokenString)

	// Mark user as online
	_, _ = UserCollection.UpdateOne(ctx, bson.M{"Id": loginData.Id}, bson.M{"$set": bson.M{"isOnline": true}})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"userId":  user.UserId,
		"points":  user.Points,
		"token":   tokenString,
		"type":    user.Type,
		"message": "Login Successful",
	})
}

func LogoutUser(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not authenticated",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userIDStr := fmt.Sprintf("%v", userID)
	
	// Delete session from Redis
	err := RedisClient.Del(ctx, userIDStr).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to logout",
		})
		return
	}

	// Clean up in-memory token
	session.RemoveToken(userIDStr)

	// Set user offline
	_, _ = UserCollection.UpdateOne(ctx, bson.M{"Id": userIDStr}, bson.M{"$set": bson.M{"isOnline": false}})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successfully logged out",
	})
}

