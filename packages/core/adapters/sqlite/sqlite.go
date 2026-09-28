package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"status-page/packages/core/domain"
	"status-page/packages/core/ports"
)

type SQLiteAdapter struct {
	db *sql.DB
}

var _ ports.StoragePort = (*SQLiteAdapter)(nil)

// New creates a new SQLiteAdapter given a connection DSN/filepath.
func New(dsn string) (*SQLiteAdapter, error) {
	if dsn == "" {
		dsn = "status.db"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize connection pooling for SQLite
	db.SetMaxOpenConns(1)

	return &SQLiteAdapter{db: db}, nil
}

// Init creates the tables and indices required.
func (s *SQLiteAdapter) Init(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS statuses (
		tenant TEXT NOT NULL,
		product TEXT NOT NULL,
		current_state TEXT NOT NULL,
		message TEXT,
		features_json TEXT NOT NULL,
		check_results_json TEXT,
		last_updated INTEGER NOT NULL,
		PRIMARY KEY(tenant, product)
	);
	CREATE INDEX IF NOT EXISTS idx_statuses_last_updated ON statuses(last_updated);

	CREATE TABLE IF NOT EXISTS incidents (
		id TEXT PRIMARY KEY,
		tenant TEXT NOT NULL,
		product TEXT,
		products_json TEXT,
		check_ids_json TEXT,
		features_json TEXT,
		title TEXT NOT NULL,
		severity TEXT NOT NULL,
		state TEXT NOT NULL,
		message TEXT,
		active INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		updates_json TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_incidents_tenant ON incidents(tenant, active);
	`
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to initialize sqlite schema: %w", err)
	}

	// Automatic schema migration for existing databases: ensure newly added columns exist
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE incidents ADD COLUMN products_json TEXT;")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE incidents ADD COLUMN check_ids_json TEXT;")
	_, _ = s.db.ExecContext(ctx, "ALTER TABLE incidents ADD COLUMN features_json TEXT;")

	return nil
}

// GetStatus retrieves a single product status for a tenant.
func (s *SQLiteAdapter) GetStatus(ctx context.Context, tenant string, product string) (*domain.Status, error) {
	query := `
	SELECT tenant, product, current_state, message, features_json, check_results_json, last_updated
	FROM statuses
	WHERE tenant = ? AND product = ?
	`
	row := s.db.QueryRowContext(ctx, query, tenant, product)

	var st domain.Status
	var featJSON, checkJSON string
	var lastUpdatedUnix int64

	err := row.Scan(&st.Tenant, &st.Product, &st.CurrentState, &st.Message, &featJSON, &checkJSON, &lastUpdatedUnix)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	if lastUpdatedUnix > 0 {
		st.LastUpdated = time.Unix(lastUpdatedUnix, 0).UTC()
	} else {
		st.LastUpdated = time.Time{}
	}

	if featJSON != "" {
		if err := json.Unmarshal([]byte(featJSON), &st.Features); err != nil {
			return nil, fmt.Errorf("failed to unmarshal features: %w", err)
		}
	} else {
		st.Features = make(map[string]domain.FeatureStatus)
	}

	if checkJSON != "" {
		if err := json.Unmarshal([]byte(checkJSON), &st.CheckResults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal check results: %w", err)
		}
	}

	return &st, nil
}

// ListStatusesByTenant returns all products for a given tenant.
func (s *SQLiteAdapter) ListStatusesByTenant(ctx context.Context, tenant string) ([]domain.Status, error) {
	query := `
	SELECT tenant, product, current_state, message, features_json, check_results_json, last_updated
	FROM statuses
	WHERE tenant = ?
	ORDER BY product ASC
	`
	rows, err := s.db.QueryContext(ctx, query, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to list statuses by tenant: %w", err)
	}
	defer rows.Close()

	return s.scanRows(rows)
}

// ListAllStatuses returns all products for all tenants.
func (s *SQLiteAdapter) ListAllStatuses(ctx context.Context) ([]domain.Status, error) {
	query := `
	SELECT tenant, product, current_state, message, features_json, check_results_json, last_updated
	FROM statuses
	ORDER BY tenant ASC, product ASC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all statuses: %w", err)
	}
	defer rows.Close()

	return s.scanRows(rows)
}

// GetOutdatedStatuses returns statuses whose last_updated is older than the given duration.
func (s *SQLiteAdapter) GetOutdatedStatuses(ctx context.Context, olderThan time.Duration) ([]domain.Status, error) {
	thresholdUnix := time.Now().UTC().Add(-olderThan).Unix()
	query := `
	SELECT tenant, product, current_state, message, features_json, check_results_json, last_updated
	FROM statuses
	WHERE last_updated < ?
	ORDER BY last_updated ASC
	`
	rows, err := s.db.QueryContext(ctx, query, thresholdUnix)
	if err != nil {
		return nil, fmt.Errorf("failed to query outdated statuses: %w", err)
	}
	defer rows.Close()

	return s.scanRows(rows)
}

// SaveStatus inserts or updates the status.
func (s *SQLiteAdapter) SaveStatus(ctx context.Context, status *domain.Status) error {
	featJSON, err := json.Marshal(status.Features)
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}

	checkJSON, err := json.Marshal(status.CheckResults)
	if err != nil {
		return fmt.Errorf("failed to marshal check results: %w", err)
	}

	var lastUpdatedUnix int64
	if !status.LastUpdated.IsZero() {
		lastUpdatedUnix = status.LastUpdated.UTC().Unix()
	}

	query := `
	INSERT INTO statuses (tenant, product, current_state, message, features_json, check_results_json, last_updated)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(tenant, product) DO UPDATE SET
		current_state = excluded.current_state,
		message = excluded.message,
		features_json = excluded.features_json,
		check_results_json = excluded.check_results_json,
		last_updated = excluded.last_updated
	`
	_, err = s.db.ExecContext(ctx, query,
		status.Tenant,
		status.Product,
		string(status.CurrentState),
		status.Message,
		string(featJSON),
		string(checkJSON),
		lastUpdatedUnix,
	)
	if err != nil {
		return fmt.Errorf("failed to save status: %w", err)
	}
	return nil
}

