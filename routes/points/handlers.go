package points

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"FunRepBackend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	UserCollection      *mongo.Collection
	PointsCollection    *mongo.Collection
	CommissionCollection *mongo.Collection
)

func contains(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func CreatePointRequest(c *gin.Context) {
	log.Println("Received createPointRequest")

	type RequestBody struct {
		ReceiverId string  `json:"receiverId"`
		Points     float64 `json:"points"`
		Pin        int     `json:"pin"`
	}

	var req RequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("Invalid request body:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	log.Println("Parsed request body:", req)

	senderIdValue, exists := c.Get("userId")
	if !exists {
		log.Println("userId not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	senderId, ok := senderIdValue.(string)
	if !ok {
		log.Println("userId is not a valid string")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
		return
	}

	log.Println("Sender ID:", senderId)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var sender, receiver models.User
	err := UserCollection.FindOne(ctx, bson.M{"Id": senderId}).Decode(&sender)
	if err != nil {
		log.Println("Sender not found:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Sender not found"})
		return
	}
	log.Println("Sender loaded:", sender.UserId, sender.Type, "Parent:", sender.Parent)

	err = UserCollection.FindOne(ctx, bson.M{"Id": req.ReceiverId}).Decode(&receiver)
	if err != nil {
		log.Println("Receiver not found:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Receiver not found"})
		return
	}
	log.Println("Receiver loaded:", receiver.UserId, receiver.Type)

	// Type validation
	switch sender.Type {
	case "Company":
		if sender.UserId == "GK00500555" && receiver.Type == "Master" {
			// Allow
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Company can only send to Master"})
			return
		}
	case "Master":
		// Can send to anyone
	case "Dealer":
		if req.ReceiverId == sender.Parent {
			// Valid refund to Master
		} else if contains(sender.Childs, receiver.UserId) &&
			(receiver.Type == "Sub-Dealer" || receiver.Type == "Customer") {
			// Valid child
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Dealer can only send to its own children or parent"})
			return
		}
	case "Sub-Dealer":
		if req.ReceiverId == sender.Parent {
			// Refund
		} else if contains(sender.Childs, receiver.UserId) ||
			(receiver.Type == "Sub-Dealer" && receiver.Parent == sender.Parent) {
			// Allow
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sub-Dealer can only send to children or peer Sub-Dealers"})
			return
		}
	case "Customer":
		if req.ReceiverId == sender.Parent {
			// ✅ Allow sending to parent (Dealer or Sub-Dealer)
		} else if receiver.Type == "Customer" && receiver.Parent == sender.Parent {
			// ✅ Allow sending to sibling Customer (same parent)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Customer can only send to their parent or sibling customers with same parent"})
			return
		}
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized sender type"})
		return
	}

	if sender.Pin != req.Pin {
		log.Println("Invalid PIN entered")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid PIN"})
		return
	}

	if sender.Points < req.Points {
		log.Println("Insufficient points")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient points"})
		return
	}

	session, err := UserCollection.Database().Client().StartSession()
	if err != nil {
		log.Println("Failed to start DB session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start session"})
		return
	}
	defer session.EndSession(ctx)

	istLoc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Println("Failed to load IST:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load IST location"})
		return
	}

	var insertedID primitive.ObjectID

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		log.Println("Starting transaction")
		if err := session.StartTransaction(); err != nil {
			return err
		}

		requestType := "P"

		res, err := PointsCollection.InsertOne(sc, bson.M{
			"senderId":   senderId,
			"receiverId": req.ReceiverId,
			"points":     req.Points,
			"status":     "pending",
			"type":       requestType,
			"timestamp":  time.Now().In(istLoc),
		})
		if err != nil {
			log.Println("Insert into pointsCollection failed:", err)
			session.AbortTransaction(sc)
			return err
		}
		insertedID = res.InsertedID.(primitive.ObjectID)
		log.Println("Inserted point request with ID:", insertedID.Hex())

		_, err = UserCollection.UpdateOne(sc, bson.M{"Id": senderId}, bson.M{
			"$inc": bson.M{"points": -req.Points},
		})
		if err != nil {
			log.Println("Failed to decrement sender points:", err)
			session.AbortTransaction(sc)
			return err
		}
		log.Println("Sender points updated")

		// Decrease initial_fund if refunding to parent
		if (sender.Type == "Master" || sender.Type == "Dealer" || sender.Type == "Sub-Dealer") && req.ReceiverId == sender.Parent {
			_, err := UserCollection.UpdateOne(sc, bson.M{"Id": senderId}, bson.M{
				"$inc": bson.M{"initial_fund": -req.Points},
			})
			if err != nil {
				log.Println("Failed to decrement initial_fund:", err)
				session.AbortTransaction(sc)
				return err
			}
		}

		// Increase initial_fund if Company sends to Master
		if sender.Type == "Company" && sender.UserId == "GK00500555" && receiver.Type == "Master" {
			_, err := UserCollection.UpdateOne(sc, bson.M{"Id": req.ReceiverId}, bson.M{
				"$inc": bson.M{"initial_fund": req.Points},
			})
			if err != nil {
				log.Println("Failed to increment initial_fund to Master:", err)
				session.AbortTransaction(sc)
				return err
			}
		}

		log.Println("Committing transaction")
		return session.CommitTransaction(sc)
	})

	if err != nil {
		log.Println("Transaction failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed", "details": err.Error()})
		return
	}

	log.Println("Request created successfully")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Request created successfully", "requestId": insertedID.Hex()})
}

