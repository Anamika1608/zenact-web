package api

import (
	"encoding/json"
	"net/http"

	"github.com/anamika/zenact-web/server/agent"
	"github.com/anamika/zenact-web/server/models"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	agent *agent.Agent
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Prompt == "" {
		http.Error(w, `{"error":"prompt is required"}`, http.StatusBadRequest)
		return
	}

	taskID := h.agent.StartTask(req.Prompt)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.CreateTaskResponse{
		TaskID: taskID,
		Status: string(models.TaskStatusPending),
	})
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	task, ok := h.agent.GetTask(taskID)
	if !ok {
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	// Strip screenshots from REST response (they are large; use WebSocket for live view)
	for i := range task.Steps {
		task.Steps[i].Screenshot = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
