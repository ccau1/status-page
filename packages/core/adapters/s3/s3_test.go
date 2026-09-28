package s3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	s3adapter "status-page/packages/core/adapters/s3"
	"status-page/packages/core/domain"
)

type mockS3Client struct {
	objects map[string][]byte
}

func newMockS3Client() *mockS3Client {
	return &mockS3Client{
		objects: make(map[string][]byte),
	}
}

func (m *mockS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return &s3.HeadBucketOutput{}, nil
}

func (m *mockS3Client) CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	return &s3.CreateBucketOutput{}, nil
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	data, err := io.ReadAll(params.Body)
	if err != nil {
		return nil, err
	}
	m.objects[*params.Key] = data
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	data, ok := m.objects[*params.Key]
	if !ok {
		return nil, &s3types.NoSuchKey{}
	}
	return &s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(data)),
	}, nil
}

func (m *mockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	var contents []s3types.Object
	prefix := ""
	if params.Prefix != nil {
		prefix = *params.Prefix
	}
	for k := range m.objects {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			contents = append(contents, s3types.Object{
				Key: aws.String(k),
			})
		}
	}
	return &s3.ListObjectsV2Output{
		Contents: contents,
	}, nil
}

func TestS3Adapter_SaveAndGet(t *testing.T) {
	client := newMockS3Client()
	adapter := s3adapter.NewWithClient(client, "test-bucket")
	ctx := context.Background()

	now := time.Now().UTC()
	st := &domain.Status{
		Tenant:       "tenant-1",
		Product:      "api-gateway",
		CurrentState: domain.StateOperational,
		LastUpdated:  now,
		Features: map[string]domain.FeatureStatus{
			"routing": {
				ID:          "routing",
				Name:        "Request Routing",
				State:       domain.StateOperational,
				LastChecked: now,
			},
		},
	}

	if err := adapter.SaveStatus(ctx, st); err != nil {
		t.Fatalf("failed to save to s3: %v", err)
	}

	fetched, err := adapter.GetStatus(ctx, "tenant-1", "api-gateway")
	if err != nil {
		t.Fatalf("failed to get from s3: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected status to be found in s3")
	}
	if fetched.Tenant != "tenant-1" || fetched.Product != "api-gateway" {
		t.Errorf("unexpected status returned: %+v", fetched)
	}

	list, err := adapter.ListStatusesByTenant(ctx, "tenant-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 status for tenant-1 in s3, got %d, err: %v", len(list), err)
	}
	_ = json.NewDecoder
}
