package ports

import (
	"context"
	"time"

	"status-page/packages/core/domain"
)

// StoragePort defines the contract for persisting and retrieving status information.
type StoragePort interface {
	// Init initializes the database schema, tables, or buckets if they don't exist.
	Init(ctx context.Context) error

	// GetStatus retrieves the status for a specific tenant and product.
	GetStatus(ctx context.Context, tenant string, product string) (*domain.Status, error)

	// ListStatusesByTenant retrieves all statuses for a given tenant.
	ListStatusesByTenant(ctx context.Context, tenant string) ([]domain.Status, error)

	// ListAllStatuses retrieves all status records across all tenants.
	ListAllStatuses(ctx context.Context) ([]domain.Status, error)

	// GetOutdatedStatuses finds statuses whose LastUpdated is older than the specified duration or are uninitialized.
	GetOutdatedStatuses(ctx context.Context, olderThan time.Duration) ([]domain.Status, error)

	// SaveStatus saves or updates a status record.
	SaveStatus(ctx context.Context, status *domain.Status) error

	// GetActiveIncidents retrieves active incident banners for a tenant (or global).
	GetActiveIncidents(ctx context.Context, tenant string) ([]domain.Incident, error)

	// SaveIncident stores or updates an incident record.
	SaveIncident(ctx context.Context, incident *domain.Incident) error

	// Close cleanly shuts down the storage connection.
	Close() error
}
