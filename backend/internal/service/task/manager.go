package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/imagegen/backend/internal/model"
	"github.com/imagegen/backend/internal/repo"
	"github.com/imagegen/backend/internal/service/image"
	"github.com/imagegen/backend/internal/service/prompt"
	"github.com/imagegen/backend/internal/ws"
)

type Manager struct {
	rabbitConn       *amqp.Connection
	imageClient      *image.Client
	taskRepo         *repo.TaskRepo
	styleProfileRepo *repo.StyleProfileRepo
	wsHub            *ws.Hub
	concurrency      int
	maxRetry         int
	timeoutSec       int
}

type TaskMessage struct {
	TaskID string `json:"task_id"`
}

func NewManager(rabbitURI string, imageClient *image.Client, taskRepo *repo.TaskRepo, styleProfileRepo *repo.StyleProfileRepo, wsHub *ws.Hub, concurrency, maxRetry, timeoutSec int) (*Manager, error) {
	conn, err := amqp.Dial(rabbitURI)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	m := &Manager{
		rabbitConn:       conn,
		imageClient:      imageClient,
		taskRepo:         taskRepo,
		styleProfileRepo: styleProfileRepo,
		wsHub:            wsHub,
		concurrency:      concurrency,
		maxRetry:         maxRetry,
		timeoutSec:       timeoutSec,
	}

	if err := m.declareTopology(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("declare topology: %w", err)
	}

	return m, nil
}

func (m *Manager) declareTopology() error {
	ch, err := m.rabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare("image_tasks", "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange image_tasks: %w", err)
	}
	if err := ch.ExchangeDeclare("image_tasks_dlx", "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange image_tasks_dlx: %w", err)
	}

	_, err = ch.QueueDeclare("task_queue", true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "image_tasks_dlx",
		"x-dead-letter-routing-key": "dead_tasks",
		"x-message-ttl":             int32(180000),
	})
	if err != nil {
		return fmt.Errorf("declare queue task_queue: %w", err)
	}

	_, err = ch.QueueDeclare("dead_task_queue", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("declare queue dead_task_queue: %w", err)
	}

	if err := ch.QueueBind("task_queue", "task", "image_tasks", false, nil); err != nil {
		return fmt.Errorf("bind task_queue: %w", err)
	}
	if err := ch.QueueBind("dead_task_queue", "dead_tasks", "image_tasks_dlx", false, nil); err != nil {
		return fmt.Errorf("bind dead_task_queue: %w", err)
	}

	log.Println("rabbitmq topology declared")
	return nil
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

// StartWorkerPool spawns N goroutines that consume from RabbitMQ and process tasks.
func (m *Manager) StartWorkerPool(ctx context.Context) {
	for i := 0; i < m.concurrency; i++ {
		go m.worker(ctx, i)
	}
	log.Printf("started %d task workers", m.concurrency)
}

func (m *Manager) worker(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker %d shutting down", id)
			return
		default:
		}

		if err := m.consumeOne(ctx, id); err != nil {
			log.Printf("worker %d consume error: %v", id, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func (m *Manager) consumeOne(ctx context.Context, id int) error {
	ch, err := m.rabbitConn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	// Qos: one message at a time per worker
	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("set qos: %w", err)
	}

	delivery, ok, err := ch.Get("task_queue", false)
	if err != nil {
		return fmt.Errorf("get message: %w", err)
	}
	if !ok {
		return nil // no message available
	}

	var msg TaskMessage
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		delivery.Nack(false, false)
		return fmt.Errorf("unmarshal message: %w", err)
	}

	log.Printf("worker %d processing task %s", id, msg.TaskID)
	m.processTask(ctx, msg.TaskID)

	delivery.Ack(false)
	return nil
}

