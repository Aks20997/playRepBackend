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
	WinningsTripleFun      float64            `bson:"winningsTripleFun"`
	WinningsAB             float64            `bson:"winningsAB"`
	TotalWinningsAB        float64            `bson:"totalWinningsAB"`
	WinningNumberRoulette  int                `bson:"winningNumberRoulette"`
	WinningNumberFunTarget int                `bson:"winningNumberFunTarget"`
	TempWinnings1          float64            `bson:"TempWinnings1"`
	TempWinnings2          float64            `bson:"TempWinnings2"`
	TempWinningsTF         float64            `bson:"TempWinningsTF"`
	TempWinningsAB         float64            `bson:"TempWinningsAB"`
	TempTotalWinningsAB    float64            `bson:"TempTotalWinningsAB"`
	Pin                    int                `bson:"pin"`
	Type                   string             `bson:"Type"`
	Childs                 []string           `bson:"Childs"`
	NextABArray            []int              `bson:"nextABArray"`
	Parent                 string             `bson:"parent"`
	IsActive               bool               `bson:"isActive"`
	IsOnline               bool               `bson:"isOnline"`
	InitialFund            float64            `bson:"initial_fund"`
	Profit_Comission       float64            `bson:"Profit_Comission"`
	Loss_Comission         float64            `bson:"Loss_Comission"`
	InLoss                 bool               `bson:"inLoss"`
}
