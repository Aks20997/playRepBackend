// package ws

// import (
// 	"context"
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/gorilla/websocket"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.mongodb.org/mongo-driver/mongo/options"
// )

// var (
// 	clients   = make(map[*websocket.Conn]string)
// 	broadcast = make(chan []byte)
// 	upgrader  = websocket.Upgrader{
// 		CheckOrigin: func(r *http.Request) bool { return true },
// 	}
// 	mutex sync.Mutex
// )

// func InitWebSocket(router *gin.Engine, client *mongo.Client) {
// 	go watchChanges(context.Background(), client.Database("FunRepDB").Collection("CommonData"))
// 	go handleBroadcasts()

// 	router.GET("/ws", func(c *gin.Context) {
// 		handleConnections(c.Writer, c.Request, client)
// 	})
// }

// func handleConnections(w http.ResponseWriter, r *http.Request, client *mongo.Client) {
// 	userId := r.URL.Query().Get("userId")
// 	if userId == "" {
// 		http.Error(w, "User ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	ws, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Println("WebSocket upgrade error:", err)
// 		return
// 	}
// 	defer ws.Close()

// 	mutex.Lock()
// 	clients[ws] = userId
// 	mutex.Unlock()

// 	commonColl := client.Database("FunRepDB").Collection("RealtimeData")
// 	userColl := client.Database("FunRepDB").Collection("Users")

// 	go sendCommonData(ws, commonColl)
// 	go sendUserSpecificData(ws, userId, userColl)

// 	for {
// 		_, _, err := ws.ReadMessage()
// 		if err != nil {
// 			mutex.Lock()
// 			delete(clients, ws)
// 			mutex.Unlock()
// 			break
// 		}
// 	}
// }

// func sendCommonData(ws *websocket.Conn, commonColl *mongo.Collection) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	projection := bson.M{
// 		"roundsData1": 0,
// 		"roundsData2": 0,
// 	}

