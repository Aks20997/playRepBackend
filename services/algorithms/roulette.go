package algorithms

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	RouletteMultiplier = 36
	RouletteMaxNumber  = 37
)

// RouletteWinningAlgo calculates winning number based on bracket and total bets
func RouletteWinningAlgo(finalDict map[int]float64, mongoClient *mongo.Client, ctx context.Context) (int, error) {
	// Step 1: Fetch Roulette_Bracket
	recordCollection := mongoClient.Database("FunRepDB").Collection("Records")
	var configDoc struct {
		RouletteBracket float64 `bson:"Roulette_Bracket"`
	}

	objectID, _ := primitive.ObjectIDFromHex("6826546053fd676dfbbb32bc")
	err := recordCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&configDoc)

	bracket := float64(30000) // default
	if err == nil {
		bracket = configDoc.RouletteBracket
		log.Printf("📊 Loaded Roulette_Bracket from DB: %.2f", bracket)
	} else {
		log.Printf("⚠️ Failed to load Roulette_Bracket, using default: %.2f | error: %v", bracket, err)
	}

	// Step 2: Total amount
	totalAmount := 0.0
	for _, val := range finalDict {
		totalAmount += val
	}
	log.Printf("💰 Total betting amount: %.2f", totalAmount)

	// Step 3: Sort entries
	type kv struct {
		Key   int
		Value float64
	}
	var sorted []kv
	for k, v := range finalDict {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value < sorted[j].Value
	})

	// Step 4: Filter winning candidates
	var winningCandidates []int
	for _, entry := range sorted {
		payout := entry.Value * RouletteMultiplier
		if payout >= totalAmount && payout-totalAmount <= bracket {
			log.Printf("✅ Candidate: %d | Bet: %.2f | Payout: %.2f", entry.Key, entry.Value, payout)
			winningCandidates = append(winningCandidates, entry.Key)
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Step 5: Return from candidates
	if len(winningCandidates) > 0 {
		chosen := winningCandidates[rng.Intn(len(winningCandidates))]
		log.Printf("🎯 Chosen from winningCandidates: %d", chosen)
		return chosen, nil
	}

	// Step 6: Fallback
	fallbackLimit := 5
	if len(sorted) < 5 {
		fallbackLimit = len(sorted)
	}
	var fallbackOptions []int
	for i := 0; i < fallbackLimit; i++ {
		fallbackOptions = append(fallbackOptions, sorted[i].Key)
	}
	if len(fallbackOptions) > 0 {
		chosen := fallbackOptions[rng.Intn(len(fallbackOptions))]
		log.Printf("🎯 Fallback chosen from lowest %d: %d", fallbackLimit, chosen)
		return chosen, nil
	}

	log.Println("❌ No fallback options available")
	return 0, errors.New("no fallback options available")
}

// RouletteLosingAlgo calculates a losing number (minimizes payout)
func RouletteLosingAlgo(finalDict map[int]float64) (int, error) {
	// Prepare zeroArr
	zeroArr := []int{}
	for i := 0; i <= RouletteMaxNumber; i++ {
		val, ok := finalDict[i]
		if !ok || val == 0 {
			zeroArr = append(zeroArr, i)
		}
	}

	// Sort entries by value ascending
	type kv struct {
		Key   int
		Value float64
	}
	var sortedEntries []kv
	for k, v := range finalDict {
		sortedEntries = append(sortedEntries, kv{k, v})
	}
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Value < sortedEntries[j].Value
	})

	// Calculate totalAmount
	totalAmount := 0.0
	for _, v := range finalDict {
		totalAmount += v
	}

	// safeKeys: keys where value*36 < totalAmount
	safeKeys := []int{}
	for _, kv := range sortedEntries {
		if kv.Value*RouletteMultiplier < totalAmount {
			safeKeys = append(safeKeys, kv.Key)
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	if len(safeKeys) > 0 {
		randomIndex := rng.Intn(len(safeKeys))
		return safeKeys[randomIndex], nil
	}

	if len(zeroArr) > 0 {
		randomZeroIndex := rng.Intn(len(zeroArr))
		return zeroArr[randomZeroIndex], nil
	}

	// fallback: first key from sortedEntries
	if len(sortedEntries) > 0 {
		return sortedEntries[0].Key, nil
	}

	return 0, errors.New("finalDict is empty")
}

// RouletteZeroAlgo calculates a number with zero or minimal bets
func RouletteZeroAlgo(finalDict map[int]float64) (int, error) {
	// Prepare zeroArr
	zeroArr := []int{}
	for i := 0; i <= RouletteMaxNumber; i++ {
		val, ok := finalDict[i]
		if !ok || val == 0 {
			zeroArr = append(zeroArr, i)
		}
	}

	// Sort entries by value ascending
	type kv struct {
		Key   int
		Value float64
	}
	var sortedEntries []kv
	for k, v := range finalDict {
		sortedEntries = append(sortedEntries, kv{k, v})
	}
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Value < sortedEntries[j].Value
	})

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	if len(zeroArr) > 0 {
		randomZeroIndex := rng.Intn(len(zeroArr))
		return zeroArr[randomZeroIndex], nil
	}

	if len(sortedEntries) > 0 {
		return sortedEntries[0].Key, nil
	}

	return 0, errors.New("finalDict is empty")
}

