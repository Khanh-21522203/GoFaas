package controller

import (
	"net/http"

	"github.com/gorilla/mux"

	"GoFaas/internal/api/common"
	"GoFaas/internal/core/invocation"
	"GoFaas/internal/observability/logging"
	"GoFaas/internal/storage/metadata"
)

// InvocationHandler handles function invocation requests
type InvocationHandler struct {
	service *invocation.Service
	logger  logging.Logger
}

// NewInvocationHandler creates a new invocation handler
func NewInvocationHandler(service *invocation.Service, logger logging.Logger) *InvocationHandler {
	return &InvocationHandler{
		service: service,
		logger:  logger,
	}
}

// InvokeFunction handles function invocation
func (h *InvocationHandler) InvokeFunction(w http.ResponseWriter, r *http.Request) {
	var req invocation.InvocationRequest
	if err := common.ParseJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	handle, err := h.service.InvokeAsync(r.Context(), req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusAccepted, handle)
}

// GetInvocationResult handles invocation result retrieval
func (h *InvocationHandler) GetInvocationResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invocationID := vars["id"]

	inv, err := h.service.GetResult(r.Context(), invocationID)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, inv)
}

// ListInvocations handles invocation listing
func (h *InvocationHandler) ListInvocations(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	filter := metadata.InvocationFilter{
		Limit:  10,
		Offset: 0,
	}

	// Parse function_id filter
	if functionID := query.Get("function_id"); functionID != "" {
		filter.FunctionID = &functionID
	}

	invocations, err := h.service.ListInvocations(r.Context(), filter)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, invocations)
}
