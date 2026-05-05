package model

import "time"

type Setting struct {
	Key       string    `json:"key" bson:"_id"`
	Value     string    `json:"value" bson:"value"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}
