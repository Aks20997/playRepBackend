package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type PointsTransferRequest struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	SenderID   string             `bson:"senderId"`
	ReceiverID string             `bson:"receiverId"`
	Amount     float64            `bson:"amount"`
	Status     string             `bson:"status"` // pending, approved, rejected
	CreatedAt  primitive.DateTime `bson:"createdAt"`
}
