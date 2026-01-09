package controller

import (
	"net/http"

	"github.com/gorilla/mux"

	"GoFaas/internal/api/common"
	"GoFaas/internal/core/function"
	"GoFaas/internal/observability/logging"
	"GoFaas/internal/storage/metadata"
	"GoFaas/pkg/errors"
	"GoFaas/pkg/types"
)

// FunctionHandler handles function management requests
type FunctionHandler struct {
	service *function.Service
	logger  logging.Logger
}

// NewFunctionHandler creates a new function handler
func NewFunctionHandler(service *function.Service, logger logging.Logger) *FunctionHandler {
	return &FunctionHandler{
		service: service,
		logger:  logger,
	}
}

// CreateFunction handles function creation
func (h *FunctionHandler) CreateFunction(w http.ResponseWriter, r *http.Request) {
	var req function.CreateFunctionRequest
	if err := common.ParseJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	fn, err := h.service.CreateFunction(r.Context(), req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusCreated, fn)
}

// GetFunction handles function retrieval by ID
func (h *FunctionHandler) GetFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	fn, err := h.service.GetFunction(r.Context(), id)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, fn)
}

// UpdateFunction handles function updates
func (h *FunctionHandler) UpdateFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req function.UpdateFunctionRequest
	if err := common.ParseJSON(r, &req); err != nil {
		common.WriteError(w, err)
		return
	}

	fn, err := h.service.UpdateFunction(r.Context(), id, req)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, fn)
}

// DeleteFunction handles function deletion
func (h *FunctionHandler) DeleteFunction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.service.DeleteFunction(r.Context(), id); err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Function deleted successfully",
	})
}

// ListFunctions handles function listing
func (h *FunctionHandler) ListFunctions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	filter := metadata.FunctionFilter{
		Limit:  10,
		Offset: 0,
	}

	// Parse runtime filter
	if runtimeStr := query.Get("runtime"); runtimeStr != "" {
		runtime := types.RuntimeType(runtimeStr)
		if runtime.IsValid() {
			filter.Runtime = &runtime
		} else {
			common.WriteError(w, errors.ValidationError("invalid runtime"))
			return
		}
	}

	functions, err := h.service.ListFunctions(r.Context(), filter)
	if err != nil {
		common.WriteError(w, err)
		return
	}

	common.WriteJSON(w, http.StatusOK, functions)
}
