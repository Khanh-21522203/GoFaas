package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"GoFaas/pkg/errors"
	"GoFaas/pkg/types"
)

// PostgresRepository implements metadata repositories using PostgreSQL
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Create implements FunctionRepository.Create
func (r *PostgresRepository) Create(ctx context.Context, fn *types.Function) error {
	query := `
		INSERT INTO functions (
			id, name, version, runtime, handler, code_source, code_source_type,
			code_checksum, code_size, timeout_seconds, memory_mb, max_concurrency,
			environment, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	envJSON, _ := json.Marshal(fn.Config.Environment)
	metaJSON, _ := json.Marshal(fn.Metadata)

	_, err := r.db.ExecContext(ctx, query,
		fn.ID, fn.Name, fn.Version, fn.Runtime, fn.Handler,
		fn.Code.Source, fn.Code.SourceType, fn.Code.Checksum, fn.Code.Size,
		int(fn.Config.Timeout.Seconds()), fn.Config.Memory, fn.Config.Concurrency,
		envJSON, metaJSON, fn.CreatedAt, fn.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return errors.Conflict(fmt.Sprintf("function %s version %s already exists", fn.Name, fn.Version))
		}
		return errors.InternalError(fmt.Sprintf("failed to create function: %v", err))
	}

	return nil
}

// GetByID implements FunctionRepository.GetByID
func (r *PostgresRepository) GetByID(ctx context.Context, id string) (*types.Function, error) {
	query := `
		SELECT id, name, version, runtime, handler, code_source, code_source_type,
		       code_checksum, code_size, timeout_seconds, memory_mb, max_concurrency,
		       environment, metadata, created_at, updated_at
		FROM functions WHERE id = $1`

	var fn types.Function
	var envJSON, metaJSON []byte
	var timeoutSeconds int

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&fn.ID, &fn.Name, &fn.Version, &fn.Runtime, &fn.Handler,
		&fn.Code.Source, &fn.Code.SourceType, &fn.Code.Checksum, &fn.Code.Size,
		&timeoutSeconds, &fn.Config.Memory, &fn.Config.Concurrency,
		&envJSON, &metaJSON, &fn.CreatedAt, &fn.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("function", id)
		}
		return nil, errors.InternalError(fmt.Sprintf("failed to get function: %v", err))
	}

	fn.Config.Timeout = time.Duration(timeoutSeconds) * time.Second
	json.Unmarshal(envJSON, &fn.Config.Environment)
	json.Unmarshal(metaJSON, &fn.Metadata)

	return &fn, nil
}

// GetByName implements FunctionRepository.GetByName
func (r *PostgresRepository) GetByName(ctx context.Context, name, version string) (*types.Function, error) {
	query := `
		SELECT id, name, version, runtime, handler, code_source, code_source_type,
		       code_checksum, code_size, timeout_seconds, memory_mb, max_concurrency,
		       environment, metadata, created_at, updated_at
		FROM functions WHERE name = $1 AND version = $2`

	var fn types.Function
	var envJSON, metaJSON []byte
	var timeoutSeconds int

	err := r.db.QueryRowContext(ctx, query, name, version).Scan(
		&fn.ID, &fn.Name, &fn.Version, &fn.Runtime, &fn.Handler,
		&fn.Code.Source, &fn.Code.SourceType, &fn.Code.Checksum, &fn.Code.Size,
		&timeoutSeconds, &fn.Config.Memory, &fn.Config.Concurrency,
		&envJSON, &metaJSON, &fn.CreatedAt, &fn.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("function", fmt.Sprintf("%s:%s", name, version))
		}
		return nil, errors.InternalError(fmt.Sprintf("failed to get function: %v", err))
	}

	fn.Config.Timeout = time.Duration(timeoutSeconds) * time.Second
	json.Unmarshal(envJSON, &fn.Config.Environment)
	json.Unmarshal(metaJSON, &fn.Metadata)

	return &fn, nil
}

// Update implements FunctionRepository.Update
func (r *PostgresRepository) Update(ctx context.Context, fn *types.Function) error {
	query := `
		UPDATE functions SET
			handler = $2, code_source = $3, code_source_type = $4,
			code_checksum = $5, code_size = $6, timeout_seconds = $7,
			memory_mb = $8, max_concurrency = $9, environment = $10,
			metadata = $11, updated_at = $12
		WHERE id = $1`

	envJSON, _ := json.Marshal(fn.Config.Environment)
	metaJSON, _ := json.Marshal(fn.Metadata)

	result, err := r.db.ExecContext(ctx, query,
		fn.ID, fn.Handler, fn.Code.Source, fn.Code.SourceType,
		fn.Code.Checksum, fn.Code.Size, int(fn.Config.Timeout.Seconds()),
		fn.Config.Memory, fn.Config.Concurrency, envJSON, metaJSON, fn.UpdatedAt,
	)

	if err != nil {
		return errors.InternalError(fmt.Sprintf("failed to update function: %v", err))
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound("function", fn.ID)
	}

	return nil
}

// Delete implements FunctionRepository.Delete
func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM functions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errors.InternalError(fmt.Sprintf("failed to delete function: %v", err))
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound("function", id)
	}

	return nil
}

// List implements FunctionRepository.List
func (r *PostgresRepository) List(ctx context.Context, filter FunctionFilter) ([]*types.Function, error) {
	query := `
		SELECT id, name, version, runtime, handler, code_source, code_source_type,
		       code_checksum, code_size, timeout_seconds, memory_mb, max_concurrency,
		       environment, metadata, created_at, updated_at
		FROM functions
		WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	if filter.Runtime != nil {
		query += fmt.Sprintf(" AND runtime = $%d", argPos)
		args = append(args, *filter.Runtime)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.InternalError(fmt.Sprintf("failed to list functions: %v", err))
	}
	defer rows.Close()

	functions := make([]*types.Function, 0)
	for rows.Next() {
		var fn types.Function
		var envJSON, metaJSON []byte
		var timeoutSeconds int

		err := rows.Scan(
			&fn.ID, &fn.Name, &fn.Version, &fn.Runtime, &fn.Handler,
			&fn.Code.Source, &fn.Code.SourceType, &fn.Code.Checksum, &fn.Code.Size,
			&timeoutSeconds, &fn.Config.Memory, &fn.Config.Concurrency,
			&envJSON, &metaJSON, &fn.CreatedAt, &fn.UpdatedAt,
		)
		if err != nil {
			return nil, errors.InternalError(fmt.Sprintf("failed to scan function: %v", err))
		}

		fn.Config.Timeout = time.Duration(timeoutSeconds) * time.Second
		json.Unmarshal(envJSON, &fn.Config.Environment)
		json.Unmarshal(metaJSON, &fn.Metadata)

		functions = append(functions, &fn)
	}

	return functions, nil
}

