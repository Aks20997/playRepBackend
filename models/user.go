package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty"`
	UserId                 string             `bson:"Id"`
	PasswordHash           string             `bson:"PasswordHash"`
	Points                 float64            `bson:"points"`
	IsBetLocked            bool               `bson:"isBetLocked"`
	IsBetLockedFunTarget   bool               `bson:"isBetLockedFunTarget"`
	WinningsRoulette       float64            `bson:"winningsRoulette"`
	WinningsFunTarget      float64            `bson:"winningsFunTarget"`
	WinningNumberRoulette  int                `bson:"winningNumberRoulette"`
	WinningNumberFunTarget int                `bson:"winningNumberFunTarget"`
	TempWinnings1          float64            `bson:"TempWinnings1"`
	TempWinnings2          float64            `bson:"TempWinnings2"`
}