// CalculateRouletteWinningNumber determines the winning number based on algorithm choice
func CalculateRouletteWinningNumber(finalDict map[int]float64, mongoClient *mongo.Client, ctx context.Context) (int, error) {
	// Fetch algorithm settings from "Records"
	recordCollection := mongoClient.Database("FunRepDB").Collection("Records")
	var algoConfig struct {
		AlgoChoice             string  `bson:"Roulette_Algo"`
		WinProbabilityRoulette float64 `bson:"Roulette_Winning_Probability"`
	}
	objectID, _ := primitive.ObjectIDFromHex("6826546053fd676dfbbb32bc")
	err := recordCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&algoConfig)
	if err != nil {
		algoConfig.AlgoChoice = "a"
		algoConfig.WinProbabilityRoulette = 30.0
	}

	winProb := algoConfig.WinProbabilityRoulette
	if winProb < 0 || winProb > 100 {
		winProb = 30.0
	}
	log.Printf("🎯 algoChoice: %s, winProb: %.2f", algoConfig.AlgoChoice, winProb)

	// Pick winning number based on algorithm
	randFloat := rand.Float64() * 100
	var winningNumber int

	switch algoConfig.AlgoChoice {
	case "a":
		if randFloat < winProb {
			log.Println("🎯 Chosen Algo: rouletteWinningAlgo (probability branch)")
			winningNumber, err = RouletteWinningAlgo(finalDict, mongoClient, ctx)
		} else if randFloat < winProb+10 {
			log.Println("🎯 Chosen Algo: rouletteZeroAlgo (mid probability branch)")
			winningNumber, err = RouletteZeroAlgo(finalDict)
		} else {
			log.Println("🎯 Chosen Algo: rouletteLosingAlgo (low probability branch)")
			winningNumber, err = RouletteLosingAlgo(finalDict)
		}
	case "l":
		log.Println("🎯 Chosen Algo: rouletteLosingAlgo (forced)")
		winningNumber, err = RouletteLosingAlgo(finalDict)
	case "z":
		log.Println("🎯 Chosen Algo: rouletteZeroAlgo (forced)")
		winningNumber, err = RouletteZeroAlgo(finalDict)
	case "w":
		log.Println("🎯 Chosen Algo: rouletteWinningAlgo (forced)")
		winningNumber, err = RouletteWinningAlgo(finalDict, mongoClient, ctx)
	default:
		if randFloat < winProb {
			log.Println("🎯 Chosen Algo: rouletteWinningAlgo (default - probability branch)")
			winningNumber, err = RouletteWinningAlgo(finalDict, mongoClient, ctx)
		} else if randFloat < winProb+10 {
			log.Println("🎯 Chosen Algo: rouletteZeroAlgo (default - mid probability branch)")
			winningNumber, err = RouletteZeroAlgo(finalDict)
		} else {
			log.Println("🎯 Chosen Algo: rouletteLosingAlgo (default - low probability branch)")
			winningNumber, err = RouletteLosingAlgo(finalDict)
		}
	}

	return winningNumber, err
}

// BuildRouletteBetDict builds the betting dictionary from user bets
func BuildRouletteBetDict(userCollection *mongo.Collection, ctx context.Context) (map[int]float64, error) {
	// Initialize finalDict with 0-37 keys and 0 values
	finalDict := make(map[int]float64)
	for i := 0; i <= RouletteMaxNumber; i++ {
		finalDict[i] = 0
	}

	// Fetch users with isBetLocked == true and winningsRoulette == 0
	filter := bson.M{"isBetLocked": true, "winningsRoulette": float64(0)}
	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		log.Println("❌ Error fetching users:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user struct {
			Bets map[string]float64 `bson:"bets"`
		}
		if err := cursor.Decode(&user); err != nil {
			log.Println("❌ Error decoding user bets:", err)
			continue
		}
		for k, v := range user.Bets {
			num, err := strconv.Atoi(k)
			if err == nil && num >= 0 && num <= RouletteMaxNumber {
				finalDict[num] += v
			}
		}
	}
	if err := cursor.Err(); err != nil {
		log.Println("❌ Cursor error:", err)
		return nil, err
	}

	return finalDict, nil
}

