package winnings

import (
	"context"
	"log"

	"FunRepBackend/repository/redis"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TransferTripleFunWinnings transfers TripleFun winnings to users
func TransferTripleFunWinnings(userCollection *mongo.Collection, ctx context.Context) error {
	filter := bson.M{"isBetLockedTripleFun": true, "winningsTripleFun": float64(0)}
	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("❌ Error fetching TF locked users: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	var updated int

	for cursor.Next(ctx) {
		var user struct {
			ID            primitive.ObjectID `bson:"_id"`
			UserId        string             `bson:"Id"`
			TempWinnings3 float64            `bson:"TempWinnings3"`
		}

		if err := cursor.Decode(&user); err != nil {
			log.Printf("❌ Failed to decode TF user: %v", err)
			continue
		}
		finalWinningsTF := user.TempWinnings3

		var update bson.M
		if user.TempWinnings3 > 0 {
			// User has winnings - preserve betsTripleFun so client can restore them
			// Keep isBetLockedTripleFun true so user can claim winnings
			update = bson.M{
				"$set": bson.M{
					"winningsTripleFun":    user.TempWinnings3,
					"TempWinnings3":        0.0,
					"winningNumberTF":       "", // Will be set by finalize
					// Keep isBetLockedTripleFun true and betsTripleFun intact for users with winnings
				},
			}
		} else {
			// User has no winnings - clear bets and unlock
			update = bson.M{
				"$set": bson.M{
					"isBetLockedTripleFun": false,
				},
				"$unset": bson.M{
					"betsTripleFun":     "",
					"betStateTripleFun": "",
				},
			}
		}

		redis.SaveTFToRedis(
			user.UserId,
			user.TempWinnings3,
			finalWinningsTF,
			user.TempWinnings3 > 0,
		)

		uid := user.ID
		upd := update
		go func(id primitive.ObjectID, doc bson.M) {
			_, e := userCollection.UpdateByID(context.Background(), id, doc)
			if e != nil {
				log.Printf("❌ Async TF mongo update failed for %s: %v", id.Hex(), e)
			}
		}(uid, upd)

		updated++
	}

	log.Printf("✅ Finalized TripleFun winnings for %d users", updated)
	return nil
}

