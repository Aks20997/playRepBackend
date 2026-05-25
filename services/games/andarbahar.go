package games

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"FunRepBackend/services/winnings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AndarBaharService handles AndarBahar game operations
type AndarBaharService struct {
	mongoClient        *mongo.Client
	userCollection     *mongo.Collection
	realtimeCollection *mongo.Collection
}

func NewAndarBaharService(mongoClient *mongo.Client) *AndarBaharService {
	db := mongoClient.Database("FunRepDB")
	return &AndarBaharService{
		mongoClient:        mongoClient,
		userCollection:     db.Collection("Users"),
		realtimeCollection: db.Collection("RealtimeData"),
	}
}

// UpdateNextWinningNumber generates and updates the next winning number for AndarBahar
func (abs *AndarBaharService) UpdateNextWinningNumber(ctx context.Context) error {
	// 1) Generate random [1..52]
	rand.Seed(time.Now().UnixNano())
	winningNum := rand.Intn(52) + 1
	State.SetCurrentWinningNumber("andarbahar", winningNum)

	// 2) Generate random unique ABArray ending with winningNum
	ABArray := make([]int, 0, 26)
	temp := make([]int, 0, 51)

	// Add all numbers except winningNum
	for i := 1; i <= 52; i++ {
		if i != winningNum {
			temp = append(temp, i)
		}
	}

	rand.Seed(time.Now().UnixNano())
	arrayLen := rand.Intn(25) + 2 // 2..26

	// Shuffle temp and pick first (arrayLen-1) numbers
	rand.Shuffle(len(temp), func(i, j int) { temp[i], temp[j] = temp[j], temp[i] })
	ABArray = append(ABArray, temp[:arrayLen-1]...)
	ABArray = append(ABArray, winningNum)

	// 3) Persist nextWinningNumberAB
	objID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	_, err := abs.realtimeCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
		"$set": bson.M{
			"nextWinningNumberAB": winningNum,
			"nextABArray":         ABArray,
		},
	})
	if err != nil {
		log.Printf("❌ Failed to update AB number: %v", err)
		return err
	}

	// 4) Filter users
	userFilter := bson.M{"isBetLockedAB": true, "winningsAB": float64(0)}
	cursor, err := abs.userCollection.Find(ctx, userFilter)
	if err != nil {
		log.Printf("❌ Error fetching AB users: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	// 5) Derive outcomes
	numeric := ((winningNum - 1) % 13) + 1
	color := "red"
	if winningNum > 26 {
		color = "black"
	}
	suit := ""
	switch {
	case winningNum >= 1 && winningNum <= 13:
		suit = "heart"
	case winningNum >= 14 && winningNum <= 26:
		suit = "diamond"
	case winningNum >= 27 && winningNum <= 39:
		suit = "spades"
	case winningNum >= 40 && winningNum <= 52:
		suit = "club"
	}

	// 6) PnL accumulators
	totalBets := 0.0
	totalPayout := 0.0

	const (
		numMultiplier   = 12.0
		colorMultiplier = 1.95
		suitMultiplier  = 3.75
		bulkMultiplier  = 2.0
	)

	normalizeKey := func(k string) string {
		return strings.ToLower(strings.TrimSpace(k))
	}
	eqSuit := func(k string) bool {
		k = normalizeKey(k)
		if k == "heart" || k == "hearts" {
			return suit == "heart"
		}
		if k == "diamond" || k == "diamonds" {
			return suit == "diamond"
		}
		if k == "spade" || k == "spades" {
			return suit == "spades"
		}
		if k == "club" || k == "clubs" {
			return suit == "club"
		}
		return false
	}

	for cursor.Next(ctx) {
		var u struct {
			ID     primitive.ObjectID `bson:"_id"`
			BetsAB map[string]float64 `bson:"betsAB"`
			Demo   bool               `bson:"exclude"`
		}
		if err := cursor.Decode(&u); err != nil {
			log.Printf("❌ Failed to decode user: %v", err)
			continue
		}
		if u.BetsAB == nil {
			u.BetsAB = map[string]float64{}
		}

		winnings := 0.0
		andarBaharWin := 0.0

		// Numeric
		if val, ok := u.BetsAB[strconv.Itoa(numeric)]; ok {
			winnings += val * numMultiplier
		}
		// Color
		if val, ok := u.BetsAB[color]; ok {
			winnings += val * colorMultiplier
		}
		// Suit
		for k, v := range u.BetsAB {
			if eqSuit(k) {
				winnings += v * suitMultiplier
			}
		}

		// Ato6 (Ace=1 → 6)
		if val, ok := u.BetsAB["Ato6"]; ok {
			if numeric >= 1 && numeric <= 6 {
				winnings += val * bulkMultiplier
			}
		}

		if val, ok := u.BetsAB["Seven"]; ok {
			if numeric >= 1 && numeric <= 6 {
				winnings += val * numMultiplier
			}
		}

		// 8toK (8 → 13)
		if val, ok := u.BetsAB["8toK"]; ok {
			if numeric >= 8 && numeric <= 13 {
				winnings += val * bulkMultiplier
			}
		}

		// Andar/Bahar logic
		if arrayLen%2 == 0 {
			// Even → Bahar wins
			if val, ok := u.BetsAB["bahar"]; ok {
				andarBaharWin = val * 2
			}
		} else {
			// Odd → Andar wins
			if val, ok := u.BetsAB["andar"]; ok {
				andarBaharWin = val * 2
			}
		}

		// Save winnings
		winnings = math.Round(winnings*100) / 100
		andarBaharWin = math.Round(andarBaharWin*100) / 100
		totalWinnings := winnings + andarBaharWin
		if _, err := abs.userCollection.UpdateByID(ctx, u.ID, bson.M{
			"$set": bson.M{
				"TempWinningsAB":      winnings,
				"TempTotalWinningsAB": totalWinnings,
			},
		}); err != nil {
			log.Printf("❌ Failed to update winnings for user %v: %v", u.ID.Hex(), err)
		}

		// PnL
		if !u.Demo {
			for _, bet := range u.BetsAB {
				totalBets += bet
			}
			totalPayout += winnings + andarBaharWin
		}
	}
	if err := cursor.Err(); err != nil {
		log.Printf("❌ Cursor error: %v", err)
		return err
	}

	// 7) Save PnL
	abs.recordPnL(ctx, totalBets, totalPayout)

	log.Printf("✅ Generated AB number %d (num:%d color:%s suit:%s) | ABArrayLen:%d | PnL: %.2f",
		winningNum, numeric, color, suit, arrayLen, math.Floor(totalBets-totalPayout))

	return nil
}

// FinalizeWinningNumber finalizes the winning number and updates history
func (abs *AndarBaharService) FinalizeWinningNumber(ctx context.Context) error {
	winningNumber := State.GetCurrentWinningNumber("andarbahar")
	if winningNumber == nil {
		return fmt.Errorf("no winning number set for andarbahar")
	}
	winningNum := winningNumber.(int)

	objID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	update := bson.M{
		"$set": bson.M{
			"winningNumberAB": winningNum,
		},
		"$push": bson.M{
			"ABHistory": bson.M{
				"$each":     []interface{}{winningNum},
				"$slice":    -5,
				"$position": 6,
			},
		},
	}

	_, err := abs.realtimeCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		log.Printf("❌ Failed to update AB winning values: %v", err)
		return err
	}

	log.Printf("✅ Finalized AndarBahar winning number: %d", winningNum)
	return nil
}

// TransferWinnings transfers winnings to users
func (abs *AndarBaharService) TransferWinnings(ctx context.Context) error {
	winningNumber := State.GetCurrentWinningNumber("andarbahar")
	if winningNumber == nil {
		return fmt.Errorf("no winning number set for andarbahar")
	}
	return winnings.TransferAndarBaharWinnings(abs.userCollection, ctx, winningNumber.(int))
}

// recordPnL records profit and loss for the round
func (abs *AndarBaharService) recordPnL(ctx context.Context, totalBets, totalPayout float64) {
	pnl := math.Floor(totalBets - totalPayout)
	nowIST := time.Now().In(time.FixedZone("IST", 19800))
	docID := nowIST.Format("2006-01-02")

	pnlCollection := abs.mongoClient.Database("FunRepDB").Collection("PNL")
	filter := bson.M{"_id": docID}
	update := bson.M{"$inc": bson.M{"AndarBaharPnL": pnl}}
	opts := options.Update().SetUpsert(true)
	if _, err := pnlCollection.UpdateOne(ctx, filter, update, opts); err != nil {
		log.Println("❌ Error updating PnL record:", err)
	}
}

