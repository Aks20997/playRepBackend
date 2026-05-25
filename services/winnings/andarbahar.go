package winnings

import (
	"context"
	"log"

	"FunRepBackend/repository/redis"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TransferAndarBaharWinnings transfers AndarBahar winnings to users
func TransferAndarBaharWinnings(userCollection *mongo.Collection, ctx context.Context, winningNumber int) error {
	filter := bson.M{"isBetLockedAB": true, "winningsAB": float64(0)}
	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("❌ Error fetching AB locked users: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	var updated int

	for cursor.Next(ctx) {
		var user struct {
			ID                  primitive.ObjectID `bson:"_id"`
			UserId              string             `bson:"Id"`
			TempWinnings4       float64            `bson:"TempWinnings4"`
			TempTotalWinningsAB float64            `bson:"TempTotalWinningsAB"`
		}

		if err := cursor.Decode(&user); err != nil {
			log.Printf("❌ Failed to decode AB user: %v", err)
			continue
		}

		finalWinningsAB := user.TempWinnings4
		finalTotalAB := user.TempTotalWinningsAB

		var update bson.M
		if user.TempWinnings4 > 0 {
			// User has winnings - preserve betsAB so client can restore them
			// Keep isBetLockedAB true so user can claim winnings
			update = bson.M{
				"$set": bson.M{
					"winningsAB":      user.TempWinnings4,
					"TempWinnings4":   0.0,
					"winningNumberAB": winningNumber,
					"totalWinningsAB": user.TempTotalWinningsAB,
					// Keep isBetLockedAB true and betsAB intact for users with winnings
				},
			}
		} else {
			// User has no winnings - clear bets and unlock
			update = bson.M{
				"$set": bson.M{
					"isBetLockedAB": false,
				},
				"$unset": bson.M{
					"betsAB":     "",
					"betStateAB": "",
				},
			}
		}

		redis.SaveABToRedis(
			user.UserId,
			finalWinningsAB,
			finalTotalAB,
			user.TempWinnings4 > 0,
			int32(winningNumber),
		)

		uid := user.ID
		upd := update
		go func(id primitive.ObjectID, doc bson.M) {
			_, e := userCollection.UpdateByID(context.Background(), id, doc)
			if e != nil {
				log.Printf("❌ Async AB mongo update failed for %s: %v", id.Hex(), e)
			}
		}(uid, upd)

		updated++
	}

	log.Printf("✅ Finalized AndarBahar winnings for %d users", updated)
	return nil
}

