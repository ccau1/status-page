package checks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"status-page/packages/core/domain"
)

// HTTPChecker implements Checker for HTTP/HTTPS endpoints.
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker creates a new HTTP checker.
func NewHTTPChecker(client *http.Client) *HTTPChecker {
	if client == nil {
		client = &http.Client{
			Timeout: 15 * time.Second,
		}
	}
	return &HTTPChecker{client: client}
}

func (h *HTTPChecker) Type() string {
	return "http"
}

func (h *HTTPChecker) Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error) {
	method := "GET"
	if m, ok := def.Parameters["method"]; ok && m != "" {
		method = strings.ToUpper(m)
	}

	req, err := http.NewRequestWithContext(ctx, method, def.Target, nil)
	if err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("invalid request: %v", err),
		}, err
	}

	for k, v := range def.Parameters {
		if strings.HasPrefix(k, "header:") {
			headerName := strings.TrimPrefix(k, "header:")
			req.Header.Set(headerName, v)
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return domain.CheckResult{
			Success: false,
			State:   domain.StateOutage,
			Message: fmt.Sprintf("connection error: %v", err),
		}, err
	}
	defer resp.Body.Close()

	expectedStatus := def.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}

	if resp.StatusCode != expectedStatus {
		state := domain.StateDegraded
		if resp.StatusCode >= 500 {
			state = domain.StateOutage
		}
		return domain.CheckResult{
			Success: false,
			State:   state,
			Message: fmt.Sprintf("expected HTTP %d but got %d", expectedStatus, resp.StatusCode),
		}, nil
	}

	if def.ExpectedBody != "" {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		if err != nil {
			return domain.CheckResult{
				Success: false,
				State:   domain.StateDegraded,
				Message: fmt.Sprintf("failed to read response body: %v", err),
			}, err
		}
		if !strings.Contains(string(bodyBytes), def.ExpectedBody) {
			return domain.CheckResult{
				Success: false,
				State:   domain.StateDegraded,
				Message: fmt.Sprintf("response body did not contain expected content: %s", def.ExpectedBody),
			}, nil
		}
	}

	return domain.CheckResult{
		Success: true,
		State:   domain.StateOperational,
		Message: fmt.Sprintf("HTTP %d OK", resp.StatusCode),
	}, nil
}

func init() {
	Register(NewHTTPChecker(nil))
}
