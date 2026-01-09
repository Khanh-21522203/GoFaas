package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"GoFaas/pkg/storage"
)

// Struct for function metadata
type Function struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"` // e.g., "go", "python"
	Code    string `json:"code"`    // Base64 encoded source code
}

// Upload function
func HandleFunctions(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var fn Function
		body, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(body, &fn); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Save function code to storage
		err := storage.SaveFunction(fn.Name, fn.Code)
		if err != nil {
			http.Error(w, "Failed to save function", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Function uploaded successfully"))
	}
}

// Invoke function
func HandleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Implementation for invoking functions can be added here.
	}
}