func (s *SQLiteAdapter) scanRows(rows *sql.Rows) ([]domain.Status, error) {
	var list []domain.Status
	for rows.Next() {
		var st domain.Status
		var featJSON, checkJSON string
		var lastUpdatedUnix int64

		if err := rows.Scan(&st.Tenant, &st.Product, &st.CurrentState, &st.Message, &featJSON, &checkJSON, &lastUpdatedUnix); err != nil {
			return nil, fmt.Errorf("failed to scan status row: %w", err)
		}
		if lastUpdatedUnix > 0 {
			st.LastUpdated = time.Unix(lastUpdatedUnix, 0).UTC()
		} else {
			st.LastUpdated = time.Time{}
		}
		if featJSON != "" {
			_ = json.Unmarshal([]byte(featJSON), &st.Features)
		}
		if st.Features == nil {
			st.Features = make(map[string]domain.FeatureStatus)
		}
		if checkJSON != "" {
			_ = json.Unmarshal([]byte(checkJSON), &st.CheckResults)
		}
		list = append(list, st)
	}
	return list, rows.Err()
}

// GetActiveIncidents returns active incidents for a given tenant or global (*).
func (s *SQLiteAdapter) GetActiveIncidents(ctx context.Context, tenant string) ([]domain.Incident, error) {
	query := `
	SELECT id, tenant, product, products_json, check_ids_json, features_json, title, severity, state, message, active, created_at, updated_at, updates_json
	FROM incidents
	WHERE active = 1 AND (tenant = ? OR tenant = '*')
	ORDER BY updated_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to query active incidents: %w", err)
	}
	defer rows.Close()

	var list []domain.Incident
	for rows.Next() {
		var inc domain.Incident
		var prod sql.NullString
		var prodsJSON sql.NullString
		var checkIDsJSON sql.NullString
		var featsJSON sql.NullString
		var msg sql.NullString
		var activeInt int
		var createdUnix, updatedUnix int64
		var updatesJSON sql.NullString

		err := rows.Scan(
			&inc.ID,
			&inc.Tenant,
			&prod,
			&prodsJSON,
			&checkIDsJSON,
			&featsJSON,
			&inc.Title,
			&inc.Severity,
			&inc.State,
			&msg,
			&activeInt,
			&createdUnix,
			&updatedUnix,
			&updatesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident row: %w", err)
		}

		if prod.Valid {
			inc.Product = prod.String
		}
		if prodsJSON.Valid && prodsJSON.String != "" {
			_ = json.Unmarshal([]byte(prodsJSON.String), &inc.Products)
		}
		if len(inc.Products) == 0 && inc.Product != "" {
			inc.Products = []string{inc.Product}
		}

		if checkIDsJSON.Valid && checkIDsJSON.String != "" {
			_ = json.Unmarshal([]byte(checkIDsJSON.String), &inc.CheckIDs)
		}
		if inc.CheckIDs == nil {
			inc.CheckIDs = []string{}
		}

		if featsJSON.Valid && featsJSON.String != "" {
			_ = json.Unmarshal([]byte(featsJSON.String), &inc.Features)
		}
		if inc.Features == nil {
			inc.Features = []string{}
		}

		if msg.Valid {
			inc.Message = msg.String
		}
		inc.Active = activeInt == 1
		inc.CreatedAt = time.Unix(createdUnix, 0).UTC()
		inc.UpdatedAt = time.Unix(updatedUnix, 0).UTC()

		if updatesJSON.Valid && updatesJSON.String != "" {
			_ = json.Unmarshal([]byte(updatesJSON.String), &inc.Updates)
		}
		if inc.Updates == nil {
			inc.Updates = []domain.IncidentUpdate{}
		}

		list = append(list, inc)
	}
	return list, rows.Err()
}

// SaveIncident creates or updates an incident.
func (s *SQLiteAdapter) SaveIncident(ctx context.Context, incident *domain.Incident) error {
	var updatesJSON, prodsJSON, checkIDsJSON, featsJSON []byte
	if len(incident.Updates) > 0 {
		var err error
		updatesJSON, err = json.Marshal(incident.Updates)
		if err != nil {
			return fmt.Errorf("failed to marshal incident updates: %w", err)
		}
	}
	if len(incident.Products) > 0 {
		prodsJSON, _ = json.Marshal(incident.Products)
	}
	if len(incident.CheckIDs) > 0 {
		checkIDsJSON, _ = json.Marshal(incident.CheckIDs)
	}
	if len(incident.Features) > 0 {
		featsJSON, _ = json.Marshal(incident.Features)
	}

	createdUnix := incident.CreatedAt.UTC().Unix()
	if createdUnix <= 0 {
		createdUnix = time.Now().UTC().Unix()
		incident.CreatedAt = time.Unix(createdUnix, 0).UTC()
	}

	updatedUnix := incident.UpdatedAt.UTC().Unix()
	if updatedUnix <= 0 {
		updatedUnix = time.Now().UTC().Unix()
		incident.UpdatedAt = time.Unix(updatedUnix, 0).UTC()
	}

	activeInt := 0
	if incident.Active {
		activeInt = 1
	}

	// For backwards compatibility, populate Product from first element if unset
	if incident.Product == "" && len(incident.Products) > 0 {
		incident.Product = incident.Products[0]
	}

	query := `
	INSERT INTO incidents (id, tenant, product, products_json, check_ids_json, features_json, title, severity, state, message, active, created_at, updated_at, updates_json)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		tenant = excluded.tenant,
		product = excluded.product,
		products_json = excluded.products_json,
		check_ids_json = excluded.check_ids_json,
		features_json = excluded.features_json,
		title = excluded.title,
		severity = excluded.severity,
		state = excluded.state,
		message = excluded.message,
		active = excluded.active,
		updated_at = excluded.updated_at,
		updates_json = excluded.updates_json
	`
	_, err := s.db.ExecContext(ctx, query,
		incident.ID,
		incident.Tenant,
		incident.Product,
		string(prodsJSON),
		string(checkIDsJSON),
		string(featsJSON),
		incident.Title,
		string(incident.Severity),
		string(incident.State),
		incident.Message,
		activeInt,
		createdUnix,
		updatedUnix,
		string(updatesJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to save incident: %w", err)
	}
	return nil
}

func (s *SQLiteAdapter) Close() error {
	return s.db.Close()
}
