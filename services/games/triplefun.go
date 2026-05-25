package games

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"FunRepBackend/services/winnings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TripleFunService handles TripleFun game operations
type TripleFunService struct {
	mongoClient        *mongo.Client
	userCollection     *mongo.Collection
	realtimeCollection *mongo.Collection
}

func NewTripleFunService(mongoClient *mongo.Client) *TripleFunService {
	db := mongoClient.Database("FunRepDB")
	return &TripleFunService{
		mongoClient:        mongoClient,
		userCollection:     db.Collection("Users"),
		realtimeCollection: db.Collection("RealtimeData"),
	}
}

// UpdateNextWinningNumber generates and updates the next winning number for TripleFun
func (tfs *TripleFunService) UpdateNextWinningNumber(ctx context.Context) error {
	// Step 1: Generate random 3-digit number (000–999)
	rand.Seed(time.Now().UnixNano())
	randomNumber := fmt.Sprintf("%03d", rand.Intn(1000))
	State.SetCurrentWinningNumber("triplefun", randomNumber)

	// Step 2: Save nextWinningNumberTF
	objID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	_, err := tfs.realtimeCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
		"$set": bson.M{
			"nextWinningNumberTF": randomNumber,
		},
	})
	if err != nil {
		log.Printf("❌ Failed to update TF number: %v", err)
		return err
	}

	// Step 3: Fetch all users
	userFilter := bson.M{
		"isBetLockedTripleFun": true,
		"winningsTripleFun":    float64(0),
	}
	cursor, err := tfs.userCollection.Find(ctx, userFilter)
	if err != nil {
		log.Printf("❌ Error fetching TF users: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	// Step 4: Prepare match patterns
	last1 := randomNumber[2:]
	last2 := randomNumber[1:]
	last3 := randomNumber

	// Track PnL but skip demo users in calculation
	totalBets := 0.0
	totalPayout := 0.0

	// Step 5: Loop users
	for cursor.Next(ctx) {
		var u struct {
			ID     primitive.ObjectID `bson:"_id"`
			BetsTF map[string]float64 `bson:"betsTripleFun"`
			Demo   bool               `bson:"exclude"`
		}
		if err := cursor.Decode(&u); err != nil {
			log.Printf("❌ Failed to decode user: %v", err)
			continue
		}

		var winnings float64

		// Single digit check
		if val, ok := u.BetsTF[last1]; ok {
			winnings += val * 9
		}
		// Two digit check
		if val, ok := u.BetsTF[last2]; ok {
			winnings += val * 90
		}
		// Three digit check
		if val, ok := u.BetsTF[last3]; ok {
			winnings += val * 900
		}

		// Save TempWinningsTF
		winnings = math.Round(winnings*100) / 100
		_, err := tfs.userCollection.UpdateByID(ctx, u.ID, bson.M{
			"$set": bson.M{"TempWinningsTF": winnings},
		})
		if err != nil {
			log.Printf("❌ Failed to update winnings for user %v: %v", u.ID.Hex(), err)
		}

		// PnL calculation — skip demo users
		if !u.Demo {
			for _, bet := range u.BetsTF {
				totalBets += bet
			}
			totalPayout += winnings
		}
	}

	if err := cursor.Err(); err != nil {
		log.Printf("❌ Cursor error: %v", err)
		return err
	}

	// Step 6: Save PnL (only from non-demo users)
	tfs.recordPnL(ctx, totalBets, totalPayout)

	log.Printf("✅ Generated TF number %s | PnL: %.2f", randomNumber, math.Floor(totalBets-totalPayout))
	return nil
}

// FinalizeWinningNumber finalizes the winning number and updates history
func (tfs *TripleFunService) FinalizeWinningNumber(ctx context.Context) error {
	winningNumber := State.GetCurrentWinningNumber("triplefun")
	if winningNumber == nil {
		return fmt.Errorf("no winning number set for triplefun")
	}
	winningStr := winningNumber.(string)

	objID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	update := bson.M{
		"$set": bson.M{
			"winningNumberTF": winningStr,
		},
		"$push": bson.M{
			"tripleFunHistory": bson.M{
				"$each":     []interface{}{winningStr},
				"$slice":    -5,
				"$position": 6,
			},
		},
	}

	_, err := tfs.realtimeCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		log.Printf("❌ Failed to update TF winning values: %v", err)
		return err
	}

	log.Printf("✅ Finalized TripleFun winning number: %s", winningStr)
	return nil
}

// TransferWinnings transfers winnings to users
func (tfs *TripleFunService) TransferWinnings(ctx context.Context) error {
	return winnings.TransferTripleFunWinnings(tfs.userCollection, ctx)
}

// recordPnL records profit and loss for the round
func (tfs *TripleFunService) recordPnL(ctx context.Context, totalBets, totalPayout float64) {
	pnl := math.Floor(totalBets - totalPayout)
	nowIST := time.Now().In(time.FixedZone("IST", 5.5*3600))
	docID := nowIST.Format("2006-01-02")

	pnlCollection := tfs.mongoClient.Database("FunRepDB").Collection("PNL")
	filter := bson.M{"_id": docID}
	update := bson.M{
		"$inc": bson.M{
			"TripleFunPnL": pnl,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, _ = pnlCollection.UpdateOne(ctx, filter, update, opts)
}

