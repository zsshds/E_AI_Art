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

type ProjectRepo struct {
	collection *mongo.Collection
}

func NewProjectRepo(db *mongo.Database) *ProjectRepo {
	return &ProjectRepo{collection: db.Collection("projects")}
}

func (r *ProjectRepo) Create(ctx context.Context, project *model.Project) error {
	project.ID = primitive.NewObjectID()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, project)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

func (r *ProjectRepo) GetByID(ctx context.Context, id string) (*model.Project, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	var project model.Project
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&project)
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}
	return &project, nil
}

func (r *ProjectRepo) List(ctx context.Context) ([]model.Project, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer cursor.Close(ctx)

	var projects []model.Project
	if err := cursor.All(ctx, &projects); err != nil {
		return nil, fmt.Errorf("decode projects: %w", err)
	}
	if projects == nil {
		projects = []model.Project{}
	}
	return projects, nil
}

func (r *ProjectRepo) Update(ctx context.Context, id string, project *model.Project) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	project.UpdatedAt = time.Now()
	_, err = r.collection.ReplaceOne(ctx, bson.M{"_id": objID}, project)
	return err
}

func (r *ProjectRepo) Delete(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}

// GetProjectsForUser returns all projects that contain the given username as a member.
func (r *ProjectRepo) GetProjectsForUser(ctx context.Context, username string) ([]model.Project, error) {
	filter := bson.M{"members": username}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find user projects: %w", err)
	}
	defer cursor.Close(ctx)

	var projects []model.Project
	if err := cursor.All(ctx, &projects); err != nil {
		return nil, fmt.Errorf("decode projects: %w", err)
	}
	if projects == nil {
		projects = []model.Project{}
	}
	return projects, nil
}