func (m *Manager) processTask(ctx context.Context, taskID string) {
	taskCtx, cancel := context.WithTimeout(ctx, time.Duration(m.timeoutSec)*time.Second)
	defer cancel()

	// Load task
	task, err := m.taskRepo.GetByID(taskCtx, taskID)
	if err != nil {
		log.Printf("[task %s] load task error: %v", taskID, err)
		return
	}

	// Load style profile
	profile, err := m.styleProfileRepo.GetByID(taskCtx, task.StyleProfileID.Hex())
	if err != nil {
		m.taskRepo.UpdateFailure(taskCtx, taskID, "style profile not found")
		m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "failed", ErrorMessage: "style profile not found"})
		return
	}

	// Update status to processing
	m.taskRepo.UpdateStatus(taskCtx, taskID, model.TaskStatusProcessing)
	m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "processing", Progress: 0})

	// Override profile's model if task specifies one
	if task.Model != "" {
		profile.Model = task.Model
	}

	// Build prompt and API request
	builder := prompt.NewPromptBuilder(profile)
	apiBody := builder.BuildAPIRequest(task.UserInput)
	if msgs, ok := apiBody["messages"].([]map[string]string); ok && len(msgs) > 0 {
		task.FinalPrompt = msgs[0]["content"]
	} else if prompt, ok := apiBody["prompt"].(string); ok {
		task.FinalPrompt = prompt
	}

	log.Printf("[task %s] submitting task to platform, model=%s, prompt=%s", taskID, apiBody["model"], task.FinalPrompt)

	// Submit async task to the platform
	var genResp *image.GenTaskResponse
	var genErr error

	if profile.ReferenceImageURL != "" {
		genResp, genErr = m.imageClient.EditImageTask(taskCtx, apiBody)
	} else {
		genResp, genErr = m.imageClient.CreateImageTask(taskCtx, apiBody)
	}

	if genErr != nil {
		errMsg := genErr.Error()
		log.Printf("[task %s] submit failed: %s", taskID, errMsg)
		m.taskRepo.UpdateFailure(taskCtx, taskID, errMsg)
		m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "failed", ErrorMessage: errMsg})
		return
	}

	// Handle chat completions synchronous response
	if len(genResp.Choices) > 0 {
		content := genResp.Choices[0].Message.Content
		log.Printf("[task %s] chat completions response: %s", taskID, content)
		// Content may be an image URL or contain image URL
		imageURL := extractImageURL(content)
		if imageURL == "" {
			imageURL = content // use content as-is
		}
		m.taskRepo.UpdateResult(taskCtx, taskID, imageURL)
		m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "done", ResultImageURL: imageURL, Progress: 100})
		log.Printf("[task %s] completed via chat completions", taskID)
		return
	}

	// Handle task-based async response
	var platformTaskID string
	if len(genResp.Tasks) > 0 {
		platformTaskID = genResp.Tasks[0].ID
	} else if len(genResp.Data) > 0 {
		imageURL := genResp.Data[0].URL
		m.taskRepo.UpdateResult(taskCtx, taskID, imageURL)
		m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "done", ResultImageURL: imageURL, Progress: 100})
		log.Printf("[task %s] completed synchronously", taskID)
		return
	}

	if platformTaskID == "" {
		m.taskRepo.UpdateFailure(taskCtx, taskID, "unrecognized platform response")
		m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "failed", ErrorMessage: "unrecognized platform response"})
		return
	}

	log.Printf("[task %s] platform task id: %s", taskID, platformTaskID)
	m.pollTaskResult(taskCtx, taskID, platformTaskID)
}

func extractImageURL(content string) string {
	start := strings.Index(content, "http")
	if start == -1 {
		return ""
	}
	end := strings.Index(content[start:], " ")
	if end == -1 {
		return content[start:]
	}
	return content[start : start+end]
}

func (m *Manager) pollTaskResult(ctx context.Context, taskID, platformTaskID string) {
	pollInterval := 2 * time.Second
	lastProgress := 0

	for {
		select {
		case <-ctx.Done():
			m.taskRepo.UpdateFailure(context.Background(), taskID, "task timed out")
			m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "failed", ErrorMessage: "task timed out"})
			return
		case <-time.After(pollInterval):
		}

		result, err := m.imageClient.GetTaskResult(ctx, platformTaskID)
		if err != nil {
			log.Printf("[task %s] poll error: %v", taskID, err)
			continue
		}

		switch result.Status {
		case image.TaskStatusNotStart, image.TaskStatusInProgress:
			if result.Progress > lastProgress {
				lastProgress = result.Progress
				m.wsHub.PushUpdate(taskID, ws.TaskUpdate{
					TaskID:   taskID,
					Status:   "processing",
					Progress: result.Progress,
				})
				log.Printf("[task %s] progress %d%%", taskID, result.Progress)
			}

		case image.TaskStatusSuccess:
			if len(result.Data) > 0 {
				imageURL := result.Data[0].URL
				m.taskRepo.UpdateResult(context.Background(), taskID, imageURL)
				m.wsHub.PushUpdate(taskID, ws.TaskUpdate{
					TaskID:         taskID,
					Status:         "done",
					ResultImageURL: imageURL,
					Progress:       100,
				})
				log.Printf("[task %s] completed successfully", taskID)
				return
			}
			m.taskRepo.UpdateFailure(context.Background(), taskID, "platform returned success but no image data")
			m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "failed", ErrorMessage: "no image data"})
			return

		case image.TaskStatusFailure:
			errMsg := result.FailReason
			if errMsg == "" {
				errMsg = "platform task failed"
			}
			m.taskRepo.UpdateFailure(context.Background(), taskID, errMsg)
			m.wsHub.PushUpdate(taskID, ws.TaskUpdate{TaskID: taskID, Status: "failed", ErrorMessage: errMsg})
			log.Printf("[task %s] platform task failed: %s", taskID, errMsg)
			return
		}
	}
}

func (m *Manager) Close() {
	if m.rabbitConn != nil {
		m.rabbitConn.Close()
	}
}

func (m *Manager) TaskContext(timeoutSec int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
}
