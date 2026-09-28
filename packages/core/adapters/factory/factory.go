package factory

import (
	"context"
	"fmt"
	"os"
	"strings"

	"status-page/packages/core/adapters/dynamodb"
	"status-page/packages/core/adapters/s3"
	"status-page/packages/core/adapters/sqlite"
	"status-page/packages/core/ports"
)

// StorageConfig holds options for initializing storage adapters.
type StorageConfig struct {
	Engine string // "sqlite", "dynamodb", "s3"

	// SQLite options
	SQLiteDSN string

	// DynamoDB options
	DynamoDBTable    string
	DynamoDBEndpoint string

	// S3 options
	S3Bucket   string
	S3Endpoint string

	// Common AWS options
	AWSRegion string
}

// LoadConfigFromEnv reads storage configuration from system environment variables.
func LoadConfigFromEnv() StorageConfig {
	engine := strings.ToLower(os.Getenv("STORAGE_ENGINE"))
	if engine == "" {
		engine = "sqlite"
	}

	sqliteDSN := os.Getenv("SQLITE_DSN")
	if sqliteDSN == "" {
		sqliteDSN = "status.db"
	}

	dynamoTable := os.Getenv("DYNAMODB_TABLE")
	if dynamoTable == "" {
		dynamoTable = "status_records"
	}
	dynamoEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
	if dynamoEndpoint == "" {
		dynamoEndpoint = os.Getenv("AWS_ENDPOINT")
	}

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		s3Bucket = "status-page-bucket"
	}
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		s3Endpoint = os.Getenv("AWS_ENDPOINT")
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	return StorageConfig{
		Engine:           engine,
		SQLiteDSN:        sqliteDSN,
		DynamoDBTable:    dynamoTable,
		DynamoDBEndpoint: dynamoEndpoint,
		S3Bucket:         s3Bucket,
		S3Endpoint:       s3Endpoint,
		AWSRegion:        region,
	}
}

// NewStoragePort instantiates and initializes a StoragePort based on the given configuration.
func NewStoragePort(ctx context.Context, cfg StorageConfig) (ports.StoragePort, error) {
	var adapter ports.StoragePort
	var err error

	switch cfg.Engine {
	case "sqlite":
		adapter, err = sqlite.New(cfg.SQLiteDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to create sqlite adapter: %w", err)
		}

	case "dynamodb":
		adapter, err = dynamodb.New(ctx, cfg.DynamoDBTable, cfg.DynamoDBEndpoint, cfg.AWSRegion)
		if err != nil {
			return nil, fmt.Errorf("failed to create dynamodb adapter: %w", err)
		}

	case "s3":
		adapter, err = s3.New(ctx, cfg.S3Bucket, cfg.S3Endpoint, cfg.AWSRegion)
		if err != nil {
			return nil, fmt.Errorf("failed to create s3 adapter: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported storage engine: %s (supported: sqlite, dynamodb, s3)", cfg.Engine)
	}

	if err := adapter.Init(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize storage adapter (%s): %w", cfg.Engine, err)
	}

	return adapter, nil
}
