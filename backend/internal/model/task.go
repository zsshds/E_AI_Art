package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusFailed     TaskStatus = "failed"
)

type Task struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	StyleProfileID   primitive.ObjectID `json:"style_profile_id" bson:"style_profile_id"`
	UserInput        string             `json:"user_input" bson:"user_input"`
	Model            string             `json:"model" bson:"model"`
	Size             string             `json:"size" bson:"size"`
	APIQuality       string             `json:"api_quality" bson:"api_quality"`
	ProjectID        string             `json:"project_id" bson:"project_id"`
	FinalPrompt      string             `json:"final_prompt" bson:"final_prompt"`
	Status           TaskStatus         `json:"status" bson:"status"`
	ResultImageURL   string             `json:"result_image_url" bson:"result_image_url"`
	ErrorMessage     string             `json:"error_message" bson:"error_message"`
	PlatformTaskID   string             `json:"platform_task_id" bson:"platform_task_id"`
	Progress         int                `json:"progress" bson:"progress"`
	RetryCount       int                `json:"retry_count" bson:"retry_count"`
	CreatedBy        string             `json:"created_by" bson:"created_by"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updated_at"`
}
