package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"
)

const (
	jobDone       = "COMPLETED"
	jobQueued     = "IN_QUEUE"
	jobInProgress = "IN_PROGRESS"
)

const (
	runPath     = "run"
	runSyncPath = "runsync"
	statusPath  = "status"
	cancelPath  = "cancel"
)

type Input struct {
	Prompt string `json:"prompt"`
	System string `json:"-"`
}

type Output struct {
	Done               bool   `json:"done"`
	Response           string `json:"response"`
	Model              string `json:"model"`
	CreatedAt          string `json:"-"`
	EvalCount          int    `json:"-"`
	EvalDuration       int    `json:"-"`
	LoadDuration       int    `json:"-"`
	PromptEvalCount    int    `json:"-"`
	PromptEvalDuration int    `json:"-"`
	TotalDuration      int    `json:"-"`
	Context            []int  `json:"-"`
}

type Job struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	Input         Input  `json:"input"`
	Output        Output `json:"output"`
	DelayTime     int    `json:"-"`
	ExecutionTime int    `json:"-"`
}

type client struct {
	client http.Client
	apiKey string
	url    string
}

type config struct {
	apiKey string
	url    string
}

var version string

func main() {
	r := http.NewServeMux()

	c := http.Client{}
	c.Timeout = time.Second * 3
	conf := loadConfig()

	client := &client{
		client: c,
		apiKey: conf.apiKey,
		url:    conf.url,
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger = logger.With("version", version)

	r.HandleFunc("POST /question", handleQuestion(logger, client))

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)

	go func() {
		if err := http.ListenAndServe(":8080", r); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}
			logger.With("error", err).Error("http server error")
		}
	}()

	logger.Info("server started")
	<-interruptChan
	logger.Info("server stopped")
}

func loadConfig() *config {
	return &config{
		apiKey: os.Getenv("BOT_API_KEY"),
		url:    os.Getenv("BOT_URL"),
	}
}

func handleQuestion(logger *slog.Logger, client *client) http.HandlerFunc {
	logger = logger.With("handler", "question")
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := client.askChatBot(r.Body)
		if err != nil {
			if err := encode[string](w, http.StatusInternalServerError, `{"error": "failed to decode body"}`); err != nil {
				logger.With("error", err).Error("encoding failed")
			}
			return
		}
		if job.Output.Done {
			if err := encode[*Job](w, http.StatusOK, job); err != nil {
				logger.With("error", err).Error("encoding failed")
			}
			return
		}
		job, err = client.pollChatBot(job)
		if err != nil {
			if err := encode[string](w, http.StatusInternalServerError, `{"error": "client unresponsive"}`); err != nil {
				logger.With("error", err).Error("encoding failed")
			}
		}
		if err := encode[*Job](w, http.StatusOK, job); err != nil {
			logger.With("error", err).Error("encoding failed")
		}
	}
}

func (c *client) askChatBot(body io.Reader) (job *Job, err error) {
	u, err := url.JoinPath(c.url, runPath)
	if err != nil {
		return nil, fmt.Errorf("url join path: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http request failed, status code %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(job); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}

	return job, nil
}

func (c *client) pollChatBot(pending *Job) (job *Job, err error) {
	if job.Output.Done {
		return job, nil
	}

	u, err := url.JoinPath(c.url, statusPath, pending.ID)
	if err != nil {
		return nil, fmt.Errorf("url join path: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Add("Content-Type", "application/json")

	do := func(c http.Client, r *http.Request) (*Job, error) {
		resp, err := c.Do(r)
		if err != nil {
			return nil, fmt.Errorf("do http request: %w", err)
		}
		defer resp.Body.Close()
		var j *Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			return nil, fmt.Errorf("json decode: %w", err)
		}
		return j, nil
	}

	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		job, err := do(c.client, req)
		if err != nil {
			return nil, err
		}
		if job.Output.Done {
			return job, nil
		}
	}

	return nil, fmt.Errorf("timeout exceeded")
}

func encode[T any](w http.ResponseWriter, status int, v T) (err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func decode[T any](r *http.Request) (v T, err error) {
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}
