package algorithms

import (
	"context"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// InverseWeightedAlgo calculates winning number and multiplier for FunTarget
func InverseWeightedAlgo(finalDict map[int]float64, winProb float64, capProb float64) (int, int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	winRound := rng.Float64()*100 < winProb
	multiplier := 9
	if rng.Float64()*100 < capProb {
		multiplier = 18
	}

	total := 0.0
	for _, v := range finalDict {
		total += v
	}
	avgBet := total / 10.0
	if avgBet == 0 {
		avgBet = 1
	}

	weightedList := []int{}
	for i := 0; i < 10; i++ {
		amount := finalDict[i]
		var baseWeight float64
		if winRound {
			baseWeight = (amount + 1) / (avgBet + 1)
		} else {
			baseWeight = (avgBet + 1) / (amount + 1)
		}
		randomFactor := 0.8 + rng.Float64()*0.4
		weight := int(math.Max(1, math.Round(baseWeight*randomFactor*3)))
		for j := 0; j < weight; j++ {
			weightedList = append(weightedList, i)
		}
	}

	chosen := weightedList[rng.Intn(len(weightedList))]
	return chosen, multiplier
}

// CalculateFunTargetWinningNumber calculates winning number and multiplier for FunTarget
func CalculateFunTargetWinningNumber(userCollection *mongo.Collection, ctx context.Context) (int, int, error) {
	recordCollection := userCollection.Database().Collection("Records")
	
	// Step 1: Fetch Algo & Probabilities
	var config struct {
		AlgoChoice string  `bson:"FunTarget_Algo"`
		WinProb    float64 `bson:"FunTarget_Winning_Probability"`
		CapProb    float64 `bson:"Cap_Probability"`
	}

	log.Printf("📊 Read Config - AlgoChoice: %s, WinProb: %.2f, CapProb: %.2f", config.AlgoChoice, config.WinProb, config.CapProb)

	objectID, _ := primitive.ObjectIDFromHex("6826546053fd676dfbbb32bc")
	err := recordCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&config)
	if err != nil {
		log.Printf("⚠️ Using default algo config: %v", err)
		config.AlgoChoice = "i"
		config.WinProb = 20
		config.CapProb = 10
	}

	// Step 2: Fetch users and build finalDict
	userFilter := bson.M{
		"isBetLockedFunTarget": true,
		"winningsFunTarget":    float64(0),
	}
	cursor, err := userCollection.Find(ctx, userFilter)
	if err != nil {
		log.Printf("❌ Error fetching users: %v", err)
		return 0, 0, err
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
		if err := cursor.Decode(&u); err != nil {
			log.Printf("❌ Decode user failed: %v", err)
			continue
		}

		for key, val := range u.BetsFT {
			num, _ := strconv.Atoi(key)
			if num >= 0 && num < 10 {
				finalDict[num] += val
			}
		}
	}

	// Step 3: Run algorithm
	var winningFT, multiplier int
	switch config.AlgoChoice {
	case "i":
		winningFT, multiplier = InverseWeightedAlgo(finalDict, config.WinProb, config.CapProb)
	default:
		winningFT, multiplier = rand.Intn(10), 9
	}

	log.Printf("✅ Processed winningFT = %d, multiplier = %d", winningFT, multiplier)
	return winningFT, multiplier, nil
}

