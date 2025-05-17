package main

import (
	"context"
	"errors"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"time"

	"FunRepBackend/routes"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Countdown struct {
	Countdown1 int `json:"countdown1"`
	Countdown2 int `json:"countdown2"`
}

var countdown1 int
var countdown2 int
var nextRoundTime int64
var nextRoundTime2 int64
var currentNextWinningNumber int
var currentNextWinningNumberFT int
var currentNextMultiplier int
var lastKey string
var lastKeyRoulette string
var mongoClient *mongo.Client

func StartCountdowns() {
	countdown1 = 25
	countdown2 = 10

	for {
		tick()
		time.Sleep(1 * time.Second)
	}
}

func tick() {
	handleTimer1(countdown1)
	handleTimer2(countdown2)

	countdown1--
	countdown2--

	if countdown1 < 0 {
		countdown1 = 59
	}
	if countdown2 < 0 {
		countdown2 = 59
	}
}

// --- Timer 1 handler ---
func handleTimer1(value int) {
	switch value {
	case 0:
		updateNextRoundTime1()
		TriggerFinalizeWinningNumber()
		callTransferWinnings()
	case 7:
		callUpdateNextWinningNumber()
	}
}

// --- Timer 2 handler ---
func handleTimer2(value int) {
	switch value {
	case 0:
		updateNextRoundTime2()
		TriggerFinalizeWinningNumberFT()
		callTransferWinningsFT()
	case 7:
		callUpdateWinningNumberFT()
	}
}

// --- Actions ---
func updateNextRoundTime1() {
	nextRoundTime = time.Now().Unix() + 60
	timestampStr := strconv.FormatInt(nextRoundTime, 10) // convert the timestamp to a string

	// Find the collection and field
	collection := mongoClient.Database("FunRepDB").Collection("RealtimeData")

	// Convert hex string to ObjectID
	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		log.Printf("❌ Failed to convert hex to ObjectID: %v", err)
		return
	}

	// Find the existing document
	var result bson.M
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&result)
	if err != nil {
		log.Printf("❌ Failed to find the document: %v", err)
		return
	}

	// Get roundsData2 if it exists
	roundsData1, exists := result["roundsData1"].(bson.M)
	if !exists {
		// Initialize roundsData2 if it doesn't exist
		roundsData1 = bson.M{}
	}

	// Check if the last timestamp exists

	if lastKeyRoulette != "" {
		// Update the last entry with the currentNextWinningNumber value
		roundsData1[lastKeyRoulette] = currentNextWinningNumber
		log.Printf("✅ Updated last entry with currentNextWinningNumber: %d", currentNextWinningNumber)
	}

	// Add a new timestamp entry with "NOT OPEN"
	roundsData1[timestampStr] = "NOT OPEN"
	lastKeyRoulette = timestampStr
	log.Printf("✅ Added new entry with timestamp %s and status: NOT OPEN", timestampStr)

	// Prepare the update document
	update := bson.M{
		"$set": bson.M{
			"roundsData1": roundsData1, // Update roundsData2 with the new timestamp and status
		},
	}

	// Apply the update to MongoDB
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, update)
	if err != nil {
		log.Printf("❌ Failed to update roundsData2: %v", err)
	} else {
		log.Printf("✅ Updated roundsData2 with timestamp %s", timestampStr)
	}
}