// 	cursor, err := commonColl.Find(ctx, bson.M{}, options.Find().SetProjection(projection))
// 	if err != nil {
// 		log.Println("Initial Find error:", err)
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	for cursor.Next(ctx) {
// 		var doc bson.M
// 		if err := cursor.Decode(&doc); err == nil {
// 			if data, err := json.Marshal(doc); err == nil {
// 				if err := SafeWrite(ws, websocket.TextMessage, data); err != nil {
// 					log.Println("Error writing initial common data:", err)
// 					return
// 				}
// 			}
// 		}
// 	}
// }

// func sendUserSpecificData(ws *websocket.Conn, userId string, userColl *mongo.Collection) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	var userDoc bson.M
// 	err := userColl.FindOne(ctx, bson.M{"Id": userId}).Decode(&userDoc)
// 	if err != nil {
// 		log.Println("Error fetching user data:", err)
// 		return
// 	}

// 	userData := map[string]interface{}{
// 		"userId":      userId,
// 		"isBetLocked": userDoc["isBetLocked"],
// 	}

// 	data, err := json.Marshal(userData)
// 	if err != nil {
// 		log.Println("Error marshalling user data:", err)
// 		return
// 	}

// 	if err := SafeWrite(ws, websocket.TextMessage, data); err != nil {
// 		log.Println("Error sending user-specific data:", err)
// 	}
// }

// func handleBroadcasts() {
// 	for {
// 		msg := <-broadcast
// 		mutex.Lock()
// 		for client := range clients {
// 			if err := SafeWrite(client, websocket.TextMessage, msg); err != nil {
// 				log.Println("Error broadcasting to client:", err)
// 				client.Close()
// 				delete(clients, client)
// 			}
// 		}
// 		mutex.Unlock()
// 	}
// }

// func watchChanges(ctx context.Context, coll *mongo.Collection) {
// 	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
// 	stream, err := coll.Watch(ctx, mongo.Pipeline{}, opts)
// 	if err != nil {
// 		log.Fatal("Mongo watch error:", err)
// 	}
// 	defer stream.Close(ctx)

// 	for stream.Next(ctx) {
// 		var event bson.M
// 		if err := stream.Decode(&event); err == nil {
// 			if fullDoc, ok := event["fullDocument"]; ok {
// 				if data, err := json.Marshal(fullDoc); err == nil {
// 					log.Println("Broadcasting updated data...")
// 					broadcast <- data
// 				}
// 			}
// 		}
// 	}
// }

// func SafeWrite(conn *websocket.Conn, messageType int, data []byte) error {
// 	mutex.Lock()
// 	defer mutex.Unlock()
// 	return conn.WriteMessage(messageType, data)
// }

//=============================================================================================

// package ws

// import (
// 	"context"
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/gorilla/websocket"
// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.mongodb.org/mongo-driver/mongo/options"
// )

// var (
// 	clients   = make(map[*websocket.Conn]string) // Map of WebSocket connections to user IDs
// 	broadcast = make(chan []byte)
// 	upgrader  = websocket.Upgrader{
// 		CheckOrigin: func(r *http.Request) bool { return true },
// 	}
// 	mutex sync.Mutex
// )

// func InitWebSocket(router *gin.Engine, client *mongo.Client) {
// 	go watchChanges(context.Background(), client.Database("FunRepDB").Collection("CommonData"))
// 	go handleBroadcasts()

// 	router.GET("/ws", func(c *gin.Context) {
// 		handleConnections(c.Writer, c.Request, client)
// 	})
// }

// func handleConnections(w http.ResponseWriter, r *http.Request, client *mongo.Client) {
// 	userId := r.URL.Query().Get("userId")
// 	if userId == "" {
// 		http.Error(w, "User ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	ws, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Println("WebSocket upgrade error:", err)
// 		return
// 	}
// 	defer ws.Close()

// 	mutex.Lock()
// 	clients[ws] = userId
// 	mutex.Unlock()

// 	// Use different collections for common and user-specific data
// 	commonColl := client.Database("FunRepDB").Collection("RealtimeData")
// 	userColl := client.Database("FunRepDB").Collection("Users")

// 	go sendCommonData(ws, commonColl)
// 	go sendUserSpecificData(ws, userId, userColl)

// 	for {
// 		_, _, err := ws.ReadMessage()
// 		if err != nil {
// 			mutex.Lock()
// 			delete(clients, ws)
// 			mutex.Unlock()
// 			break
// 		}
// 	}
// }

// // Send common data to all clients
// func sendCommonData(ws *websocket.Conn, commonColl *mongo.Collection) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	// Define projection to exclude sensitive fields
// 	projection := bson.M{
// 		"roundsData1": 0,
// 		"roundsData2": 0,
// 	}

// 	// Fetch common data (e.g., all documents, excluding certain fields)
// 	cursor, err := commonColl.Find(ctx, bson.M{}, options.Find().SetProjection(projection))
// 	if err != nil {
// 		log.Println("Initial Find error:", err)
// 		return
// 	}
// 	defer cursor.Close(ctx)

// 	// Send each document as a WebSocket message
// 	for cursor.Next(ctx) {
// 		var doc bson.M
// 		if err := cursor.Decode(&doc); err == nil {
// 			if data, err := json.Marshal(doc); err == nil {
// 				ws.WriteMessage(websocket.TextMessage, data) // Send document to client
// 			}
// 		}
// 	}
// }

// // Send user-specific data to the connected user
// func sendUserSpecificData(ws *websocket.Conn, userId string, userColl *mongo.Collection) {
// 	// Query MongoDB for user-specific data (e.g., isBetLocked)
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	var userDoc bson.M
// 	err := userColl.FindOne(ctx, bson.M{"Id": userId}).Decode(&userDoc)
// 	if err != nil {
// 		log.Println("Error fetching user data:", err)
// 		return
// 	}

// 	// Extract and send user-specific data (e.g., isBetLocked)
// 	userData := map[string]interface{}{
// 		"userId":      userId,
// 		"isBetLocked": userDoc["isBetLocked"],
// 	}

// 	// Marshal and send to client
// 	data, err := json.Marshal(userData)
// 	if err != nil {
// 		log.Println("Error marshalling user data:", err)
// 		return
// 	}

// 	if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
// 		log.Println("Error sending message to client:", err)
// 	}
// }

// func handleBroadcasts() {
// 	for {
// 		// Handle new messages to broadcast to clients
// 		msg := <-broadcast
// 		mutex.Lock()
// 		for client := range clients {
// 			// Send message to all connected clients
// 			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
// 				client.Close()          // Close client connection on error
// 				delete(clients, client) // Remove client from map
// 			}
// 		}
// 		mutex.Unlock()
// 	}
// }

// func watchChanges(ctx context.Context, coll *mongo.Collection) {
// 	// MongoDB change stream to listen for changes in the collection
// 	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
// 	stream, err := coll.Watch(ctx, mongo.Pipeline{}, opts)
// 	if err != nil {
// 		log.Fatal("Mongo watch error:", err)
// 	}
// 	defer stream.Close(ctx)

// 	// Continuously check for changes in the MongoDB collection
// 	for stream.Next(ctx) {
// 		var event bson.M
// 		if err := stream.Decode(&event); err == nil {
// 			if fullDoc, ok := event["fullDocument"]; ok {
// 				// If full document exists in the change event, broadcast the update
// 				if data, err := json.Marshal(fullDoc); err == nil {
// 					broadcast <- data // Send update to broadcast channel
// 				}
// 			}
// 		}
// 	}
// }

// func handleConnections(w http.ResponseWriter, r *http.Request, coll *mongo.Collection) {
// 	// Extract userId from query params
// 	userId := r.URL.Query().Get("userId")
// 	if userId == "" {
// 		http.Error(w, "User ID is required", http.StatusBadRequest)
// 		return
// 	}

// 	// Upgrade HTTP connection to WebSocket
// 	ws, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Println("WebSocket upgrade error:", err)
// 		return
// 	}
// 	defer ws.Close()

// 	// Register the user and their WebSocket connection
// 	mutex.Lock()
// 	clients[ws] = userId
// 	mutex.Unlock()

// 	commonColl := mongoClient.Database("FunRepDB").Collection("CommonData")
// 	userColl := mongoClient.Database("FunRepDB").Collection("UserData")
// 	// Send initial common data to the client
// 	go sendCommonData(ws, coll)

// 	// Send user-specific data to the client
// 	go sendUserSpecificData(ws, userId, coll)

// 	// Keep connection alive and remove on disconnect
// 	for {
// 		_, _, err := ws.ReadMessage()
// 		if err != nil {
// 			// Remove client from map when disconnected
// 			mutex.Lock()
// 			delete(clients, ws)
// 			mutex.Unlock()
// 			break
// 		}
// 	}
// }

//==========================================================================================================

package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan []byte)
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	mutex sync.Mutex
)

