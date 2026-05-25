package ws

import (
	"FunRepBackend/config"
	"FunRepBackend/models"
	"FunRepBackend/session"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	Conn   *websocket.Conn
	Mu     sync.Mutex
	UserID string
	Token  string
}

type GameDataPayload struct {
	Type      string `json:"type"`
	Roulette        GameNumbers       `json:"roulette"`
	RouletteHistory []int             `json:"rouletteHistory"`
	FunTarget       FunTargetNumbers  `json:"funTarget"`
	FunTargetHistory []int            `json:"funTargetHistory"`
	TripleFun       TripleFunNumbers  `json:"tripleFun"`
	TripleFunHistory []string         `json:"tripleFunHistory"`
	AndarBahar      AndarBaharNumbers `json:"andarBahar"`
	ABHistory       []int             `json:"abHistory"`
}

type GameNumbers struct {
	Winning int `json:"winning"`
	Next    int `json:"next"`
}

type FunTargetNumbers struct {
	Winning        int `json:"winning"`
	Next           int `json:"next"`
	Multiplier     int `json:"multiplier"`
	NextMultiplier int `json:"nextMultiplier"`
}

type TripleFunNumbers struct {
	Winning string `json:"winning"`
	Next    string `json:"next"`
}

type AndarBaharNumbers struct {
	Winning   int   `json:"winning"`
	Next      int   `json:"next"`
	NextArray []int `json:"nextArray"`
}

const realtimeHexID = "682106fe8bd0bfa24147c16a"

var (
	userClients       = make(map[string]*Client) // userId -> Client
	validTokens       = make(map[string]string)  // userId -> latest valid token
	upgrader          = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	mutex             sync.Mutex
	realtimeObjectID  primitive.ObjectID
	startWatcherOnce  sync.Once
)

func init() {
	var err error
	realtimeObjectID, err = primitive.ObjectIDFromHex(realtimeHexID)
	if err != nil {
		log.Fatalf("invalid realtime data object id: %v", err)
	}
}

func RegisterValidToken(userId, token string) {
	mutex.Lock()
	defer mutex.Unlock()
	validTokens[userId] = token
}

func InitWebSocket(router *gin.Engine, coll *mongo.Collection, getRoundStates func() map[string]models.RoundState, getNextRounds func(string) []models.RoundState) {
	startWatcherOnce.Do(func() {
		go startRealtimeWatcher(coll)
	})

	router.GET("/ws", func(c *gin.Context) {
		handleConnections(c.Writer, c.Request, coll, getRoundStates, getNextRounds)
	})
}

func handleConnections(w http.ResponseWriter, r *http.Request, coll *mongo.Collection, getRoundStates func() map[string]models.RoundState, getNextRounds func(string) []models.RoundState) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("⚠️ recovered in handleConnections: %v", rec)
		}
	}()

	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(config.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["user_id"] == nil {
		http.Error(w, "invalid claims", http.StatusUnauthorized)
		return
	}
	userId := fmt.Sprintf("%v", claims["user_id"])

	// Use session package to validate token
	if !session.IsValidToken(userId, tokenString) {
		log.Printf("Unauthorized connection attempt for userId: %s\n", userId)
		http.Error(w, "unauthorized - outdated session", http.StatusUnauthorized)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("webSocket upgrade error:", err)
		return
	}

	client := &Client{Conn: wsConn, UserID: userId, Token: tokenString}

	mutex.Lock()
	if existing, ok := userClients[userId]; ok {
		existing.Mu.Lock()
		_ = existing.Conn.WriteJSON(map[string]string{"type": "force_logout"})
		_ = existing.Conn.Close()
		existing.Mu.Unlock()
	}
	userClients[userId] = client
	mutex.Unlock()

	sendInitialGameState(client, coll)
	
	// Send server time to newly connected client (stateless timer)
	// Pass nil to use the registered function to get per-game remaining times
	SendServerTime(client, nil)

	log.Println("User connected:", userId)
	go handleClientMessages(client, coll)

	for {
		if _, _, err := client.Conn.ReadMessage(); err != nil {
			break
		}
	}

	mutex.Lock()
	if current, ok := userClients[userId]; ok && current == client {
		delete(userClients, userId)
	}
	mutex.Unlock()

	_ = client.Conn.Close()
	log.Println("User disconnected:", userId)
}

