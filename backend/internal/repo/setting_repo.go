package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/imagegen/backend/internal/model"
)

type SettingRepo struct {
	col *mongo.Collection
}

func NewSettingRepo(db *mongo.Database) *SettingRepo {
	return &SettingRepo{col: db.Collection("settings")}
}

func (r *SettingRepo) Get(ctx context.Context, key string) (string, error) {
	var s model.Setting
	err := r.col.FindOne(ctx, bson.M{"_id": key}).Decode(&s)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

func (r *SettingRepo) Set(ctx context.Context, key, value string) error {
	opts := options.Update().SetUpsert(true)
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": key}, bson.M{
		"$set": bson.M{"value": value, "updated_at": time.Now()},
	}, opts)
	return err
}

func (r *SettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	cursor, err := r.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	result := make(map[string]string)
	for cursor.Next(ctx) {
		var s model.Setting
		if err := cursor.Decode(&s); err != nil {
			continue
		}
		result[s.Key] = s.Value
	}
	return result, nil
}
