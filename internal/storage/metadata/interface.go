package metadata

import (
	"context"

	"GoFaas/pkg/types"
)

// FunctionRepository defines function storage operations
type FunctionRepository interface {
	Create(ctx context.Context, fn *types.Function) error
	GetByID(ctx context.Context, id string) (*types.Function, error)
	GetByName(ctx context.Context, name, version string) (*types.Function, error)
	Update(ctx context.Context, fn *types.Function) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter FunctionFilter) ([]*types.Function, error)
}

// InvocationRepository defines invocation storage operations
type InvocationRepository interface {
	CreateInvocation(ctx context.Context, inv *types.Invocation) error
	GetInvocationByID(ctx context.Context, id string) (*types.Invocation, error)
	UpdateInvocation(ctx context.Context, inv *types.Invocation) error
	ListInvocations(ctx context.Context, filter InvocationFilter) ([]*types.Invocation, error)
}

// FunctionFilter represents function query filters
type FunctionFilter struct {
	Runtime *types.RuntimeType
	Limit   int
	Offset  int
}

// InvocationFilter represents invocation query filters
type InvocationFilter struct {
	FunctionID *string
	Status     *types.ExecutionStatus
	Limit      int
	Offset     int
}
