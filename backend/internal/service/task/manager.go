package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/service/image"
	"github.com/imagegen/backend/internal/service/prompt"
)

type Manager struct {
	rabbitConn  *amqp.Connection
	imageClient *image.Client
	concurrency int
	maxRetry    int
}

type TaskMessage struct {
	TaskID string `json:"task_id"`
}

func NewManager(rabbitURI string, imageClient *image.Client, concurrency, maxRetry int) (*Manager, error) {
	conn, err := amqp.Dial(rabbitURI)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	return &Manager{
		rabbitConn:  conn,
		imageClient: imageClient,
		concurrency: concurrency,
		maxRetry:    maxRetry,
	}, nil
}

func (m *Manager) PublishTask(ctx context.Context, taskID string) error {
	ch, err := m.rabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	msg := TaskMessage{TaskID: taskID}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	return ch.PublishWithContext(ctx,
		"image_tasks", // exchange
		"task",        // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)
}

type TaskProcessor interface {
	GetTask(ctx context.Context, taskID string) (*model.Task, error)
	GetStyleProfile(ctx context.Context, profileID string) (*model.StyleProfile, error)
	UpdateTaskResult(ctx context.Context, taskID string, imageURL string) error
	UpdateTaskFailure(ctx context.Context, taskID string, errMsg string) error
}

// ProcessTask handles a single task through the full pipeline
func (m *Manager) ProcessTask(ctx context.Context, task *model.Task, profile *model.StyleProfile, processor TaskProcessor) error {
	builder := prompt.NewPromptBuilder(profile)
	apiReq := builder.BuildAPIRequest(task.UserInput)
	task.FinalPrompt = apiReq.Prompt

	log.Printf("[task %s] generating image with prompt: %s", task.ID.Hex(), task.FinalPrompt)

	var result *image.GenerationResponse
	var err error

	if profile.ReferenceImageURL != "" {
		result, err = m.imageClient.Edit(ctx, apiReq, profile.ReferenceImageURL)
	} else {
		result, err = m.imageClient.Generate(ctx, apiReq)
	}

	if err != nil {
		return fmt.Errorf("image generation: %w", err)
	}

	if len(result.Data) == 0 {
		return fmt.Errorf("no image data in response")
	}

	imageURL := result.Data[0].URL
	if err := processor.UpdateTaskResult(ctx, task.ID.Hex(), imageURL); err != nil {
		return fmt.Errorf("update task result: %w", err)
	}

	log.Printf("[task %s] completed successfully", task.ID.Hex())
	return nil
}

func (m *Manager) Close() {
	if m.rabbitConn != nil {
		m.rabbitConn.Close()
	}
}

// Helper to create a context with timeout
func (m *Manager) TaskContext(timeoutSec int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
}
