package checks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"status-page/packages/core/domain"
)

// PlaywrightChecker delegates browser synthetic checks to the Playwright runner microservice.
type PlaywrightChecker struct {
	client    *http.Client
	runnerURL string
}

// PlaywrightRunRequest defines the payload sent to the Playwright runner service.
type PlaywrightRunRequest struct {
	Scenario   string            `json:"scenario,omitempty"`
	Target     string            `json:"target"`
	TimeoutMs  int               `json:"timeoutMs,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

// PlaywrightRunResponse defines the expected response from the Playwright runner service.
type PlaywrightRunResponse struct {
	Success   bool                   `json:"success"`
	State     domain.StatusState     `json:"state"`
	LatencyMs int64                  `json:"latencyMs"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewPlaywrightChecker creates a new PlaywrightChecker instance.
func NewPlaywrightChecker(client *http.Client, defaultRunnerURL string) *PlaywrightChecker {
	if client == nil {
		client = &http.Client{
			Timeout: 45 * time.Second,
		}
	}
	return &PlaywrightChecker{
		client:    client,
		runnerURL: defaultRunnerURL,
	}
}

func (p *PlaywrightChecker) Type() string {
	return "playwright"
}

func (p *PlaywrightChecker) getRunnerURL(def domain.CheckDefinition) string {
	// 1. Parameter override in check definition
	if customURL, ok := def.Parameters["runner_url"]; ok && strings.TrimSpace(customURL) != "" {
		return strings.TrimSpace(customURL)
	}
	// 2. Struct default
	if p.runnerURL != "" {
		return p.runnerURL
	}
	// 3. Environment variable
	if envURL := strings.TrimSpace(os.Getenv("PLAYWRIGHT_RUNNER_URL")); envURL != "" {
		return envURL
	}
	// 4. Default container service URL
	return "http://playwright-runner:8088/run"
}

func (p *PlaywrightChecker) Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error) {
	runnerURL := p.getRunnerURL(def)

	scenario := "page-load"
	if s, ok := def.Parameters["scenario"]; ok && s != "" {
		scenario = s
	}

	timeoutMs := def.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 20000
	}

	payload := PlaywrightRunRequest{
		Scenario:   scenario,
		Target:     def.Target,
		TimeoutMs:  timeoutMs,
		Parameters: def.Parameters,
	}

	reqBytes, err := json.Marshal(payload)
	if err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("failed to marshal runner request: %v", err),
		}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, runnerURL, bytes.NewReader(reqBytes))
	if err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("failed to create request to runner: %v", err),
		}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("playwright-runner connection error: %v", err),
		}, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("failed to read response from playwright-runner: %v", err),
		}, err
	}

	var runRes PlaywrightRunResponse
	if err := json.Unmarshal(bodyBytes, &runRes); err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("invalid response from playwright-runner (HTTP %d): %s", resp.StatusCode, string(bodyBytes)),
		}, fmt.Errorf("invalid json from runner: %w", err)
	}

	state := runRes.State
	if state == "" {
		if runRes.Success {
			state = domain.StateOperational
		} else {
			state = domain.StateOutage
		}
	}

	return domain.CheckResult{
		Success: runRes.Success,
		State:   state,
		Message: runRes.Message,
	}, nil
}

func init() {
	Register(NewPlaywrightChecker(nil, ""))
}
