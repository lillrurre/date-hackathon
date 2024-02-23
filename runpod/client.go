package runpod

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	runPath     = "run"
	runSyncPath = "runsync"
	statusPath  = "status"
	cancelPath  = "cancel"
)

type Client struct {
	apiKey       string
	url          string
	systemPrompt string
	client       *http.Client
	logger       *slog.Logger
}

func NewClient(logger *slog.Logger, baseUrl, apiKey, systemPrompt string, requestTimeout time.Duration) *Client {
	c := &http.Client{}
	c.Timeout = requestTimeout

	return &Client{
		client:       c,
		apiKey:       apiKey,
		url:          baseUrl,
		systemPrompt: systemPrompt,
		logger:       logger.With("component", "client"),
	}
}

func (c *Client) RunRequest(body io.Reader) (job *Job, err error) {
	req, err := c.newRequest(body, http.MethodPost, runPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return doRequest(c.client, req)
}

func (c *Client) RunSyncRequest(j *Job) (job *Job, err error) {
	j.Input.System = c.systemPrompt
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("failed to json marshal: %w", err)
	}
	req, err := c.newRequest(bytes.NewReader(b), http.MethodPost, runSyncPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return doRequest(c.client, req)
}

func (c *Client) StatusRequest(pending *Job) (job *Job, err error) {
	if pending.Output.Done {
		return job, nil
	}

	req, err := c.newRequest(nil, http.MethodGet, statusPath, pending.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for i := 0; i < 3; i++ {
		time.Sleep(time.Second * 3)
		job, err = doRequest(c.client, req)
		if err != nil {
			return nil, err
		}
		if job.Output.Done {
			return job, nil
		}
	}

	return nil, fmt.Errorf("timeout exceeded")
}

func (c *Client) CancelRequest(job *Job) (*Job, error) {
	req, err := c.newRequest(nil, http.MethodPost, cancelPath, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return doRequest(c.client, req)
}

func doRequest(c *http.Client, req *http.Request) (*Job, error) {
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http request failed, status code %d", resp.StatusCode)
	}

	job := new(Job)
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}

	return job, nil
}

func (c *Client) newRequest(body io.Reader, method string, paths ...string) (req *http.Request, err error) {
	u, err := url.JoinPath(c.url, paths...)
	if err != nil {
		return nil, fmt.Errorf("url join path: %w", err)
	}

	req, err = http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+c.apiKey)
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}
