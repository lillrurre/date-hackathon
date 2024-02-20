package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

	client := runpod.NewClient(logger, conf.Url, conf.ApiKey, conf.RequestTimeout)

	r := http.NewServeMux()
	r.HandleFunc("POST /job", handleJob(logger, client))
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
		job, err := client.RunRequest(r.Body)
		if err != nil {
			logger.With("error", err).Error("ask client")
			_ = encode(w, http.StatusInternalServerError, BadResponse{Message: "failed to decode body"})
			return
		}
		if job.Output.Done {
			_ = encode(w, http.StatusOK, job)
			return
		}
		job, err = client.StatusRequest(job)
		if err != nil {
			logger.With("error", err).Error("poll client")
			_ = encode(w, http.StatusInternalServerError, BadResponse{Message: "client unresponsive"})
			return
		}
		_ = encode[*runpod.Job](w, http.StatusOK, job)
	}
}

func encode[T any](w http.ResponseWriter, status int, v T) (err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}