func ReceivePointRequest(c *gin.Context) {
	type RequestBody struct {
		RequestId string `json:"requestId"`
	}

	var req RequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userIdValue, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userId := userIdValue.(string)

	reqObjectId, err := primitive.ObjectIDFromHex(req.RequestId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pointRequest struct {
		ID         primitive.ObjectID `bson:"_id"`
		SenderId   string             `bson:"senderId"`
		ReceiverId string             `bson:"receiverId"`
		Points     float64            `bson:"points"`
		Status     string             `bson:"status"`
		Type       string             `bson:"type"`
	}
	err = PointsCollection.FindOne(ctx, bson.M{"_id": reqObjectId}).Decode(&pointRequest)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	var receiverUser models.User
	err = UserCollection.FindOne(ctx, bson.M{"Id": userId}).Decode(&receiverUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	session, err := UserCollection.Database().Client().StartSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start session"})
		return
	}
	defer session.EndSession(ctx)

	// Case 1: Receiver accepting a pending request
	if pointRequest.Status == "pending" && pointRequest.ReceiverId == userId {
		err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
			if err := session.StartTransaction(); err != nil {
				return err
			}

			_, err := UserCollection.UpdateOne(sc, bson.M{"Id": userId}, bson.M{
				"$inc": bson.M{"points": pointRequest.Points},
			})
			if err != nil {
				session.AbortTransaction(sc)
				return err
			}

			_, err = PointsCollection.UpdateOne(sc, bson.M{"_id": reqObjectId}, bson.M{
				"$set": bson.M{"status": "received"},
			})
			if err != nil {
				session.AbortTransaction(sc)
				return err
			}

			if receiverUser.Type == "Master" || receiverUser.Type == "Dealer" || receiverUser.Type == "Sub-Dealer" {
				if pointRequest.SenderId == receiverUser.Parent {
					_, err := UserCollection.UpdateOne(sc, bson.M{"Id": userId}, bson.M{
						"$inc": bson.M{"initial_fund": pointRequest.Points},
					})
					if err != nil {
						session.AbortTransaction(sc)
						return err
					}
				}
			}

			return session.CommitTransaction(sc)
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Points received successfully"})
		return
	}

	// Case 2: Sender accepting a rejection from receiver (parent accepting child's rejection)
	if pointRequest.Status == "pending_rejected" && pointRequest.SenderId == userId {
		var senderUser models.User
		err = UserCollection.FindOne(ctx, bson.M{"Id": userId}).Decode(&senderUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sender user"})
			return
		}

		err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
			if err := session.StartTransaction(); err != nil {
				return err
			}

			// Refund points to sender (parent)
			_, err := UserCollection.UpdateOne(sc, bson.M{"Id": userId}, bson.M{
				"$inc": bson.M{"points": pointRequest.Points},
			})
			if err != nil {
				session.AbortTransaction(sc)
				return err
			}

			// Mark request as rejected_by_receiver
			_, err = PointsCollection.UpdateOne(sc, bson.M{"_id": reqObjectId}, bson.M{
				"$set": bson.M{"status": "rejected_by_receiver"},
			})
			if err != nil {
				session.AbortTransaction(sc)
				return err
			}

			// Restore initial_fund if this was a child-to-parent request
			var receiverUser models.User
			err2 := UserCollection.FindOne(sc, bson.M{"Id": pointRequest.ReceiverId}).Decode(&receiverUser)
			if err2 == nil &&
				(senderUser.Type == "Master" || senderUser.Type == "Dealer" || senderUser.Type == "Sub-Dealer") &&
				senderUser.Parent == receiverUser.UserId {
				_, err := UserCollection.UpdateOne(sc, bson.M{"Id": userId}, bson.M{
					"$inc": bson.M{"initial_fund": pointRequest.Points},
				})
				if err != nil {
					session.AbortTransaction(sc)
					return err
				}
			}

			return session.CommitTransaction(sc)
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Rejection accepted, points refunded successfully"})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request state or unauthorized action"})
}

