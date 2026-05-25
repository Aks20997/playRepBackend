package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"FunRepBackend/session"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	JWTSecret   string
	RedisClient *redis.Client
)

// JWTAuthMiddleware validates JWT tokens and sets user context
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{
				"success": false,
				"message": "Authorization header missing",
				"code":    "AUTH_MISSING",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.AbortWithStatusJSON(401, gin.H{
				"success": false,
				"message": "Invalid token format",
				"code":    "AUTH_INVALID_FORMAT",
			})
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(401, gin.H{
				"success": false,
				"message": "Invalid token",
				"code":    "AUTH_INVALID_TOKEN",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{
				"success": false,
				"message": "Invalid claims",
				"code":    "AUTH_INVALID_CLAIMS",
			})
			return
		}

		userID := fmt.Sprintf("%v", claims["user_id"])

		// Create context with timeout for Redis operations
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Verify session
		val, err := RedisClient.Get(ctx, userID).Result()
		if err == redis.Nil || val != tokenString {
			// Session expired or invalid - clean up in-memory token and set user offline
			session.RemoveToken(userID)
			
			// Set user offline in database if session expired (not just invalid token)
			if err == redis.Nil {
				go func(uid string) {
					updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer updateCancel()
					_, _ = UserCollection.UpdateOne(updateCtx, bson.M{"Id": uid}, bson.M{"$set": bson.M{"isOnline": false}})
				}(userID)
			}
			
			c.AbortWithStatusJSON(401, gin.H{
				"success": false,
				"message": "Session expired or invalid",
				"code":    "SESSION_EXPIRED",
			})
			return
		}

		// Refresh TTL only on POST requests
		if c.Request.Method == "POST" {
			RedisClient.Expire(ctx, userID, 5*time.Minute)
		}

		c.Set("userId", userID)
		c.Next()
	}
}

