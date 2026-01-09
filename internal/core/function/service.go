package function

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"

	"GoFaas/internal/observability/logging"
	"GoFaas/internal/storage/function"
	"GoFaas/internal/storage/metadata"
	"GoFaas/pkg/errors"
	"GoFaas/pkg/types"
	"GoFaas/pkg/utils"
)

// Service implements function management business logic
type Service struct {
	repo    metadata.FunctionRepository
	storage function.Storage
	logger  logging.Logger
}

// NewService creates a new function service
func NewService(repo metadata.FunctionRepository, storage function.Storage, logger logging.Logger) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		logger:  logger,
	}
}

// CreateFunction creates a new function
func (s *Service) CreateFunction(ctx context.Context, req CreateFunctionRequest) (*types.Function, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Decode function code
	codeBytes, err := base64.StdEncoding.DecodeString(req.Code)
	if err != nil {
		return nil, errors.ValidationError(fmt.Sprintf("invalid base64 code: %v", err))
	}

	// Generate function ID
	functionID := uuid.New().String()

	// Calculate checksum
	checksum := utils.SHA256Hash(codeBytes)

	// Store function code
	codeLocation, err := s.storage.Store(ctx, functionID, codeBytes)
	if err != nil {
		return nil, errors.InternalError(fmt.Sprintf("failed to store function code: %v", err))
	}

	// Create function entity
	fn := &types.Function{
		ID:      functionID,
		Name:    req.Name,
		Version: req.Version,
		Runtime: req.Runtime,
		Handler: req.Handler,
		Code: types.FunctionCode{
			Source:     codeLocation,
			SourceType: "local",
			Checksum:   checksum,
			Size:       int64(len(codeBytes)),
		},
		Config: types.FunctionConfig{
			Timeout:     req.Timeout,
			Memory:      req.Memory,
			Environment: req.Environment,
			Concurrency: req.Concurrency,
		},
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to database
	if err := s.repo.Create(ctx, fn); err != nil {
		// Cleanup stored code on failure
		s.storage.Delete(ctx, codeLocation)
		return nil, err
	}

	s.logger.Info("Function created successfully",
		logging.F("function_id", functionID),
		logging.F("name", req.Name),
		logging.F("version", req.Version),
		logging.F("runtime", req.Runtime),
	)

	return fn, nil
}

// GetFunction retrieves a function by ID
func (s *Service) GetFunction(ctx context.Context, id string) (*types.Function, error) {
	fn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return fn, nil
}

// GetFunctionByName retrieves a function by name and version
func (s *Service) GetFunctionByName(ctx context.Context, name, version string) (*types.Function, error) {
	fn, err := s.repo.GetByName(ctx, name, version)
	if err != nil {
		return nil, err
	}

	return fn, nil
}

// UpdateFunction updates an existing function
func (s *Service) UpdateFunction(ctx context.Context, id string, req UpdateFunctionRequest) (*types.Function, error) {
	// Get existing function
	fn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Handler != nil {
		fn.Handler = *req.Handler
	}
	if req.Timeout != nil {
		if *req.Timeout <= 0 {
			return nil, errors.ValidationError("timeout must be positive")
		}
		fn.Config.Timeout = *req.Timeout
	}
	if req.Memory != nil {
		if *req.Memory <= 0 {
			return nil, errors.ValidationError("memory must be positive")
		}
		fn.Config.Memory = *req.Memory
	}
	if req.Environment != nil {
		fn.Config.Environment = req.Environment
	}
	if req.Concurrency != nil {
		if *req.Concurrency <= 0 {
			return nil, errors.ValidationError("concurrency must be positive")
		}
		fn.Config.Concurrency = *req.Concurrency
	}

	// Update code if provided
	if req.Code != nil {
		codeBytes, err := base64.StdEncoding.DecodeString(*req.Code)
		if err != nil {
			return nil, errors.ValidationError(fmt.Sprintf("invalid base64 code: %v", err))
		}

		// Store new code
		codeLocation, err := s.storage.Store(ctx, id, codeBytes)
		if err != nil {
			return nil, errors.InternalError(fmt.Sprintf("failed to store function code: %v", err))
		}

		// Update code metadata
		fn.Code.Source = codeLocation
		fn.Code.Checksum = utils.SHA256Hash(codeBytes)
		fn.Code.Size = int64(len(codeBytes))
	}

	fn.UpdatedAt = time.Now()

	// Save changes
	if err := s.repo.Update(ctx, fn); err != nil {
		return nil, err
	}

	s.logger.Info("Function updated successfully",
		logging.F("function_id", id),
		logging.F("name", fn.Name),
	)

	return fn, nil
}

// DeleteFunction deletes a function
func (s *Service) DeleteFunction(ctx context.Context, id string) error {
	// Get function to retrieve code location
	fn, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete from database first
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Delete stored code (best effort)
	if err := s.storage.Delete(ctx, fn.Code.Source); err != nil {
		s.logger.Error("Failed to delete function code",
			logging.F("function_id", id),
			logging.F("code_location", fn.Code.Source),
			logging.F("error", err),
		)
	}

	s.logger.Info("Function deleted successfully",
		logging.F("function_id", id),
		logging.F("name", fn.Name),
	)

	return nil
}

// ListFunctions lists functions with filters
func (s *Service) ListFunctions(ctx context.Context, filter metadata.FunctionFilter) ([]*types.Function, error) {
	functions, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return functions, nil
}

// validateCreateRequest validates function creation request
func (s *Service) validateCreateRequest(req CreateFunctionRequest) error {
	if err := utils.ValidateFunctionName(req.Name); err != nil {
		return errors.ValidationError(err.Error())
	}

	if req.Version == "" {
		return errors.ValidationError("version is required")
	}

	if !req.Runtime.IsValid() {
		return errors.ValidationError(fmt.Sprintf("unsupported runtime: %s", req.Runtime))
	}

	if req.Handler == "" {
		return errors.ValidationError("handler is required")
	}

	if req.Code == "" {
		return errors.ValidationError("function code is required")
	}

	if req.Timeout <= 0 {
		return errors.ValidationError("timeout must be positive")
	}

	if req.Memory <= 0 {
		return errors.ValidationError("memory must be positive")
	}

	if req.Concurrency <= 0 {
		return errors.ValidationError("concurrency must be positive")
	}

	return nil
}
