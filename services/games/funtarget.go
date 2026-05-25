package games

import (
	"context"
	"log"
	"math"
	"strconv"
	"time"

	"FunRepBackend/services/algorithms"
	"FunRepBackend/services/winnings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FunTargetService handles FunTarget game operations
type FunTargetService struct {
	mongoClient        *mongo.Client
	userCollection     *mongo.Collection
	realtimeCollection *mongo.Collection
}

func NewFunTargetService(mongoClient *mongo.Client) *FunTargetService {
	db := mongoClient.Database("FunRepDB")
	return &FunTargetService{
		mongoClient:        mongoClient,
		userCollection:     db.Collection("Users"),
		realtimeCollection: db.Collection("RealtimeData"),
	}
}

// UpdateNextWinningNumber calculates and updates the next winning number and multiplier
func (fts *FunTargetService) UpdateNextWinningNumber(ctx context.Context) error {
	// Use the algorithm service
	winningFT, multiplier, err := algorithms.CalculateFunTargetWinningNumber(fts.userCollection, ctx)
	if err != nil {
		return err
	}

	// Update game state
	State.SetCurrentWinningNumber("funtarget", winningFT)
	State.SetMultiplier(multiplier)

	// Save FT values to RealtimeData
	objID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	_, err = fts.realtimeCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
		"$set": bson.M{
			"nextWinningNumberFT": winningFT,
			"nextMultiplier":      multiplier,
		},
	})
	if err != nil {
		log.Printf("❌ Failed to update FT values: %v", err)
		return err
	}

	// Calculate and save user winnings
	userFilter := bson.M{
		"isBetLockedFunTarget": true,
		"winningsFunTarget":    float64(0),
	}
	cursor, err := fts.userCollection.Find(ctx, userFilter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var users []struct {
		ID     primitive.ObjectID `bson:"_id"`
		BetsFT map[string]float64 `bson:"betsFT"`
	}
	for cursor.Next(ctx) {
		var u struct {
			ID     primitive.ObjectID `bson:"_id"`
			BetsFT map[string]float64 `bson:"betsFT"`
		}
		if err := cursor.Decode(&u); err != nil {
			continue
		}
		users = append(users, u)
	}

	for _, u := range users {
		winKey := strconv.Itoa(winningFT)
		winAmount := float64(0)
		if val, ok := u.BetsFT[winKey]; ok {
			if multiplier == 18 {
				winAmount = math.Round(val*18*100) / 100
			} else {
				winAmount = math.Round(val*9*100) / 100
			}
		}
		_, _ = fts.userCollection.UpdateByID(ctx, u.ID, bson.M{"$set": bson.M{"TempWinnings2": winAmount}})
	}

	// Record PnL
	fts.recordPnL(ctx, winningFT, multiplier)

	log.Printf("✅ Processed winningFT = %d, multiplier = %d", winningFT, multiplier)
	return nil
}

// FinalizeWinningNumber finalizes the winning number and updates history
func (fts *FunTargetService) FinalizeWinningNumber(ctx context.Context) error {
	winningFT := State.GetCurrentWinningNumber("funtarget")
	multiplier := State.CurrentNextMultiplier

	if winningFT == nil {
		return nil
	}
	winningNum := winningFT.(int)

	// Update game state to move next to current
	State.SetCurrentWinningNumber("funtarget", winningNum)

	objID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	update := bson.M{
		"$set": bson.M{
			"winningNumberFT": winningNum,
			"multiplier":      multiplier,
		},
		"$push": bson.M{
			"funTargetHistory": bson.M{
				"$each":     []interface{}{winningNum},
				"$slice":    -10,
				"$position": 11,
			},
		},
	}

	_, err := fts.realtimeCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		log.Printf("❌ Failed to update FT winning values: %v", err)
		return err
	}

	log.Printf("✅ Finalized FunTarget winning number: %d, multiplier: %d", winningNum, multiplier)
	return nil
}

// TransferWinnings transfers winnings to users
func (fts *FunTargetService) TransferWinnings(ctx context.Context) error {
	winningFT := State.GetCurrentWinningNumber("funtarget")
	if winningFT == nil {
		return nil
	}
	winningNum := winningFT.(int)
	return winnings.TransferFunTargetWinnings(fts.userCollection, ctx, winningNum)
}

// recordPnL records profit and loss for the round
func (fts *FunTargetService) recordPnL(ctx context.Context, winningFT, multiplier int) {
	// Build finalDict for PnL calculation
	userFilter := bson.M{
		"isBetLockedFunTarget": true,
		"winningsFunTarget":    float64(0),
	}
	cursor, err := fts.userCollection.Find(ctx, userFilter)
	if err != nil {
		log.Printf("❌ Failed to find users for PnL calculation: %v", err)
		return
	}
	defer cursor.Close(ctx)

	finalDict := make(map[int]float64)
	for i := 0; i < 10; i++ {
		finalDict[i] = 0
	}

	for cursor.Next(ctx) {
		var u struct {
			BetsFT map[string]float64 `bson:"betsFT"`
		}
		if err := cursor.Decode(&u); err == nil {
			for key, val := range u.BetsFT {
				num, _ := strconv.Atoi(key)
				if num >= 0 && num < 10 {
					finalDict[num] += val
				}
			}
		}
	}

	totalAmount := 0.0
	for _, v := range finalDict {
		totalAmount += v
	}
	spentAmount := finalDict[winningFT] * float64(multiplier)
	pnl := math.Floor(totalAmount - spentAmount)

	// Ignore Demo Users in PnL
	var demoUsersTotal, demoUsersSpent float64
	demoCursor, err := fts.userCollection.Find(ctx, bson.M{"exclude": true, "isBetLockedFunTarget": true})
	if err == nil && demoCursor != nil {
		for demoCursor.Next(ctx) {
			var du struct {
				BetsFT map[string]float64 `bson:"betsFT"`
			}
			if err := demoCursor.Decode(&du); err == nil {
				for k, v := range du.BetsFT {
					num, _ := strconv.Atoi(k)
					demoUsersTotal += v
					if num == winningFT {
						demoUsersSpent += v * float64(multiplier)
					}
				}
			}
		}
		demoCursor.Close(ctx)
	}

	pnl -= math.Floor(demoUsersTotal - demoUsersSpent)

	nowIST := time.Now().In(time.FixedZone("IST", 5.5*3600))
	docID := nowIST.Format("2006-01-02")

	pnlCollection := fts.mongoClient.Database("FunRepDB").Collection("PNL")
	filter := bson.M{"_id": docID}
	update := bson.M{
		"$inc": bson.M{
			"FunTargetPnL": pnl,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, _ = pnlCollection.UpdateOne(ctx, filter, update, opts)
}

