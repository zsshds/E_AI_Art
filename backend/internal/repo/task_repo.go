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

const (
	DefaultTaskListPage     = 1
	DefaultTaskListPageSize = 20
	MaxTaskListPageSize     = 100
)

type TaskRepo struct {
	collection *mongo.Collection
}

func NewTaskRepo(db *mongo.Database) *TaskRepo {
	return &TaskRepo{
		collection: db.Collection("tasks"),
	}
}

func (r *TaskRepo) Create(ctx context.Context, task *model.Task) error {
	task.ID = primitive.NewObjectID()
	task.Status = model.TaskStatusPending
	task.RetryCount = 0
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (r *TaskRepo) GetByID(ctx context.Context, id string) (*model.Task, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	var task model.Task
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&task)
	if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}
	return &task, nil
}

func (r *TaskRepo) List(ctx context.Context, filter bson.M, page, pageSize int) ([]model.Task, int64, error) {
	if filter == nil {
		filter = bson.M{}
	}
	if page <= 0 {
		page = DefaultTaskListPage
	}
	if pageSize <= 0 {
		pageSize = DefaultTaskListPageSize
	}
	if pageSize > MaxTaskListPageSize {
		pageSize = MaxTaskListPageSize
	}

	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []model.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, 0, fmt.Errorf("decode tasks: %w", err)
	}
	return tasks, total, nil
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

func (r *TaskRepo) UpdateResult(ctx context.Context, id string, imageURL string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	update := bson.M{
		"$set": bson.M{
			"status":           model.TaskStatusDone,
			"result_image_url": imageURL,
			"updated_at":       time.Now(),
		},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

func (r *TaskRepo) UpdateFailure(ctx context.Context, id string, errMsg string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	update := bson.M{
		"$set": bson.M{
			"status":        model.TaskStatusFailed,
			"error_message": errMsg,
			"updated_at":    time.Now(),
		},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

func (r *TaskRepo) IncrementRetry(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	update := bson.M{
		"$inc": bson.M{"retry_count": 1},
		"$set": bson.M{"updated_at": time.Now()},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

// GetConversation walks the parent_task_id chain upward and returns tasks from root to the given task.
func (r *TaskRepo) GetConversation(ctx context.Context, taskID string) ([]model.Task, error) {
	const maxDepth = 50
	ids := make([]primitive.ObjectID, 0, maxDepth)

	currentID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	ids = append(ids, currentID)

	for i := 0; i < maxDepth; i++ {
		var task model.Task
		err := r.collection.FindOne(ctx, bson.M{"_id": currentID}).Decode(&task)
		if err != nil {
			return nil, fmt.Errorf("find task %s: %w", currentID.Hex(), err)
		}
		if task.ParentTaskID.IsZero() {
			break
		}
		currentID = task.ParentTaskID
		ids = append(ids, currentID)
	}

	// Reverse so we return root-first
	result := make([]model.Task, len(ids))
	for i, id := range ids {
		var task model.Task
		err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
		if err != nil {
			return nil, fmt.Errorf("find task %s: %w", id.Hex(), err)
		}
		result[len(ids)-1-i] = task
	}

	return result, nil
}

// HasActiveChild checks whether a task has a child task in pending or processing status.
func (r *TaskRepo) HasActiveChild(ctx context.Context, parentID string) (bool, error) {
	objID, err := primitive.ObjectIDFromHex(parentID)
	if err != nil {
		return false, fmt.Errorf("invalid id: %w", err)
	}

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"parent_task_id": objID,
		"status":         bson.M{"$in": []string{"pending", "processing"}},
	})
	if err != nil {
		return false, fmt.Errorf("count active children: %w", err)
	}
	return count > 0, nil
}

// UpdateFinalPrompt persists the generated final_prompt after prompt building.
func (r *TaskRepo) UpdateFinalPrompt(ctx context.Context, id string, finalPrompt string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}

	update := bson.M{
		"$set": bson.M{
			"final_prompt": finalPrompt,
			"updated_at":   time.Now(),
		},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}

func (r *TaskRepo) ListStaleProcessing(ctx context.Context, before time.Time) ([]model.Task, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"status":     model.TaskStatusProcessing,
		"updated_at": bson.M{"$lt": before},
	})
	if err != nil {
		return nil, fmt.Errorf("find stale processing tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []model.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("decode stale processing tasks: %w", err)
	}
	return tasks, nil
}
