package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// models

type RideFareModel struct {
	ID                primitive.ObjectID
	UserID            string
	PackageSlug       string // ex: van, luxury, etc...
	TotalPriceInCents float64
}
