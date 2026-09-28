package checks

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"status-page/packages/core/domain"
)

// SyntheticChecker simulates status checks for mock, preview, or staging environments.
type SyntheticChecker struct{}

// NewSyntheticChecker creates a new SyntheticChecker.
func NewSyntheticChecker() *SyntheticChecker {
	return &SyntheticChecker{}
}

func (s *SyntheticChecker) Type() string {
	return "synthetic"
}

func (s *SyntheticChecker) Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error) {
	state := domain.StateOperational
	success := true
	message := "Synthetic check passed"

	if simulatedState, ok := def.Parameters["simulated_state"]; ok && simulatedState != "" {
		switch domain.StatusState(simulatedState) {
		case domain.StateDegraded:
			state = domain.StateDegraded
			success = true
			message = "Synthetic check reported degraded performance"
		case domain.StateOutage:
			state = domain.StateOutage
			success = false
			message = "Synthetic check reported service outage"
		case domain.StateMaintenance:
			state = domain.StateMaintenance
			success = true
			message = "Synthetic check in planned maintenance"
		}
	}

	if simulatedLatency, ok := def.Parameters["simulated_latency_ms"]; ok {
		if ms, err := strconv.Atoi(simulatedLatency); err == nil && ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}

	if customMsg, ok := def.Parameters["message"]; ok && customMsg != "" {
		message = customMsg
	}

	return domain.CheckResult{
		Success: success,
		State:   state,
		Message: fmt.Sprintf("[%s] %s", def.Target, message),
	}, nil
}

func init() {
	Register(NewSyntheticChecker())
}
