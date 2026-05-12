package repo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/imagegen/backend/internal/model"
)

type StyleProfileRepo struct {
	collection *mongo.Collection
}

func NewStyleProfileRepo(db *mongo.Database) *StyleProfileRepo {
	return &StyleProfileRepo{
		collection: db.Collection("style_profiles"),
	}
}

func (r *StyleProfileRepo) Create(ctx context.Context, profile *model.StyleProfile) error {
	profile.ID = primitive.NewObjectID()
	profile.Version = 1
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, profile)
	if err != nil {
		return fmt.Errorf("insert style profile: %w", err)
	}
	return nil
}

func (r *StyleProfileRepo) GetByID(ctx context.Context, id string) (*model.StyleProfile, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	var profile model.StyleProfile
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&profile)
	if err != nil {
		return nil, fmt.Errorf("find style profile: %w", err)
	}
	return &profile, nil
}

func (r *StyleProfileRepo) List(ctx context.Context, createdBy string, projectIDs []string) ([]model.StyleProfile, error) {
	filter := bson.M{}
	if createdBy != "" {
		filter["created_by"] = createdBy
	}
	if len(projectIDs) > 0 {
		filter["project_id"] = bson.M{"$in": projectIDs}
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("list style profiles: %w", err)
	}
	defer cursor.Close(ctx)

	var profiles []model.StyleProfile
	if err := cursor.All(ctx, &profiles); err != nil {
		return nil, fmt.Errorf("decode style profiles: %w", err)
	}
	return profiles, nil
}

func (r *StyleProfileRepo) Update(ctx context.Context, id string, profile *model.StyleProfile) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	profile.UpdatedAt = time.Now()
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objID}, profile)
	if err != nil {
		return fmt.Errorf("update style profile: %w", err)
	}
	return nil
}