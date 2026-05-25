package winnings

import (
	"context"
	"log"
	"time"

	"FunRepBackend/repository/redis"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TransferRouletteWinnings transfers roulette winnings to users
func TransferRouletteWinnings(userCollection *mongo.Collection, ctx context.Context, winningNumber int) error {
	filter := bson.M{"isBetLocked": true, "winningsRoulette": float64(0)}
	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("❌ Error fetching locked users: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	var updatedCount int
	for cursor.Next(ctx) {
		var user struct {
			ID            primitive.ObjectID `bson:"_id"`
			UserId        string             `bson:"Id"`
			TempWinnings1 float64            `bson:"TempWinnings1"`
		}

		if err := cursor.Decode(&user); err != nil {
			log.Printf("❌ Failed to decode user: %v", err)
			continue
		}
		finalWinnings := user.TempWinnings1

		// Determine the Mongo update
		var update bson.M
		if user.TempWinnings1 > 0 {
			// User has winnings - preserve bets so client can restore them
			// Keep isBetLocked true so user can claim winnings
			update = bson.M{
				"$set": bson.M{
					"winningsRoulette":      user.TempWinnings1,
					"TempWinnings1":         0.0,
					"winningNumberRoulette": winningNumber,
					// Keep isBetLocked true and bets intact for users with winnings
				},
			}
		} else {
			// User has no winnings - clear bets and unlock
			update = bson.M{
				"$set": bson.M{
					"isBetLocked": false,
				},
				"$unset": bson.M{
					"bets":     "",
					"betState": "",
				},
			}
		}

		// Save to Redis immediately
		redis.SaveRouletteToRedis(
			user.UserId,
			user.TempWinnings1,
			finalWinnings,
			int32(winningNumber),
			user.TempWinnings1 > 0,
			int32(winningNumber),
		)

		// Persist to Mongo async
		uid := user.ID
		upd := update
		go func(id primitive.ObjectID, updateDoc bson.M) {
			mongoCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, mongoErr := userCollection.UpdateByID(mongoCtx, id, updateDoc)
			if mongoErr != nil {
				log.Printf("❌ Mongo async update failed for user %s: %v", id.Hex(), mongoErr)
			}
		}(uid, upd)

		updatedCount++
	}

	log.Printf("✅ Finalized winnings for %d users (redis written, mongo saving async)", updatedCount)
	return nil
}