func updateNextRoundTime2() {
	// Get the next round time (current time + 60 seconds)
	nextRoundTime2 = time.Now().Unix() + 60
	timestampStr := strconv.FormatInt(nextRoundTime2, 10) // convert the timestamp to a string

	// Find the collection and field
	collection := mongoClient.Database("FunRepDB").Collection("RealtimeData")

	// Convert hex string to ObjectID
	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	if err != nil {
		log.Printf("❌ Failed to convert hex to ObjectID: %v", err)
		return
	}

	// Find the existing document
	var result bson.M
	err = collection.FindOne(context.Background(), bson.M{"_id": objectID}).Decode(&result)
	if err != nil {
		log.Printf("❌ Failed to find the document: %v", err)
		return
	}

	// Get roundsData2 if it exists
	roundsData2, exists := result["roundsData2"].(bson.M)
	if !exists {
		// Initialize roundsData2 if it doesn't exist
		roundsData2 = bson.M{}
	}

	// Check if the last timestamp exists

	if lastKey != "" {
		// Update the last entry with the currentNextWinningNumberFT value
		roundsData2[lastKey] = currentNextWinningNumberFT
		log.Printf("✅ Updated last entry with currentNextWinningNumberFT: %d", currentNextWinningNumberFT)
	}

	// Add a new timestamp entry with "NOT OPEN"
	roundsData2[timestampStr] = "NOT OPEN"
	lastKey = timestampStr
	log.Printf("✅ Added new entry with timestamp %s and status: NOT OPEN", timestampStr)

	// Prepare the update document
	update := bson.M{
		"$set": bson.M{
			"roundsData2": roundsData2, // Update roundsData2 with the new timestamp and status
		},
	}

	// Apply the update to MongoDB
	_, err = collection.UpdateOne(context.Background(), bson.M{"_id": objectID}, update)
	if err != nil {
		log.Printf("❌ Failed to update roundsData2: %v", err)
	} else {
		log.Printf("✅ Updated roundsData2 with timestamp %s", timestampStr)
	}
}

func callUpdateNextWinningNumber() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := UpdateNextWinningNumber(mongoClient, ctx)
	if err != nil {
		log.Println("Failed to update nextWinningNumber:", err)
	}
}

func callUpdateWinningNumberFT() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := UpdateNextMultiplierAndWinningFT(ctx)
	if err != nil {
		log.Println("❌ Error updating FT values:", err)
	} else {
		log.Println("✅ FT values updated successfully.")
	}
}

func callTransferWinnings() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := TransferWinnings(ctx)
	if err != nil {
		log.Println("❌ Error updating FT values:", err)
	} else {
		log.Println("✅ FT values updated successfully.")
	}
}

func callTransferWinningsFT() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := TransferWinningsFT(ctx)
	if err != nil {
		log.Println("❌ Error updating FT values:", err)
	} else {
		log.Println("✅ FT values updated successfully.")
	}
}

func FinalizeWinningNumber(ctx context.Context) error {
	collection := mongoClient.Database("FunRepDB").Collection("RealtimeData")
	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	filter := bson.M{"_id": objectID}

	// Create the update document
	update := bson.M{
		"$set": bson.M{
			"winningNumber": currentNextWinningNumber,
		},
		"$push": bson.M{
			"rouletteHistory": bson.M{
				"$each":     []interface{}{currentNextWinningNumber},
				"$slice":    -5, // Keep the last 5 elements in the array
				"$position": 6,  // Add new value at the end of the array
			},
		},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("❌ Failed to update winningNumber: %v", err)
		return err
	}

	log.Printf("✅ Transferred nextWinningNumber (%d) to winningNumber and updated winningNumberHistory", currentNextWinningNumber)
	return nil
}

func TriggerFinalizeWinningNumber() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := FinalizeWinningNumber(ctx)
	if err != nil {
		log.Println("❌ Error finalizing winning number:", err)
	}
}

func FinalizeWinningNumberFT(ctx context.Context) error {
	log.Printf("currentNextWinningNumberFt : %d and currentNextMultiplier : %d", currentNextWinningNumberFT, currentNextMultiplier)

	collection := mongoClient.Database("FunRepDB").Collection("RealtimeData")
	objectID, err := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	filter := bson.M{"_id": objectID}

	// Create the update document
	update := bson.M{
		"$set": bson.M{
			"winningNumberFT": currentNextWinningNumberFT,
			"multiplier":      currentNextMultiplier,
		},
		"$push": bson.M{
			"funTargetHistory": bson.M{
				"$each":     []interface{}{currentNextWinningNumberFT},
				"$slice":    -10, // Keep the last 10 elements in the array (negative for reverse order)
				"$position": 11,  // Add new value at the end of the array
			},
		},
	}

	_, err = collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("❌ Failed to update FT winning values: %v", err)
		return err
	}

	log.Printf("✅ Transferred nextWinningNumberFT (%d) and multiplier (%d), and updated funTargetHistory", currentNextWinningNumberFT, currentNextMultiplier)
	return nil
}

