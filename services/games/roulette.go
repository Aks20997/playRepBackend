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

// RouletteService handles roulette game operations
type RouletteService struct {
	mongoClient *mongo.Client
	userCollection *mongo.Collection
	realtimeCollection *mongo.Collection
}

func NewRouletteService(mongoClient *mongo.Client) *RouletteService {
	db := mongoClient.Database("FunRepDB")
	return &RouletteService{
		mongoClient: mongoClient,
		userCollection: db.Collection("Users"),
		realtimeCollection: db.Collection("RealtimeData"),
	}
}

// UpdateNextWinningNumber calculates and updates the next winning number
func (rs *RouletteService) UpdateNextWinningNumber(ctx context.Context) error {
	// Build betting dictionary
	finalDict, err := algorithms.BuildRouletteBetDict(rs.userCollection, ctx)
	if err != nil {
		return err
	}

	// Calculate winning number
	winningNumber, err := algorithms.CalculateRouletteWinningNumber(finalDict, rs.mongoClient, ctx)
	if err != nil {
		log.Println("❌ Error generating winning number:", err)
		return err
	}

	// Update game state
	State.SetCurrentWinningNumber("roulette", winningNumber)

	// Update nextWinningNumber in RealtimeData
	realtimeID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	update := bson.M{"$set": bson.M{"nextWinningNumber": winningNumber}}
	_, err = rs.realtimeCollection.UpdateOne(ctx, bson.M{"_id": realtimeID}, update)
	if err != nil {
		log.Println("❌ Error updating nextWinningNumber:", err)
		return err
	}

	// Update TempWinnings1 for users
	filter := bson.M{"isBetLocked": true, "winningsRoulette": float64(0)}
	cursor, err := rs.userCollection.Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user struct {
			ID   string             `bson:"Id"`
			Bets map[string]float64 `bson:"bets"`
		}
		if err := cursor.Decode(&user); err != nil {
			continue
		}

		key := strconv.Itoa(winningNumber)
		betAmount, ok := user.Bets[key]
		if ok {
			tempWinnings := math.Round(betAmount*36*100) / 100
			update := bson.M{"$set": bson.M{"TempWinnings1": tempWinnings}}
			_, _ = rs.userCollection.UpdateOne(ctx, bson.M{"Id": user.ID}, update)
		}
	}

	// Record PnL
	rs.recordPnL(ctx, finalDict, winningNumber)

	return nil
}

// FinalizeWinningNumber finalizes the winning number and updates history
func (rs *RouletteService) FinalizeWinningNumber(ctx context.Context) error {
	winningNumber := State.GetCurrentWinningNumber("roulette").(int)
	
	realtimeID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	update := bson.M{
		"$set": bson.M{
			"winningNumber": winningNumber,
		},
		"$push": bson.M{
			"rouletteHistory": bson.M{
				"$each":     []interface{}{winningNumber},
				"$slice":    -5,
				"$position": 6,
			},
		},
	}

	_, err := rs.realtimeCollection.UpdateOne(ctx, bson.M{"_id": realtimeID}, update)
	if err != nil {
		return err
	}

	log.Printf("✅ Finalized winning number: %d", winningNumber)
	return nil
}

// TransferWinnings transfers winnings to users
func (rs *RouletteService) TransferWinnings(ctx context.Context) error {
	return winnings.TransferRouletteWinnings(rs.userCollection, ctx, State.GetCurrentWinningNumber("roulette").(int))
}

// recordPnL records profit and loss for the round
func (rs *RouletteService) recordPnL(ctx context.Context, finalDict map[int]float64, winningNumber int) {
	totalAmount := 0.0
	for _, v := range finalDict {
		totalAmount += v
	}
	spentAmount := finalDict[winningNumber] * 36
	pnl := math.Floor(totalAmount - spentAmount)

	// Ignore Demo Users in PnL
	var demoUsersTotal, demoUsersSpent float64
	demoCursor, _ := rs.userCollection.Find(ctx, bson.M{"exclude": true, "isBetLocked": true})
	for demoCursor.Next(ctx) {
		var du struct {
			Bets map[string]float64 `bson:"bets"`
		}
		if err := demoCursor.Decode(&du); err == nil {
			for k, v := range du.Bets {
				num, _ := strconv.Atoi(k)
				demoUsersTotal += v
				if num == winningNumber {
					demoUsersSpent += v * 36
				}
			}
		}
	}
	demoCursor.Close(ctx)

	pnl -= math.Floor(demoUsersTotal - demoUsersSpent)

	nowIST := time.Now().In(time.FixedZone("IST", 5.5*3600))
	docID := nowIST.Format("2006-01-02")

	pnlCollection := rs.mongoClient.Database("FunRepDB").Collection("PNL")
	filter := bson.M{"_id": docID}
	update := bson.M{
		"$inc": bson.M{
			"RoulettePnL": pnl,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, _ = pnlCollection.UpdateOne(ctx, filter, update, opts)
}

