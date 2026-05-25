package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"FunRepBackend/config"
	"FunRepBackend/routes"
	"FunRepBackend/services/games"
	"FunRepBackend/services/rounds"
	"FunRepBackend/ws"

	"github.com/gin-gonic/gin"
)

var (
	roundManager  *rounds.RoundManager
	rouletteSvc   *games.RouletteService
	funTargetSvc  *games.FunTargetService
	tripleFunSvc  *games.TripleFunService
	andarBaharSvc *games.AndarBaharService
)

// RegisterGameLogic registers game logic callbacks that will be triggered based on round duration
// offset is the time offset in seconds to stagger this game's timer from others
func RegisterGameLogic(game string, column string, roundDuration int, offset int) {
	if roundManager == nil {
		log.Fatal("RoundManager not initialized")
	}

	// Define update and end handlers based on game
	var onUpdate, onEnd func()

	switch game {
	case "roulette":
		onUpdate = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := rouletteSvc.UpdateNextWinningNumber(ctx); err != nil {
				log.Printf("❌ Error updating roulette winning number: %v", err)
			}
		}
		onEnd = func() {
			ctx := context.Background()
			if err := rouletteSvc.FinalizeWinningNumber(ctx); err != nil {
				log.Printf("❌ Error finalizing roulette: %v", err)
			}
			if err := rouletteSvc.TransferWinnings(ctx); err != nil {
				log.Printf("❌ Error transferring roulette winnings: %v", err)
			}
			winningNumber := games.State.GetCurrentWinningNumber("roulette")
			if winningNumber != nil {
				// Current round end timestamp (current time)
				endTsMs := time.Now().Unix() * 1000
				// Next round end timestamp (current time + round duration)
				nextRoundEndTsMs := (time.Now().Unix() + int64(roundDuration)) * 1000
				roundManager.FinalizeRoundHistory(column, endTsMs, winningNumber, nextRoundEndTsMs)
			}
		}
	case "funtarget":
		onUpdate = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := funTargetSvc.UpdateNextWinningNumber(ctx); err != nil {
				log.Printf("❌ Error updating FunTarget winning number: %v", err)
			}
		}
		onEnd = func() {
			ctx := context.Background()
			if err := funTargetSvc.FinalizeWinningNumber(ctx); err != nil {
				log.Printf("❌ Error finalizing FunTarget: %v", err)
			}
			if err := funTargetSvc.TransferWinnings(ctx); err != nil {
				log.Printf("❌ Error transferring FunTarget winnings: %v", err)
			}
			winningNumber := games.State.GetCurrentWinningNumber("funtarget")
			if winningNumber != nil {
				// Current round end timestamp (current time)
				endTsMs := time.Now().Unix() * 1000
				// Next round end timestamp (current time + round duration)
				nextRoundEndTsMs := (time.Now().Unix() + int64(roundDuration)) * 1000
				roundManager.FinalizeRoundHistory(column, endTsMs, winningNumber, nextRoundEndTsMs)
			}
		}
	case "triplefun":
		onUpdate = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tripleFunSvc.UpdateNextWinningNumber(ctx); err != nil {
				log.Printf("❌ Error updating TripleFun winning number: %v", err)
			}
		}
		onEnd = func() {
			ctx := context.Background()
			if err := tripleFunSvc.FinalizeWinningNumber(ctx); err != nil {
				log.Printf("❌ Error finalizing TripleFun: %v", err)
			}
			if err := tripleFunSvc.TransferWinnings(ctx); err != nil {
				log.Printf("❌ Error transferring TripleFun winnings: %v", err)
			}
			winningNumber := games.State.GetCurrentWinningNumber("triplefun")
			if winningNumber != nil {
				// Current round end timestamp (current time)
				endTsMs := time.Now().Unix() * 1000
				// Next round end timestamp (current time + round duration)
				nextRoundEndTsMs := (time.Now().Unix() + int64(roundDuration)) * 1000
				roundManager.FinalizeRoundHistory(column, endTsMs, winningNumber, nextRoundEndTsMs)
			}
		}
	case "andarbahar":
		onUpdate = func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := andarBaharSvc.UpdateNextWinningNumber(ctx); err != nil {
				log.Printf("❌ Error updating AndarBahar winning number: %v", err)
			}
		}
		onEnd = func() {
			ctx := context.Background()
			if err := andarBaharSvc.FinalizeWinningNumber(ctx); err != nil {
				log.Printf("❌ Error finalizing AndarBahar: %v", err)
			}
			if err := andarBaharSvc.TransferWinnings(ctx); err != nil {
				log.Printf("❌ Error transferring AndarBahar winnings: %v", err)
			}
			winningNumber := games.State.GetCurrentWinningNumber("andarbahar")
			if winningNumber != nil {
				// Current round end timestamp (current time)
				endTsMs := time.Now().Unix() * 1000
				// Next round end timestamp (current time + round duration)
				nextRoundEndTsMs := (time.Now().Unix() + int64(roundDuration)) * 1000
				roundManager.FinalizeRoundHistory(column, endTsMs, winningNumber, nextRoundEndTsMs)
			}
		}
	}

	roundManager.RegisterGameCallbacks(game, column, roundDuration, offset, onUpdate, onEnd)
}

func main() {
	// Initialize configuration
	config.LoadEnv()
	config.InitRedis()
	mongoClient := config.InitMongo()
	defer func() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Fatal("Mongo disconnect error:", err)
		}
	}()

	db := mongoClient.Database("FunRepDB")

	// Initialize services
	roundManager = rounds.NewRoundManager(db)
	rouletteSvc = games.NewRouletteService(mongoClient)
	funTargetSvc = games.NewFunTargetService(mongoClient)
	tripleFunSvc = games.NewTripleFunService(mongoClient)
	andarBaharSvc = games.NewAndarBaharService(mongoClient)

	// Setup router
	router := gin.Default()
	router.Use(config.SetupCORS())
	routes.InitRoutes(router, db, config.JWTSecret)

	// Initialize WebSocket
	realtimeCollection := db.Collection("RealtimeData")
	ws.InitWebSocket(router, realtimeCollection, nil, nil)

	// Register game logic callbacks (triggered based on round duration)
	// Round durations: Roulette=60s, FunTarget=60s, TripleFun=120s, AndarBahar=45s
	// Offsets: Each game has a 15-second offset to stagger timers (roulette=0, funtarget=15, triplefun=30, andarbahar=45)
	RegisterGameLogic("roulette", "roundsData1", 60, 0)
	RegisterGameLogic("funtarget", "roundsData2", 60, 15)
	RegisterGameLogic("triplefun", "roundsData3", 120, 30)
	RegisterGameLogic("andarbahar", "roundsData4", 45, 45)

	// HTTP to HTTPS redirect (port 8080)
	go func() {
		err := http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}))
		if err != nil {
			log.Fatal("HTTP redirect server failed:", err)
		}
	}()

	// Localhost HTTP server (port 3000)
	go func() {
		err := router.Run(":3000")
		if err != nil {
			log.Fatal("Localhost HTTP server failed:", err)
		}
	}()

	// HTTPS server (port 443)
	err := router.RunTLS(":443", "cert.pem", "funrep.key")
	if err != nil {
		log.Fatal("HTTPS server failed:", err)
	}
}
