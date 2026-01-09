package invocation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"GoFaas/internal/messaging"
	"GoFaas/internal/observability/logging"
	"GoFaas/internal/storage/metadata"
	"GoFaas/pkg/errors"
	"GoFaas/pkg/types"
)

const (
	// ExecutionQueueName is the queue name for function executions
	ExecutionQueueName = "faas_executions"
)

// Service implements invocation business logic
type Service struct {
	functionRepo   metadata.FunctionRepository
	invocationRepo metadata.InvocationRepository
	queue          messaging.Queue
	logger         logging.Logger
}

// NewService creates a new invocation service
func NewService(
	functionRepo metadata.FunctionRepository,
	invocationRepo metadata.InvocationRepository,
	queue messaging.Queue,
	logger logging.Logger,
) *Service {
	return &Service{
		functionRepo:   functionRepo,
		invocationRepo: invocationRepo,
		queue:          queue,
		logger:         logger,
	}
}

// InvokeAsync invokes a function asynchronously
func (s *Service) InvokeAsync(ctx context.Context, req InvocationRequest) (*InvocationHandle, error) {
	// Validate function exists
	fn, err := s.functionRepo.GetByID(ctx, req.FunctionID)
	if err != nil {
		return nil, err
	}

	// Create invocation record
	invocationID := uuid.New().String()
	invocation := &types.Invocation{
		ID:         invocationID,
		FunctionID: req.FunctionID,
		Payload:    req.Payload,
		Headers:    req.Headers,
		Status:     types.StatusPending,
		CreatedAt:  time.Now(),
	}

	if err := s.invocationRepo.Create(ctx, invocation); err != nil {
		return nil, err
	}

	// Create execution request
	execReq := ExecutionRequest{
		InvocationID: invocationID,
		FunctionID:   req.FunctionID,
		Payload:      req.Payload,
		Headers:      req.Headers,
		Timeout:      req.Timeout,
	}

	// If no timeout specified, use function's default timeout
	if execReq.Timeout == nil {
		execReq.Timeout = &fn.Config.Timeout
	}

	// Enqueue execution request
	payload, err := json.Marshal(execReq)
	if err != nil {
		return nil, errors.InternalError(fmt.Sprintf("failed to marshal execution request: %v", err))
	}

	headers := map[string]string{
		"invocation_id": invocationID,
		"function_id":   req.FunctionID,
	}

	if err := s.queue.Enqueue(ctx, ExecutionQueueName, payload, headers); err != nil {
		return nil, errors.InternalError(fmt.Sprintf("failed to enqueue execution: %v", err))
	}

	s.logger.Info("Function invoked asynchronously",
		logging.F("invocation_id", invocationID),
		logging.F("function_id", req.FunctionID),
		logging.F("function_name", fn.Name),
	)

	return &InvocationHandle{
		InvocationID: invocationID,
		FunctionID:   req.FunctionID,
		Status:       types.StatusPending,
		CreatedAt:    invocation.CreatedAt,
	}, nil
}

// GetResult retrieves invocation result
func (s *Service) GetResult(ctx context.Context, invocationID string) (*types.Invocation, error) {
	invocation, err := s.invocationRepo.GetByID(ctx, invocationID)
	if err != nil {
		return nil, err
	}

	return invocation, nil
}

// ListInvocations lists invocations with filters
func (s *Service) ListInvocations(ctx context.Context, filter metadata.InvocationFilter) ([]*types.Invocation, error) {
	invocations, err := s.invocationRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return invocations, nil
}

// UpdateInvocationStatus updates invocation status (used by workers)
func (s *Service) UpdateInvocationStatus(ctx context.Context, invocationID string, status types.ExecutionStatus) error {
	invocation, err := s.invocationRepo.GetByID(ctx, invocationID)
	if err != nil {
		return err
	}

	invocation.Status = status

	now := time.Now()
	if status == types.StatusRunning && invocation.StartedAt == nil {
		invocation.StartedAt = &now
	}

	if status.IsTerminal() && invocation.CompletedAt == nil {
		invocation.CompletedAt = &now
	}

	return s.invocationRepo.Update(ctx, invocation)
}

// UpdateInvocationResult updates invocation result (used by workers)
func (s *Service) UpdateInvocationResult(ctx context.Context, invocationID string, result ExecutionResult) error {
	invocation, err := s.invocationRepo.GetByID(ctx, invocationID)
	if err != nil {
		return err
	}

	invocation.Status = result.Status
	invocation.Result = result.Result
	invocation.Error = result.Error
	invocation.Metrics = result.Metrics

	now := time.Now()
	if invocation.StartedAt == nil {
		invocation.StartedAt = &now
	}
	invocation.CompletedAt = &now

	return s.invocationRepo.Update(ctx, invocation)
}
