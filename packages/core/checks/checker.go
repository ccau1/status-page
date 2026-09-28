package checks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"status-page/packages/core/domain"
)

// Checker defines the interface that all modular check types must implement.
type Checker interface {
	// Type returns the unique identifier for this check method (e.g. "http", "tcp", "dns").
	Type() string
	// Execute performs the check and returns the result.
	Execute(ctx context.Context, def domain.CheckDefinition) (domain.CheckResult, error)
}

// Registry manages the set of available modular checkers.
type Registry struct {
	mu       sync.RWMutex
	checkers map[string]Checker
}

// NewRegistry creates a new checker registry.
func NewRegistry() *Registry {
	return &Registry{
		checkers: make(map[string]Checker),
	}
}

// Register adds a checker to the registry.
func (r *Registry) Register(c Checker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checkers[c.Type()] = c
}

// Get retrieves a checker by its type.
func (r *Registry) Get(checkType string) (Checker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.checkers[checkType]
	return c, ok
}

// Types returns all registered checker types.
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.checkers))
	for k := range r.checkers {
		types = append(types, k)
	}
	return types
}

// DefaultRegistry is the global default registry with built-in checkers.
var DefaultRegistry = NewRegistry()

// Register registers a checker into the default registry.
func Register(c Checker) {
	DefaultRegistry.Register(c)
}

// Get gets a checker from the default registry.
func Get(checkType string) (Checker, bool) {
	return DefaultRegistry.Get(checkType)
}

// ExecuteCheck executes a check using the registered checker.
func ExecuteCheck(ctx context.Context, reg *Registry, def domain.CheckDefinition) domain.CheckResult {
	if reg == nil {
		reg = DefaultRegistry
	}

	checker, ok := reg.Get(def.Type)
	if !ok {
		return domain.CheckResult{
			CheckID:   def.ID,
			Feature:   def.Feature,
			Type:      def.Type,
			Success:   false,
			State:     domain.StateOutage,
			LatencyMs: 0,
			Message:   fmt.Sprintf("unknown check type: %s", def.Type),
			Timestamp: time.Now().UTC(),
		}
	}

	timeout := 10 * time.Second
	if def.TimeoutMs > 0 {
		timeout = time.Duration(def.TimeoutMs) * time.Millisecond
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	res, err := checker.Execute(checkCtx, def)
	latency := time.Since(start).Milliseconds()

	res.CheckID = def.ID
	res.Feature = def.Feature
	res.Type = def.Type
	res.LatencyMs = latency
	res.Timestamp = time.Now().UTC()

	if err != nil && res.Message == "" {
		res.Message = err.Error()
	}
	if !res.Success && res.State == "" {
		res.State = domain.StateOutage
	}
	if res.Success && res.State == "" {
		res.State = domain.StateOperational
	}

	return res
}
