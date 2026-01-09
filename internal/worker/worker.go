package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"GoFaas/internal/core/invocation"
	"GoFaas/internal/messaging"
	"GoFaas/internal/observability/logging"
	"GoFaas/internal/storage/function"
	"GoFaas/internal/storage/metadata"
	"GoFaas/internal/worker/runtime"
	"GoFaas/pkg/types"
)

// Worker processes function execution requests from the queue
type Worker struct {
	id             string
	queue          messaging.Queue
	functionRepo   metadata.FunctionRepository
	invocationRepo metadata.InvocationRepository
	functionStore  function.Storage
	runtime        runtime.Runtime
	invocationSvc  *invocation.Service
	logger         logging.Logger
	stopCh         chan struct{}
}

// Config holds worker configuration
type Config struct {
	ID             string
	Queue          messaging.Queue
	FunctionRepo   metadata.FunctionRepository
	InvocationRepo metadata.InvocationRepository
	FunctionStore  function.Storage
	Runtime        runtime.Runtime
	InvocationSvc  *invocation.Service
	Logger         logging.Logger
}

// NewWorker creates a new worker
func NewWorker(cfg Config) *Worker {
	return &Worker{
		id:             cfg.ID,
		queue:          cfg.Queue,
		functionRepo:   cfg.FunctionRepo,
		invocationRepo: cfg.InvocationRepo,
		functionStore:  cfg.FunctionStore,
		runtime:        cfg.Runtime,
		invocationSvc:  cfg.InvocationSvc,
		logger:         cfg.Logger.WithFields(logging.F("worker_id", cfg.ID)),
		stopCh:         make(chan struct{}),
	}
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Worker starting")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker stopping due to context cancellation")
			return ctx.Err()
		case <-w.stopCh:
			w.logger.Info("Worker stopping")
			return nil
		default:
			if err := w.processNextMessage(ctx); err != nil {
				w.logger.Error("Failed to process message", logging.F("error", err))
				// Continue processing despite errors
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.stopCh)
}

// processNextMessage dequeues and processes a single message
func (w *Worker) processNextMessage(ctx context.Context) error {
	// Dequeue message with timeout
	msg, err := w.queue.Dequeue(ctx, invocation.ExecutionQueueName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dequeue message: %w", err)
	}

	// No message available
	if msg == nil {
		return nil
	}

	w.logger.Info("Processing execution request",
		logging.F("message_id", msg.ID),
		logging.F("attempts", msg.Attempts),
	)

	// Parse execution request
	var execReq invocation.ExecutionRequest
	if err := json.Unmarshal(msg.Payload, &execReq); err != nil {
		w.logger.Error("Failed to unmarshal execution request",
			logging.F("message_id", msg.ID),
			logging.F("error", err),
		)
		// Dead letter invalid messages
		w.queue.DeadLetter(ctx, msg, fmt.Sprintf("invalid payload: %v", err))
		return nil
	}

	// Execute function
	result, err := w.executeFunction(ctx, execReq)
	if err != nil {
		w.logger.Error("Failed to execute function",
			logging.F("invocation_id", execReq.InvocationID),
			logging.F("function_id", execReq.FunctionID),
			logging.F("error", err),
		)

		// Retry logic: if attempts < 3, nack and retry
		if msg.Attempts < 3 {
			w.queue.Nack(ctx, msg)
			return nil
		}

		// Max retries exceeded, dead letter
		w.queue.DeadLetter(ctx, msg, fmt.Sprintf("max retries exceeded: %v", err))
		return nil
	}

	// Update invocation result
	if err := w.invocationSvc.UpdateInvocationResult(ctx, execReq.InvocationID, *result); err != nil {
		w.logger.Error("Failed to update invocation result",
			logging.F("invocation_id", execReq.InvocationID),
			logging.F("error", err),
		)
		// Still ack the message to avoid reprocessing
	}

	// Acknowledge message
	if err := w.queue.Ack(ctx, msg); err != nil {
		w.logger.Error("Failed to acknowledge message",
			logging.F("message_id", msg.ID),
			logging.F("error", err),
		)
	}

	w.logger.Info("Execution completed",
		logging.F("invocation_id", execReq.InvocationID),
		logging.F("status", result.Status),
		logging.F("duration", result.Metrics.Duration),
	)

	return nil
}

// executeFunction executes a function
func (w *Worker) executeFunction(ctx context.Context, req invocation.ExecutionRequest) (*invocation.ExecutionResult, error) {
	// Update invocation status to running
	if err := w.invocationSvc.UpdateInvocationStatus(ctx, req.InvocationID, types.StatusRunning); err != nil {
		w.logger.Warn("Failed to update invocation status to running",
			logging.F("invocation_id", req.InvocationID),
			logging.F("error", err),
		)
	}

	// Get function metadata
	fn, err := w.functionRepo.GetByID(ctx, req.FunctionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	// Retrieve function code
	code, err := w.functionStore.Retrieve(ctx, fn.Code.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve function code: %w", err)
	}

	// Determine timeout
	timeout := fn.Config.Timeout
	if req.Timeout != nil {
		timeout = *req.Timeout
	}

	// Prepare execution spec
	spec := runtime.ExecutionSpec{
		FunctionID:  req.FunctionID,
		Code:        code,
		Runtime:     fn.Runtime,
		Handler:     fn.Handler,
		Payload:     req.Payload,
		Environment: fn.Config.Environment,
		Timeout:     timeout,
		Limits: runtime.ResourceLimits{
			MemoryBytes: int64(fn.Config.Memory) * 1024 * 1024, // Convert MB to bytes
			Timeout:     timeout,
		},
	}

	// Execute function
	runtimeResult, err := w.runtime.Execute(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("runtime execution failed: %w", err)
	}

	// Convert runtime result to invocation result
	result := &invocation.ExecutionResult{
		Status:  runtimeResult.Status,
		Result:  json.RawMessage(runtimeResult.Result),
		Error:   runtimeResult.Error,
		Metrics: &runtimeResult.Metrics,
	}

	return result, nil
}