// CreateInvocation creates a new invocation record
func (r *PostgresRepository) CreateInvocation(ctx context.Context, inv *types.Invocation) error {
	query := `
		INSERT INTO invocations (
			id, function_id, payload, headers, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)`

	payloadJSON, _ := json.Marshal(inv.Payload)
	headersJSON, _ := json.Marshal(inv.Headers)

	_, err := r.db.ExecContext(ctx, query,
		inv.ID, inv.FunctionID, payloadJSON, headersJSON, inv.Status, inv.CreatedAt,
	)

	if err != nil {
		return errors.InternalError(fmt.Sprintf("failed to create invocation: %v", err))
	}

	return nil
}

// GetInvocationByID retrieves an invocation by ID
func (r *PostgresRepository) GetInvocationByID(ctx context.Context, id string) (*types.Invocation, error) {
	query := `
		SELECT id, function_id, payload, headers, status, result,
		       error_type, error_message, error_stack,
		       duration_ns, cpu_time_ns, memory_peak, network_in, network_out,
		       created_at, started_at, completed_at
		FROM invocations WHERE id = $1`

	var inv types.Invocation
	var payloadJSON, headersJSON, resultJSON []byte
	var errorType, errorMessage, errorStack sql.NullString
	var durationNs, cpuTimeNs, memoryPeak, networkIn, networkOut sql.NullInt64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID, &inv.FunctionID, &payloadJSON, &headersJSON, &inv.Status, &resultJSON,
		&errorType, &errorMessage, &errorStack,
		&durationNs, &cpuTimeNs, &memoryPeak, &networkIn, &networkOut,
		&inv.CreatedAt, &inv.StartedAt, &inv.CompletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("invocation", id)
		}
		return nil, errors.InternalError(fmt.Sprintf("failed to get invocation: %v", err))
	}

	json.Unmarshal(payloadJSON, &inv.Payload)
	json.Unmarshal(headersJSON, &inv.Headers)
	if len(resultJSON) > 0 {
		json.Unmarshal(resultJSON, &inv.Result)
	}

	if errorType.Valid {
		inv.Error = &types.ExecutionError{
			Type:    errorType.String,
			Message: errorMessage.String,
			Stack:   errorStack.String,
		}
	}

	if durationNs.Valid {
		inv.Metrics = &types.ExecutionMetrics{
			Duration:   time.Duration(durationNs.Int64),
			CPUTime:    time.Duration(cpuTimeNs.Int64),
			MemoryPeak: memoryPeak.Int64,
			NetworkIn:  networkIn.Int64,
			NetworkOut: networkOut.Int64,
		}
	}

	return &inv, nil
}

