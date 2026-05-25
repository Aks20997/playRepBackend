package config

import (
	"context"
	"log"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	RoundsDataID = "69314e8483520b7bf43cc484"
)

func GetRoundsDataColumn(ctx context.Context, db *mongo.Database, roundsDataIDHex, column string) (map[string]interface{}, error) {
	collection := db.Collection("RoundsData")

	objectID, err := primitive.ObjectIDFromHex(roundsDataIDHex)
	if err != nil {
		return nil, err
	}

	var result bson.M
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&result)
	if err != nil {
		return nil, err
	}

	data, exists := result[column].(bson.M)
	if !exists {
		return map[string]interface{}{}, nil
	}

	// Convert bson.M to map[string]interface{}
	resultMap := make(map[string]interface{})
	for k, v := range data {
		resultMap[k] = v
	}

	return resultMap, nil
}

func UpdateRoundsData(ctx context.Context, db *mongo.Database, roundsDataIDHex, column string, data map[string]interface{}) error {
	collection := db.Collection("RoundsData")

	objectID, err := primitive.ObjectIDFromHex(roundsDataIDHex)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			column: data,
		},
	}

	_, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		log.Printf("❌ Failed to update %s: %v", column, err)
		return err
	}

	return nil
}

func MarkRoundOpen(ctx context.Context, db *mongo.Database, column string, endTs int64) error {
	endTsSec := endTs
	if endTsSec > 1e12 {
		endTsSec = endTsSec / 1000
	}

	data, err := GetRoundsDataColumn(ctx, db, RoundsDataID, column)
	if err != nil {
		return err
	}

	if data == nil {
		data = map[string]interface{}{}
	}

	for key, value := range data {
		if strValue, ok := value.(string); ok && strValue == "NOT OPEN" {
			delete(data, key)
			log.Printf("🗑️ Removed existing NOT OPEN entry with key: %s", key)
		}
	}

	data[strconv.FormatInt(endTsSec, 10)] = "NOT OPEN"
	return UpdateRoundsData(ctx, db, RoundsDataID, column, data)
}

func FinalizeRoundHistory(ctx context.Context, db *mongo.Database, column string, endTs int64, value interface{}, nextRoundEndTs int64) error {
	endTsSec := endTs
	if endTsSec > 1e12 {
		endTsSec = endTsSec / 1000
	}

	data, err := GetRoundsDataColumn(ctx, db, RoundsDataID, column)
	if err != nil {
		return err
	}

	if data == nil {
		data = map[string]interface{}{}
	}

	found := false
	for key, val := range data {
		if strValue, ok := val.(string); ok && strValue == "NOT OPEN" {
			data[key] = value
			found = true
			log.Printf("✅ Updated NOT OPEN entry (key: %s) with value: %v", key, value)
			break
		}
	}

	if !found {
		data[strconv.FormatInt(endTsSec, 10)] = value
		log.Printf("⚠️ No NOT OPEN entry found, created new entry (key: %d) with value: %v", endTsSec, value)
	}

	if nextRoundEndTs > 0 {
		nextRoundEndTsSec := nextRoundEndTs
		if nextRoundEndTsSec > 1e12 {
			nextRoundEndTsSec = nextRoundEndTsSec / 1000
		}

		for key, val := range data {
			if strValue, ok := val.(string); ok && strValue == "NOT OPEN" {
				delete(data, key)
				log.Printf("🗑️ Removed remaining NOT OPEN entry with key: %s", key)
			}
		}

		data[strconv.FormatInt(nextRoundEndTsSec, 10)] = "NOT OPEN"
		log.Printf("✅ Created new NOT OPEN entry for next round (key: %d)", nextRoundEndTsSec)
	}

	return UpdateRoundsData(ctx, db, RoundsDataID, column, data)
}
