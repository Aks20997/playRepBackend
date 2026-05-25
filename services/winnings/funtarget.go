package winnings

import (
	"context"
	"log"

	"FunRepBackend/repository/redis"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TransferFunTargetWinnings transfers FunTarget winnings to users
func TransferFunTargetWinnings(userCollection *mongo.Collection, ctx context.Context, winningNumber int) error {
	filter := bson.M{"isBetLockedFunTarget": true, "winningsFunTarget": float64(0)}
	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("❌ Error fetching FT locked users: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	var updated int

	for cursor.Next(ctx) {
		var user struct {
			ID            primitive.ObjectID `bson:"_id"`
			UserId        string             `bson:"Id"`
			TempWinnings2 float64            `bson:"TempWinnings2"`
		}

		if err := cursor.Decode(&user); err != nil {
			log.Printf("❌ Failed to decode FT user: %v", err)
			continue
		}
		finalWinningsFT := user.TempWinnings2

		var update bson.M
		if user.TempWinnings2 > 0 {
			// User has winnings - preserve betsFT so client can restore them
			// Keep isBetLockedFunTarget true so user can claim winnings
			update = bson.M{
				"$set": bson.M{
					"winningsFunTarget":      user.TempWinnings2,
					"TempWinnings2":          0.0,
					"winningNumberFunTarget": winningNumber,
					// Keep isBetLockedFunTarget true and betsFT intact for users with winnings
				},
			}
		} else {
			// User has no winnings - clear bets and unlock
			update = bson.M{
				"$set": bson.M{
					"isBetLockedFunTarget": false,
				},
				"$unset": bson.M{
					"betsFT":     "",
					"betStateFT": "",
				},
			}
		}

		// Save in Redis immediately
		redis.SaveFTToRedis(
			user.UserId,
			user.TempWinnings2,
			finalWinningsFT,
			int32(winningNumber),
			user.TempWinnings2 > 0,
			int32(winningNumber),
		)

		// Synchronous Mongo update to ensure data consistency
		_, err = userCollection.UpdateByID(ctx, user.ID, update)
		if err != nil {
			log.Printf("❌ FT mongo update failed for %s: %v", user.ID.Hex(), err)
			continue
		}

		updated++
	}

	log.Printf("✅ Finalized Fun Target winnings for %d users", updated)
	return nil
}