// UpdateInvocation updates an invocation record
func (r *PostgresRepository) UpdateInvocation(ctx context.Context, inv *types.Invocation) error {
	query := `
		UPDATE invocations SET
			status = $2, result = $3,
			error_type = $4, error_message = $5, error_stack = $6,
			duration_ns = $7, cpu_time_ns = $8, memory_peak = $9,
			network_in = $10, network_out = $11,
			started_at = $12, completed_at = $13
		WHERE id = $1`

	var resultJSON []byte
	if inv.Result != nil {
		resultJSON, _ = json.Marshal(inv.Result)
	}

	var errorType, errorMessage, errorStack sql.NullString
	if inv.Error != nil {
		errorType = sql.NullString{String: inv.Error.Type, Valid: true}
		errorMessage = sql.NullString{String: inv.Error.Message, Valid: true}
		errorStack = sql.NullString{String: inv.Error.Stack, Valid: true}
	}

	var durationNs, cpuTimeNs, memoryPeak, networkIn, networkOut sql.NullInt64
	if inv.Metrics != nil {
		durationNs = sql.NullInt64{Int64: int64(inv.Metrics.Duration), Valid: true}
		cpuTimeNs = sql.NullInt64{Int64: int64(inv.Metrics.CPUTime), Valid: true}
		memoryPeak = sql.NullInt64{Int64: inv.Metrics.MemoryPeak, Valid: true}
		networkIn = sql.NullInt64{Int64: inv.Metrics.NetworkIn, Valid: true}
		networkOut = sql.NullInt64{Int64: inv.Metrics.NetworkOut, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		inv.ID, inv.Status, resultJSON,
		errorType, errorMessage, errorStack,
		durationNs, cpuTimeNs, memoryPeak, networkIn, networkOut,
		inv.StartedAt, inv.CompletedAt,
	)

	if err != nil {
		return errors.InternalError(fmt.Sprintf("failed to update invocation: %v", err))
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound("invocation", inv.ID)
	}

	return nil
}

// ListInvocations lists invocations with filters
func (r *PostgresRepository) ListInvocations(ctx context.Context, filter InvocationFilter) ([]*types.Invocation, error) {
	query := `
		SELECT id, function_id, payload, headers, status, result,
		       error_type, error_message, error_stack,
		       duration_ns, cpu_time_ns, memory_peak, network_in, network_out,
		       created_at, started_at, completed_at
		FROM invocations
		WHERE 1=1`

	args := []interface{}{}
	argPos := 1

	if filter.FunctionID != nil {
		query += fmt.Sprintf(" AND function_id = $%d", argPos)
		args = append(args, *filter.FunctionID)
		argPos++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *filter.Status)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.InternalError(fmt.Sprintf("failed to list invocations: %v", err))
	}
	defer rows.Close()

	invocations := make([]*types.Invocation, 0)
	for rows.Next() {
		var inv types.Invocation
		var payloadJSON, headersJSON, resultJSON []byte
		var errorType, errorMessage, errorStack sql.NullString
		var durationNs, cpuTimeNs, memoryPeak, networkIn, networkOut sql.NullInt64

		err := rows.Scan(
			&inv.ID, &inv.FunctionID, &payloadJSON, &headersJSON, &inv.Status, &resultJSON,
			&errorType, &errorMessage, &errorStack,
			&durationNs, &cpuTimeNs, &memoryPeak, &networkIn, &networkOut,
			&inv.CreatedAt, &inv.StartedAt, &inv.CompletedAt,
		)
		if err != nil {
			return nil, errors.InternalError(fmt.Sprintf("failed to scan invocation: %v", err))
		}

		json.Unmarshal(payloadJSON, &inv.Payload)
		json.Unmarshal(headersJSON, &inv.Headers)
		if len(resultJSON) > 0 {
			json.Unmarshal(resultJSON, &inv.Result)
		}

		if errorType.Valid {
			inv.Error = &types.ExecutionError{
				Type:    errorType.String,
				Message: errorMessage.String,
				Stack:   errorStack.String,
			}
		}

		if durationNs.Valid {
			inv.Metrics = &types.ExecutionMetrics{
				Duration:   time.Duration(durationNs.Int64),
				CPUTime:    time.Duration(cpuTimeNs.Int64),
				MemoryPeak: memoryPeak.Int64,
				NetworkIn:  networkIn.Int64,
				NetworkOut: networkOut.Int64,
			}
		}

		invocations = append(invocations, &inv)
	}

	return invocations, nil
}