func handleClientMessages(client *Client, coll *mongo.Collection) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("⚠️ recovered in handleClientMessages for user %s: %v", client.UserID, rec)
		}
	}()

	for {
		_, msg, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(msg, &payload); err != nil {
			continue
		}

		if payload["type"] == "location_update" {
			lat, ok1 := payload["lat"].(float64)
			lng, ok2 := payload["lng"].(float64)
			if !ok1 || !ok2 {
				continue
			}

			go func(userId string, lat, lng float64) {
				defer func() {
					if rec := recover(); rec != nil {
						log.Printf("⚠️ recovered in location update for user %s: %v", userId, rec)
					}
				}()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				filter := bson.M{"user_id": userId}
				update := bson.M{
					"$set": bson.M{
						"location": bson.M{
							"lat": lat,
							"lng": lng,
						},
					},
				}
				_, err := coll.UpdateOne(ctx, filter, update)
				if err != nil {
					log.Println("Failed to update location:", err)
				}
			}(client.UserID, lat, lng)
		}
	}
}

func sendInitialGameState(client *Client, coll *mongo.Collection) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload, err := loadGameData(ctx, coll)
	if err != nil {
		log.Printf("failed to load game data for initial push: %v", err)
		return
	}

	sendGameData(client, payload)
}


func loadGameData(ctx context.Context, coll *mongo.Collection) (*GameDataPayload, error) {
	var doc struct {
		WinningNumber        int    `bson:"winningNumber"`
		NextWinningNumber    int    `bson:"nextWinningNumber"`
		WinningNumberFT      int    `bson:"winningNumberFT"`
		NextWinningNumberFT  int    `bson:"nextWinningNumberFT"`
		Multiplier           int    `bson:"multiplier"`
		NextMultiplier       int    `bson:"nextMultiplier"`
		WinningNumberTF      string `bson:"winningNumberTF"`
		NextWinningNumberTF  string `bson:"nextWinningNumberTF"`
		WinningNumberAB      int    `bson:"winningNumberAB"`
		NextWinningNumberAB  int    `bson:"nextWinningNumberAB"`
		NextABArray          []int  `bson:"nextABArray"`
		RouletteHistory      []int    `bson:"rouletteHistory"`
		FunTargetHistory     []int    `bson:"funTargetHistory"`
		TripleFunHistory     []string `bson:"tripleFunHistory"`
		ABHistory            []int    `bson:"ABHistory"`
	}

	if err := coll.FindOne(ctx, bson.M{"_id": realtimeObjectID}).Decode(&doc); err != nil {
		return nil, err
	}

	return &GameDataPayload{
		Type: "game_data",
		Roulette: GameNumbers{
			Winning: doc.WinningNumber,
			Next:    doc.NextWinningNumber,
		},
		RouletteHistory: doc.RouletteHistory,
		FunTarget: FunTargetNumbers{
			Winning:        doc.WinningNumberFT,
			Next:           doc.NextWinningNumberFT,
			Multiplier:     doc.Multiplier,
			NextMultiplier: doc.NextMultiplier,
		},
		FunTargetHistory: doc.FunTargetHistory,
		TripleFun: TripleFunNumbers{
			Winning: doc.WinningNumberTF,
			Next:    doc.NextWinningNumberTF,
		},
		TripleFunHistory: doc.TripleFunHistory,
		AndarBahar: AndarBaharNumbers{
			Winning:   doc.WinningNumberAB,
			Next:      doc.NextWinningNumberAB,
			NextArray: doc.NextABArray,
		},
		ABHistory: doc.ABHistory,
	}, nil
}

func sendGameData(client *Client, payload *GameDataPayload) {
	client.Mu.Lock()
	defer client.Mu.Unlock()
	if err := client.Conn.WriteJSON(payload); err != nil {
		log.Printf("failed to send game data to user %s: %v", client.UserID, err)
	}
}

func broadcastGameData(payload *GameDataPayload) {
	mutex.Lock()
	clients := make([]*Client, 0, len(userClients))
	for _, c := range userClients {
		clients = append(clients, c)
	}
	mutex.Unlock()

	for _, c := range clients {
		sendGameData(c, payload)
	}
}

func startRealtimeWatcher(coll *mongo.Collection) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("⚠️ recovered in startRealtimeWatcher: %v", rec)
		}
	}()

	ctx := context.Background()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "operationType", Value: bson.D{{Key: "$in", Value: bson.A{"insert", "update", "replace"}}}},
			{Key: "documentKey._id", Value: realtimeObjectID},
		}}},
	}

	stream, err := coll.Watch(ctx, pipeline, options.ChangeStream().SetFullDocument(options.UpdateLookup))
	if err != nil {
		log.Printf("failed to start realtime watcher: %v", err)
		return
	}
	defer stream.Close(ctx)

	for stream.Next(ctx) {
		innerCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		payload, err := loadGameData(innerCtx, coll)
		cancel()
		if err != nil {
			log.Printf("failed to refresh game data after change event: %v", err)
			continue
		}
		broadcastGameData(payload)
	}

	if err := stream.Err(); err != nil {
		log.Printf("realtime watcher error: %v", err)
	}
}