func TriggerFinalizeWinningNumberFT() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := FinalizeWinningNumberFT(ctx)
	if err != nil {
		log.Println("❌ Error finalizing FT values:", err)
	}
}

func rouletteWinningAlgo(finalDict map[int]float64, mongoClient *mongo.Client, ctx context.Context) (int, error) {
	const multiplier = 36

	// 🧮 Step 1: Fetch bracket from MongoDB
	recordCollection := mongoClient.Database("FunRepDB").Collection("Record")
	var configDoc struct {
		RouletteBracket float64 `bson:"rouletteBracket"`
	}

	err := recordCollection.FindOne(ctx, bson.M{"_id": "Roulette_Bracket"}).Decode(&configDoc)
	bracket := float64(30000) // default
	if err == nil {
		bracket = configDoc.RouletteBracket
	} else {
		log.Println("⚠️ Using default bracket due to error:", err)
	}

	// 🧮 Step 2: Total amount
	totalAmount := 0.0
	for _, val := range finalDict {
		totalAmount += val
	}

	// 🧮 Step 3: Sort entries by value (ascending)
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

	// 🧮 Step 4: Filter winning candidates
	var winningCandidates []int
	for _, entry := range sorted {
		payout := entry.Value * multiplier
		if payout >= totalAmount && payout-totalAmount <= bracket {
			winningCandidates = append(winningCandidates, entry.Key)
		}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 🧮 Step 5: Return random from candidates if any
	if len(winningCandidates) > 0 {
		return winningCandidates[rng.Intn(len(winningCandidates))], nil
	}

	// 🧮 Step 6: Fallback from lowest 5 values
	fallbackLimit := 5
	if len(sorted) < 5 {
		fallbackLimit = len(sorted)
	}
	var fallbackOptions []int
	for i := 0; i < fallbackLimit; i++ {
		fallbackOptions = append(fallbackOptions, sorted[i].Key)
	}

	if len(fallbackOptions) > 0 {
		return fallbackOptions[rng.Intn(len(fallbackOptions))], nil
	}

	return 0, errors.New("no fallback options available")
}

func rouletteLosingAlgo(finalDict map[int]float64) (int, error) {
	const multiplier = 36

	// Prepare zeroArr
	zeroArr := []int{}
	for i := 0; i <= 37; i++ {
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
		if kv.Value*multiplier < totalAmount {
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

func rouletteZeroAlgo(finalDict map[int]float64) (int, error) {
	// Prepare zeroArr
	zeroArr := []int{}
	for i := 0; i <= 37; i++ {
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

func UpdateNextWinningNumber(mongoClient *mongo.Client, ctx context.Context) error {
	userCollection := mongoClient.Database("FunRepDB").Collection("Users")

	// Initialize finalDict with 0-37 keys and 0 values
	finalDict := make(map[int]float64)
	for i := 0; i <= 37; i++ {
		finalDict[i] = 0
	}

	// Fetch users with isBetLocked == true and winningsRoulette == 0
	filter := bson.M{"isBetLocked": true, "winningsRoulette": float64(0)}
	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		log.Println("❌ Error fetching users:", err)
		return err
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
			if err == nil && num >= 0 && num <= 37 {
				finalDict[num] += v
			}
		}
	}
	if err := cursor.Err(); err != nil {
		log.Println("❌ Cursor error:", err)
		return err
	}

	// Fetch algorithm settings from "Records"
	recordCollection := mongoClient.Database("FunRepDB").Collection("Records")
	var algoConfig struct {
		AlgoChoice             string  `bson:"Roulette_Algo"`
		WinProbabilityRoulette float64 `bson:"Roulette_Winning_Probability"`
	}
	objectID, _ := primitive.ObjectIDFromHex("6826546053fd676dfbbb32bc")
	err = recordCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&algoConfig)
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
			winningNumber, err = rouletteWinningAlgo(finalDict, mongoClient, ctx)
		} else if randFloat < winProb+10 {
			winningNumber, err = rouletteZeroAlgo(finalDict)
		} else {
			winningNumber, err = rouletteLosingAlgo(finalDict)
		}
	case "l":
		winningNumber, err = rouletteLosingAlgo(finalDict)
	case "z":
		winningNumber, err = rouletteZeroAlgo(finalDict)
	case "w":
		winningNumber, err = rouletteWinningAlgo(finalDict, mongoClient, ctx)
	default:
		if randFloat < winProb {
			winningNumber, err = rouletteWinningAlgo(finalDict, mongoClient, ctx)
		} else if randFloat < winProb+10 {
			winningNumber, err = rouletteZeroAlgo(finalDict)
		} else {
			winningNumber, err = rouletteLosingAlgo(finalDict)
		}
	}
	if err != nil {
		log.Println("❌ Error generating winning number:", err)
		return err
	}
	currentNextWinningNumber = winningNumber

	// Update nextWinningNumber in RealtimeData
	realtimeCollection := mongoClient.Database("FunRepDB").Collection("RealtimeData")
	realtimeID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	update := bson.M{"$set": bson.M{"nextWinningNumber": currentNextWinningNumber}}

	result, err := realtimeCollection.UpdateOne(ctx, bson.M{"_id": realtimeID}, update)
	if err != nil {
		log.Println("❌ Error updating nextWinningNumber:", err)
		return err
	}
	log.Printf("✅ Update result: matched=%d, modified=%d, newWinningNumber=%d", result.MatchedCount, result.ModifiedCount, currentNextWinningNumber)

	// Update TempWinnings1 for users
	cursor, err = userCollection.Find(ctx, filter)
	if err != nil {
		log.Println("❌ Error finding users for TempWinnings1 update:", err)
		return err
	}
	defer cursor.Close(ctx)

	found := false
	for cursor.Next(ctx) {
		found = true
		var user struct {
			ID   string             `bson:"Id"`
			Bets map[string]float64 `bson:"bets"`
		}
		if err := cursor.Decode(&user); err != nil {
			log.Println("❌ Error decoding user:", err)
			continue
		}

		key := strconv.Itoa(currentNextWinningNumber)
		betAmount, ok := user.Bets[key]
		if ok {
			tempWinnings := math.Round(betAmount*36*100) / 100
			update := bson.M{"$set": bson.M{"TempWinnings1": tempWinnings}}
			_, err := userCollection.UpdateOne(ctx, bson.M{"Id": user.ID}, update)
			if err != nil {
				log.Printf("❌ Failed to update TempWinnings1 for user %s: %v", user.ID, err)
			} else {
				log.Printf("💰 Updated TempWinnings1 for user %s = %f", user.ID, tempWinnings)
			}
		}
	}
	if !found {
		log.Println("⚠️ No users matched the filter for winnings update")
	}
	if err := cursor.Err(); err != nil {
		log.Println("❌ Cursor error on winnings update:", err)
	}

	// Record PnL
	totalAmount := 0.0
	for _, v := range finalDict {
		totalAmount += v
	}
	spentAmount := finalDict[currentNextWinningNumber] * 36
	pnl := math.Floor(totalAmount - spentAmount)

	nowIST := time.Now().In(time.FixedZone("IST", 5.5*3600))
	docID := nowIST.Format("2006-01-02")

	pnlCollection := mongoClient.Database("FunRepDB").Collection("PNL")
	filter = bson.M{"_id": docID}
	update = bson.M{
		"$inc": bson.M{
			"RoulettePnL": pnl,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err = pnlCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Println("❌ Error updating PnL record:", err)
	}

	return nil
}

func UpdateNextMultiplierAndWinningFT(ctx context.Context) error {
	userCollection := mongoClient.Database("FunRepDB").Collection("Users")
	recordCollection := mongoClient.Database("FunRepDB").Collection("Records")
	realtimeCollection := mongoClient.Database("FunRepDB").Collection("RealtimeData")

	// Step 1: Fetch Algo & Probabilities
	var config struct {
		AlgoChoice string  `bson:"FunTarget_Algo"`
		WinProb    float64 `bson:"FunTarget_Winning_Probability"`
		CapProb    float64 `bson:"Cap_Probability"`
	}

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
		return err
	}
	defer cursor.Close(ctx)

	finalDict := make(map[int]float64)
	for i := 0; i < 10; i++ {
		finalDict[i] = 0
	}

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
			log.Printf("❌ Decode user failed: %v", err)
			continue
		}
		users = append(users, u)

		for key, val := range u.BetsFT {
			num, _ := strconv.Atoi(key)
			finalDict[num] += val
		}
	}

	// Step 3: Run algorithm
	var winningFT, multiplier int
	switch config.AlgoChoice {
	case "i":
		winningFT, multiplier = inverseWeightedAlgo(finalDict, config.WinProb, config.CapProb)
	default:
		winningFT, multiplier = rand.Intn(10), 9
	}

	currentNextWinningNumberFT = winningFT
	currentNextMultiplier = multiplier

	// Step 4: Save FT values
	objID, _ := primitive.ObjectIDFromHex("682106fe8bd0bfa24147c16a")
	_, err = realtimeCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
		"$set": bson.M{
			"nextWinningNumberFT": currentNextWinningNumberFT,
			"nextMultiplier":      currentNextMultiplier,
		},
	})
	if err != nil {
		log.Printf("❌ Failed to update FT values: %v", err)
		return err
	}

	// Step 5: Calculate and save user winnings
	for _, u := range users {
		winKey := strconv.Itoa(currentNextWinningNumberFT)
		winAmount := float64(0)
		if val, ok := u.BetsFT[winKey]; ok {
			if currentNextMultiplier == 18 {
				winAmount = math.Round(val*18*100) / 100
			} else {
				winAmount = math.Round(val*9*100) / 100
			}
		}
		_, err := userCollection.UpdateByID(ctx, u.ID, bson.M{"$set": bson.M{"TempWinnings2": winAmount}})
		if err != nil {
			log.Printf("❌ Failed to update winnings for user %v: %v", u.ID.Hex(), err)
		}
	}

	totalAmount := 0.0
	for _, v := range finalDict {
		totalAmount += v
	}
	spentAmount := finalDict[currentNextWinningNumberFT] * float64(currentNextMultiplier)
	pnl := math.Floor(totalAmount - spentAmount)

	nowIST := time.Now().In(time.FixedZone("IST", 5.5*3600))
	docID := nowIST.Format("2006-01-02")

	pnlCollection := mongoClient.Database("FunRepDB").Collection("PNL")
	filter := bson.M{"_id": docID}
	update := bson.M{
		"$inc": bson.M{
			"FunTargetPnL": pnl,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err = pnlCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Println("❌ Error updating PnL record:", err)
	}

	log.Printf("✅ Processed winningFT = %d, multiplier = %d", currentNextWinningNumberFT, currentNextMultiplier)
	return nil

}

