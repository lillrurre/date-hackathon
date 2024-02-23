package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/lillrurre/date-hackathon-backend/config"
	"github.com/lillrurre/date-hackathon-backend/runpod"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
)

type BadResponse struct {
	Message string `json:"error"`
}

var version string

func main() {

	conf := config.LoadConfig()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger = logger.With("version", version)

	client := runpod.NewClient(logger, conf.Url, conf.ApiKey, conf.SystemPrompt, conf.RequestTimeout)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders: []string{"Link"},
		MaxAge:         300,
		Debug:          true,
	}))
	r.Handle("POST /job", handleJob(logger, client))

	go func() {
		if err := http.ListenAndServe(":8080", r); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			logger.With("error", err).Error("http server error")
		}
	}()

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	logger.Info("server started")
	<-interruptChan
	logger.Info("server stopped")
}

func handleJob(logger *slog.Logger, client *runpod.Client) http.HandlerFunc {
	logger = logger.With("handler", "job")
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("request received")
		var j *runpod.Job
		if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
			logger.With("error", err).Error("failed to decode json")
		}
		job, err := client.RunSyncRequest(j)
		if err != nil {
			if job != nil {
				_, _ = client.CancelRequest(job)
			}
			logger.With("error", err).Error("ask client")
			_ = encode(w, http.StatusInternalServerError, BadResponse{Message: "failed to decode body"})
			return
		}
		_ = encode(w, http.StatusOK, job)
	}
}

func encode[T any](w http.ResponseWriter, status int, v T) (err error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