func RejectPointRequest(c *gin.Context) {
	type RequestBody struct {
		RequestId string `json:"requestId"`
	}

	var req RequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userIdValue, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userId := userIdValue.(string)

	reqObjectId, err := primitive.ObjectIDFromHex(req.RequestId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pointRequest struct {
		ID         primitive.ObjectID `bson:"_id"`
		SenderId   string             `bson:"senderId"`
		ReceiverId string             `bson:"receiverId"`
		Points     float64            `bson:"points"`
		Status     string             `bson:"status"`
	}
	err = PointsCollection.FindOne(ctx, bson.M{"_id": reqObjectId}).Decode(&pointRequest)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	var updateStatus string
	var refundTo string

	switch {
	case pointRequest.Status == "pending":
		if userId == pointRequest.SenderId {
			updateStatus = "rejected_by_sender"
			refundTo = pointRequest.SenderId
		} else if userId == pointRequest.ReceiverId {
			updateStatus = "pending_rejected"
			refundTo = ""
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "User is neither sender nor receiver of this request"})
			return
		}
	case pointRequest.Status == "pending_rejected":
		c.JSON(http.StatusBadRequest, gin.H{"error": "This request has already been rejected. Please wait for sender to handle it."})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request is not in a valid state to reject"})
		return
	}

	session, err := UserCollection.Database().Client().StartSession()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start session"})
		return
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return err
		}

		if refundTo != "" {
			_, err = UserCollection.UpdateOne(sc, bson.M{"Id": refundTo}, bson.M{
				"$inc": bson.M{"points": pointRequest.Points},
			})
			if err != nil {
				session.AbortTransaction(sc)
				return err
			}
		}

		_, err = PointsCollection.UpdateOne(sc, bson.M{"_id": reqObjectId}, bson.M{
			"$set": bson.M{"status": updateStatus},
		})

		if updateStatus == "rejected_by_sender" || updateStatus == "rejected_by_receiver" {
			var senderUser models.User
			var receiverUser models.User

			err1 := UserCollection.FindOne(sc, bson.M{"Id": pointRequest.SenderId}).Decode(&senderUser)
			err2 := UserCollection.FindOne(sc, bson.M{"Id": pointRequest.ReceiverId}).Decode(&receiverUser)

			if err1 == nil && err2 == nil &&
				(senderUser.Type == "Master" || senderUser.Type == "Dealer" || senderUser.Type == "Sub-Dealer") &&
				senderUser.Parent == receiverUser.UserId {

				_, err := UserCollection.UpdateOne(sc, bson.M{"Id": senderUser.UserId}, bson.M{
					"$inc": bson.M{"initial_fund": pointRequest.Points},
				})
				if err != nil {
					session.AbortTransaction(sc)
					return err
				}
			}
		}

		if err != nil {
			session.AbortTransaction(sc)
			return err
		}

		return session.CommitTransaction(sc)
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Request rejected successfully"})
}