func inverseWeightedAlgo(finalDict map[int]float64, winProb float64, capProb float64) (int, int) {
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

func TransferWinnings(ctx context.Context) error {
	userCollection := mongoClient.Database("FunRepDB").Collection("Users")

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
			TempWinnings1 float64            `bson:"TempWinnings1"`
		}

		if err := cursor.Decode(&user); err != nil {
			log.Printf("❌ Failed to decode user: %v", err)
			continue
		}

		var update bson.M
		if user.TempWinnings1 > 0 {
			update = bson.M{
				"$set": bson.M{
					"winningsRoulette":      user.TempWinnings1,
					"TempWinnings1":         0.0,
					"winningNumberRoulette": currentNextWinningNumber,
				},
				"$unset": bson.M{
					"bets": "",
				},
			}
		} else {
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

		_, err := userCollection.UpdateByID(ctx, user.ID, update)
		if err != nil {
			log.Printf("❌ Failed to update user %v: %v", user.ID.Hex(), err)
			continue
		}
		updatedCount++
	}

	log.Printf("✅ Finalized winnings for %d users", updatedCount)
	return nil
}

func TransferWinningsFT(ctx context.Context) error {
	userCollection := mongoClient.Database("FunRepDB").Collection("Users")

	filter := bson.M{"isBetLockedFunTarget": true, "winningsFunTarget": float64(0)}
	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		log.Printf("❌ Error fetching FunTarget locked users: %v", err)
		return err
	}
	defer cursor.Close(ctx)

	var updatedCount int
	for cursor.Next(ctx) {
		var user struct {
			ID            primitive.ObjectID `bson:"_id"`
			TempWinnings2 float64            `bson:"TempWinnings2"`
		}

		if err := cursor.Decode(&user); err != nil {
			log.Printf("❌ Failed to decode user: %v", err)
			continue
		}

		var update bson.M
		if user.TempWinnings2 > 0 {
			update = bson.M{
				"$set": bson.M{
					"winningsFunTarget":      user.TempWinnings2,
					"TempWinnings2":          0.0,
					"winningNumberFunTarget": currentNextWinningNumberFT,
				},
			}
		} else {
			update = bson.M{
				"$set": bson.M{
					"isBetLockedFunTarget": false,
				},
				"$unset": bson.M{
					"betsFT": "",
				},
			}
		}

		_, err := userCollection.UpdateByID(ctx, user.ID, update)
		if err != nil {
			log.Printf("❌ Failed to update user %v: %v", user.ID.Hex(), err)
			continue
		}
		updatedCount++
	}

	log.Printf("✅ Finalized FT winnings for %d users", updatedCount)
	return nil
}

