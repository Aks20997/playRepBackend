package pnl

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	UserCollection      *mongo.Collection
	DailyPnlCollection  *mongo.Collection
)

func GetDailyPnL(c *gin.Context) {
	_, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	loc, _ := time.LoadLocation("Asia/Kolkata")
	today := time.Now().In(loc).Format("2006-01-02")

	var dailyDoc bson.M
	err := DailyPnlCollection.FindOne(ctx, bson.M{"date": today}).Decode(&dailyDoc)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "DailyPnL not found for today"})
		return
	}

	// Extract user-specific PnL if available
	if userPnL, ok := dailyDoc["masters"].(bson.M); ok {
		c.JSON(http.StatusOK, gin.H{"data": userPnL})
	} else {
		c.JSON(http.StatusOK, gin.H{"data": dailyDoc})
	}
}

func StartDailyPnLTracker() {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	c := cron.New(cron.WithLocation(loc))

	_, err := c.AddFunc("0 0 * * *", func() {
		initializeDailyPnL()
	})
	if err != nil {
		panic("Failed to schedule daily PnL initialization")
	}

	_, err = c.AddFunc("*/10 * * * *", func() {
		updateDailyPnL()
	})
	if err != nil {
		panic("Failed to schedule 10-min PnL updates")
	}

	c.Start()
}

func initializeDailyPnL() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loc, _ := time.LoadLocation("Asia/Kolkata")
	today := time.Now().In(loc).Format("2006-01-02")

	count, _ := DailyPnlCollection.CountDocuments(ctx, bson.M{"date": today})
	if count > 0 {
		return
	}

	cursor, err := UserCollection.Find(ctx, bson.M{})
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	baselines := make(map[string]float64)
	for cursor.Next(ctx) {
		var user struct {
			UserId string  `bson:"Id"`
			Points float64 `bson:"points"`
		}
		if err := cursor.Decode(&user); err == nil {
			baselines[user.UserId] = user.Points
		}
	}

	doc := bson.M{
		"date":      today,
		"createdAt": time.Now().In(loc),
		"baselines": baselines,
		"updatedAt": time.Now().In(loc),
	}

	_, _ = DailyPnlCollection.InsertOne(ctx, doc)
}

func updateDailyPnL() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	loc, _ := time.LoadLocation("Asia/Kolkata")
	today := time.Now().In(loc).Format("2006-01-02")

	var dailyDoc bson.M
	err := DailyPnlCollection.FindOne(ctx, bson.M{"date": today}).Decode(&dailyDoc)
	if err != nil {
		return
	}

	baselines, ok := dailyDoc["baselines"].(bson.M)
	if !ok {
		return
	}

	masterCursor, err := UserCollection.Find(ctx, bson.M{"Type": "Master"})
	if err != nil {
		return
	}
	defer masterCursor.Close(ctx)

	mastersData := bson.M{}

	for masterCursor.Next(ctx) {
		var master struct {
			UserId string   `bson:"Id"`
			Childs []string `bson:"Childs"`
		}
		if err := masterCursor.Decode(&master); err != nil {
			continue
		}

		masterData := bson.M{}
		masterTotalPnL := float64(0)

		for _, dealerId := range master.Childs {
			var dealer struct {
				UserId string   `bson:"Id"`
				Childs []string `bson:"Childs"`
			}
			err := UserCollection.FindOne(ctx, bson.M{"Id": dealerId}).Decode(&dealer)
			if err != nil {
				continue
			}

			dealerTotalPnL := float64(0)
			childData := bson.M{}

			for _, childId := range dealer.Childs {
				var child struct {
					UserId string  `bson:"Id"`
					Points float64 `bson:"points"`
				}
				err := UserCollection.FindOne(ctx, bson.M{"Id": childId}).Decode(&child)
				if err != nil {
					continue
				}

				prevPts := 0.0
				if val, ok := baselines[child.UserId].(float64); ok {
					prevPts = val
				}

				childPnL := child.Points - prevPts
				childData[child.UserId] = childPnL
				dealerTotalPnL += childPnL
			}

			masterData[dealerId] = bson.M{
				"dealerPnL": dealerTotalPnL,
				"children":  childData,
			}
			masterTotalPnL += dealerTotalPnL
		}

		mastersData[master.UserId] = bson.M{
			"masterPnL": masterTotalPnL,
			"dealers":   masterData,
		}
	}

	_, _ = DailyPnlCollection.UpdateOne(ctx, bson.M{"date": today}, bson.M{
		"$set": bson.M{
			"masters":  mastersData,
			"updatedAt": time.Now().In(loc),
		},
	})
}