func GetPointRequests(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userIdValue, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userId, ok := userIdValue.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
		return
	}

	status := c.Query("status")

	filter := bson.M{
		"$or": []bson.M{
			{"senderId": userId},
			{"receiverId": userId},
		},
	}

	if status != "" {
		statusList := strings.Split(status, ",")
		filter["status"] = bson.M{"$in": statusList}
	}

	cursor, err := PointsCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests"})
		return
	}
	defer cursor.Close(ctx)

	var requests []bson.M
	if err := cursor.All(ctx, &requests); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func CalculateAndDistributeCommission() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cursor, err := UserCollection.Find(ctx, bson.M{
		"Type": bson.M{"$in": []string{"Master", "Dealer", "Sub-Dealer"}},
	})
	if err != nil {
		log.Println("❌ Failed to fetch users:", err)
		return
	}
	defer cursor.Close(ctx)

	users := map[string]models.User{}
	var userList []models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err == nil {
			users[user.UserId] = user
			userList = append(userList, user)
		}
	}

	priority := map[string]int{"Master": 1, "Dealer": 2, "Sub-Dealer": 3}
	sort.Slice(userList, func(i, j int) bool {
		return priority[userList[i].Type] < priority[userList[j].Type]
	})

	commissionMap := map[string]float64{}
	skipUser := map[string]bool{}

	for _, user := range userList {
		userID := user.UserId

		if user.InitialFund < 0 {
			lossCommission := math.Abs(user.InitialFund) * (user.Loss_Comission / 100)
			commissionMap[userID] = lossCommission
			skipUser[userID] = true

			newFund := -1 * (math.Abs(user.InitialFund) + lossCommission)
			_, _ = UserCollection.UpdateOne(ctx, bson.M{"Id": userID}, bson.M{
				"$set": bson.M{"initial_fund": newFund, "inLoss": true},
			})
			continue
		}

		profitCommission := user.InitialFund * (user.Profit_Comission / 100)

		if _, ok := commissionMap[userID]; !ok {
			commissionMap[userID] = 0
		}
		commissionMap[userID] += profitCommission

		parentID := user.Parent
		if _, ok := users[parentID]; ok && !skipUser[parentID] {
			if _, ok := commissionMap[parentID]; !ok {
				commissionMap[parentID] = 0
			}
			if commissionMap[parentID] >= profitCommission {
				commissionMap[parentID] -= profitCommission
			} else {
				diff := commissionMap[parentID]
				commissionMap[userID] -= (profitCommission - diff)
				commissionMap[parentID] = 0
			}
		}

		_, _ = UserCollection.UpdateOne(ctx, bson.M{"Id": userID}, bson.M{
			"$set": bson.M{"initial_fund": 0},
		})
	}

	now := time.Now()
	for userID, amount := range commissionMap {
		if amount <= 0 || skipUser[userID] {
			continue
		}

		user := users[userID]
		senderID := user.Parent
		if user.Type == "Master" {
			senderID = "GK00555078"
		}

		_, err := PointsCollection.InsertOne(ctx, bson.M{
			"senderId":   senderID,
			"receiverId": userID,
			"points":     amount,
			"status":     "pending",
			"type":       "G",
			"timestamp":  now,
		})
		if err != nil {
			log.Println("Failed to insert commission for", userID, ":", err)
			continue
		}

		timestampKey := now.Format("2006-01-02T15:04:05Z07:00")
		_, _ = CommissionCollection.UpdateOne(ctx, bson.M{"userId": userID}, bson.M{
			"$set": bson.M{
				fmt.Sprintf("commissions.%s", timestampKey): bson.M{
					"percentage":        user.Profit_Comission,
					"initial_fund":      user.InitialFund,
					"actual_commission": amount,
				},
			},
		}, options.Update().SetUpsert(true))
	}

	log.Println("✅ Commission distribution complete")
}