func GetCountdownValues(c *gin.Context) {
	currentTime := time.Now().Unix()
	countdown := Countdown{
		Countdown1: int(nextRoundTime - currentTime),
		Countdown2: int(nextRoundTime2 - currentTime),
	}

	c.JSON(200, countdown)
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI("mongodb+srv://royalplay209:IbgiZSjLJ85QT4Y2@fungamecluster.eioe1np.mongodb.net/?retryWrites=true&w=majority&appName=FunGameCluster")
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Mongo connection error:", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Fatal("Mongo disconnect error:", err)
		}
	}()

	mongoClient = client
	db := client.Database("FunRepDB")
	router := gin.Default()
	// roulette := db.Collection("RealtimeData")

	routes.InitRoutes(router, db)
	router.GET("/api/countdown", GetCountdownValues)
	go StartCountdowns()
	// ws.InitWebSocket(router, roulette)

	// Optional HTTP to HTTPS redirect (port 8080)
	go func() {
		err := http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}))
		if err != nil {
			log.Fatal("HTTP redirect server failed:", err)
		}
	}()

	// go func() {
	// 	err := router.Run(":3000") // Localhost-only plain HTTP access
	// 	if err != nil {
	// 		log.Fatal("Localhost HTTP server failed:", err)
	// 	}
	// }()

	// HTTPS server using cert.pem and funrep.key
	err = router.RunTLS(":443", "cert.pem", "funrep.key")
	if err != nil {
		log.Fatal("HTTPS server failed:", err)
	}
}
