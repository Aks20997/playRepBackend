package routes

import (
	"FunRepBackend/routes/admin"
	"FunRepBackend/routes/auth"
	"FunRepBackend/routes/bets"
	"FunRepBackend/routes/gamedata"
	"FunRepBackend/routes/location"
	"FunRepBackend/routes/pnl"
	"FunRepBackend/routes/points"
	"FunRepBackend/routes/user"
	"FunRepBackend/routes/winnings"

	"FunRepBackend/config"

	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func InitRoutes(router *gin.Engine, db *mongo.Database, jwtSecret string) {
	// Use Redis client from config
	redisClient := config.RedisClient

	// Initialize collections
	userCollection := db.Collection("Users")
	realtimeDataCollection := db.Collection("RealtimeData")
	roundsDataCollection := db.Collection("RoundsData")
	dailyPnlCollection := db.Collection("DailyPnL")
	commissionCollection := db.Collection("commission")
	pointsCollection := db.Collection("points")

	// Initialize auth package
	auth.JWTSecret = jwtSecret
	auth.RedisClient = redisClient
	auth.UserCollection = userCollection

	// Initialize user package
	user.UserCollection = userCollection
	user.RealtimeDataCollection = realtimeDataCollection

	// Initialize bets package
	bets.UserCollection = userCollection

	// Initialize winnings package
	winnings.UserCollection = userCollection
	winnings.RedisClient = redisClient

	// Initialize gamedata package
	gamedata.RealtimeDataCollection = realtimeDataCollection
	gamedata.RoundsDataCollection = roundsDataCollection

	// Initialize points package
	points.UserCollection = userCollection
	points.PointsCollection = pointsCollection
	points.CommissionCollection = commissionCollection

	// Initialize admin package
	admin.UserCollection = userCollection

	// Initialize location package
	location.UserCollection = userCollection

	// Initialize pnl package
	pnl.UserCollection = userCollection
	pnl.DailyPnlCollection = dailyPnlCollection

	// Start background services
	go pnl.StartDailyPnLTracker()
	go auth.StartSessionCleanup()
	go StartDailyRoundsDataCleanup(roundsDataCollection)

	// Setup routes
	userRoutes := router.Group("/api")
	{
		// Auth routes
		userRoutes.POST("/login", auth.LoginUser)
		userRoutes.POST("/logout", auth.JWTAuthMiddleware(jwtSecret), auth.LogoutUser)

		// User routes
		userRoutes.GET("/user/:id", user.GetUserById)
		userRoutes.GET("/getPoints", auth.JWTAuthMiddleware(jwtSecret), user.GetUserPoints)
		userRoutes.GET("/getUserChilds", auth.JWTAuthMiddleware(jwtSecret), user.GetUserChilds)
		userRoutes.GET("/getVersionInfo", user.GetVersionInfo)

		// Bet routes
		userRoutes.POST("/rouletteBets", auth.JWTAuthMiddleware(jwtSecret), bets.UpdateBets)
		userRoutes.POST("/funTargetBets", auth.JWTAuthMiddleware(jwtSecret), bets.UpdateBetsFT)
		userRoutes.POST("/tripleFunBets", auth.JWTAuthMiddleware(jwtSecret), bets.UpdateBetsTripleFun)
		userRoutes.POST("/andarBaharBets", auth.JWTAuthMiddleware(jwtSecret), bets.UpdateBetsAndarBahar)
		userRoutes.GET("/getBetState", auth.JWTAuthMiddleware(jwtSecret), bets.GetBetState)
		userRoutes.GET("/getBetsFT", auth.JWTAuthMiddleware(jwtSecret), bets.GetBetsFT)
		userRoutes.GET("/getBetsTF", auth.JWTAuthMiddleware(jwtSecret), bets.GetBetStateTF)
		userRoutes.GET("/getBetsAB", auth.JWTAuthMiddleware(jwtSecret), bets.GetBetStateAB)

		// Winnings routes
		userRoutes.GET("/claimRouletteWinnings", auth.JWTAuthMiddleware(jwtSecret), winnings.ClaimRouletteWinnings)
		userRoutes.GET("/claimFtWinnings", auth.JWTAuthMiddleware(jwtSecret), winnings.ClaimFTWinnings)
		userRoutes.GET("/claimTfWinnings", auth.JWTAuthMiddleware(jwtSecret), winnings.ClaimTripleFunWinnings)
		userRoutes.GET("/claimABWinnings", auth.JWTAuthMiddleware(jwtSecret), winnings.ClaimAndarBaharWinnings)
		userRoutes.GET("/getRouletteWinnings", auth.JWTAuthMiddleware(jwtSecret), winnings.GetRouletteWinnings)
		userRoutes.GET("/getFTWinnings", auth.JWTAuthMiddleware(jwtSecret), winnings.GetFunTargetWinnings)
		userRoutes.GET("/getWinningsTripleFun", auth.JWTAuthMiddleware(jwtSecret), winnings.GetTripleFunWinnings)
		userRoutes.GET("/getWinningsAB", auth.JWTAuthMiddleware(jwtSecret), winnings.GetAndarBaharWinnings)

		// Game data routes
		userRoutes.GET("/getRouletteData", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetRouletteData)
		userRoutes.GET("/getFtData", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetFtData)
		userRoutes.GET("/getTfData", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetTfData)
		userRoutes.GET("/getABData", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetAbData)
		userRoutes.GET("/getRouletteHistory", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetRouletteHistory)
		userRoutes.GET("/getFtHistory", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetFtHistory)
		userRoutes.GET("/getTfHistory", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetTripleFunHistory)
		userRoutes.GET("/getAbHistory", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetAndarBaharHistory)
		userRoutes.GET("/getDrawDetails", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetDrawDetails)

		// Round state routes (with startTs and endTs)
		userRoutes.GET("/getCurrentRounds", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetCurrentRounds)
		userRoutes.GET("/getCurrentRound/:game", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetCurrentRound)

		// Stateless timer route - returns server Unix time (single source of truth)
		userRoutes.GET("/getServerTime", auth.JWTAuthMiddleware(jwtSecret), gamedata.GetServerTime)

		// Points routes
		userRoutes.POST("/sendPoints", auth.JWTAuthMiddleware(jwtSecret), points.CreatePointRequest)
		userRoutes.POST("/receivePoints", auth.JWTAuthMiddleware(jwtSecret), points.ReceivePointRequest)
		userRoutes.POST("/rejectPoints", auth.JWTAuthMiddleware(jwtSecret), points.RejectPointRequest)
		userRoutes.GET("/pointRequests", auth.JWTAuthMiddleware(jwtSecret), points.GetPointRequests)

		// Admin routes
		userRoutes.POST("/resetPassword", auth.JWTAuthMiddleware(jwtSecret), admin.ResetPasswordByID)
		userRoutes.POST("/changePassword", auth.JWTAuthMiddleware(jwtSecret), admin.ChangePassword)
		userRoutes.POST("/changePin", auth.JWTAuthMiddleware(jwtSecret), admin.ChangePin)
		userRoutes.POST("/resetPin", auth.JWTAuthMiddleware(jwtSecret), admin.ResetPin)
		userRoutes.POST("/admin-db", admin.AdminDBHandler)

		// Location routes
		userRoutes.POST("/saveLocation", auth.JWTAuthMiddleware(jwtSecret), location.SaveUserLocation)
		userRoutes.GET("/getAllowedLocation", auth.JWTAuthMiddleware(jwtSecret), location.GetAllowedLocation)

		// PnL routes
		userRoutes.GET("/getDailyPnL", auth.JWTAuthMiddleware(jwtSecret), pnl.GetDailyPnL)
	}
}

// StartDailyRoundsDataCleanup schedules a daily job to clear roundsData1-4 fields at midnight IST
func StartDailyRoundsDataCleanup(roundsDataCollection *mongo.Collection) {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Printf("❌ Failed to load Asia/Kolkata timezone: %v", err)
		loc = time.UTC
	}

	c := cron.New(cron.WithLocation(loc))

	_, err = c.AddFunc("0 0 * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		objectID, err := primitive.ObjectIDFromHex("69314e8483520b7bf43cc484")
		if err != nil {
			log.Printf("❌ Invalid ObjectID: %v", err)
			return
		}

		filter := bson.M{"_id": objectID}
		update := bson.M{
			"$unset": bson.M{
				"roundsData1": "",
				"roundsData2": "",
				"roundsData3": "",
				"roundsData4": "",
			},
		}

		result, err := roundsDataCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			log.Printf("Failed to clear roundsData fields: %v", err)
		} else {
			log.Printf("Successfully cleared roundsData1-4 at 12:00 AM IST (matched: %d, modified: %d)",
				result.MatchedCount, result.ModifiedCount)
		}
	})

	if err != nil {
		log.Printf("Failed to schedule daily roundsData cleanup: %v", err)
		return
	}

	c.Start()
	log.Println("Daily roundsData cleanup scheduler started - will clear roundsData1-4 at 12:00 AM IST daily")
}