func InitWebSocket(router *gin.Engine, coll *mongo.Collection) {
	go watchChanges(context.Background(), coll) // Listen for MongoDB changes
	go handleBroadcasts()                       // Broadcast changes to WebSocket clients

	router.GET("/ws", func(c *gin.Context) {
		handleConnections(c.Writer, c.Request, coll)
	})
}

func handleConnections(w http.ResponseWriter, r *http.Request, coll *mongo.Collection) {
	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer ws.Close()

	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	// Send initial full collection data to the client
	go func(conn *websocket.Conn) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Define projection to exclude sensitive fields (e.g., "roundsData1", "roundsData2")
		projection := bson.M{
			"roundsData1": 0,
			"roundsData2": 0,
		}

		// Fetch documents from MongoDB with the projection applied
		cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetProjection(projection))
		if err != nil {
			log.Println("Initial Find error:", err)
			return
		}
		defer cursor.Close(ctx)

		// Send all documents to the client
		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err == nil {
				if data, err := json.Marshal(doc); err == nil {
					conn.WriteMessage(websocket.TextMessage, data) // Send document to WebSocket client
				}
			}
		}
	}(ws)

	// Keep connection alive and remove on disconnect
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			// If client disconnects, remove it from the clients map
			mutex.Lock()
			delete(clients, ws)
			mutex.Unlock()
			break
		}
	}
}

func handleBroadcasts() {
	for {
		// Handle new messages to broadcast to clients
		msg := <-broadcast
		mutex.Lock()
		for client := range clients {
			// Send message to all connected clients
			if err := client.WriteMessage(websocket.TextMessage, msg); err != nil {
				client.Close()          // Close client connection on error
				delete(clients, client) // Remove client from map
			}
		}
		mutex.Unlock()
	}
}

func watchChanges(ctx context.Context, coll *mongo.Collection) {
	// MongoDB change stream to listen for changes in the collection
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
	stream, err := coll.Watch(ctx, mongo.Pipeline{}, opts)
	if err != nil {
		log.Fatal("Mongo watch error:", err)
	}
	defer stream.Close(ctx)

	// Continuously check for changes in the MongoDB collection
	for stream.Next(ctx) {
		var event bson.M
		if err := stream.Decode(&event); err == nil {
			if fullDoc, ok := event["fullDocument"]; ok {
				// If full document exists in the change event, broadcast the update
				if data, err := json.Marshal(fullDoc); err == nil {
					broadcast <- data // Send update to broadcast channel
				}
			}
		}
	}
}
