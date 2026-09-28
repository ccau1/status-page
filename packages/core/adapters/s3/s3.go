package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"status-page/packages/core/domain"
	"status-page/packages/core/ports"
)

// S3ClientAPI is an interface covering the AWS S3 methods used by S3Adapter, enabling unit testing.
type S3ClientAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	CreateBucket(ctx context.Context, params *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
}

type S3Adapter struct {
	client     S3ClientAPI
	bucketName string
	prefix     string
}

var _ ports.StoragePort = (*S3Adapter)(nil)

// New creates an S3Adapter using AWS SDK config.
func New(ctx context.Context, bucketName string, endpoint string, region string) (*S3Adapter, error) {
	if bucketName == "" {
		bucketName = "status-page-bucket"
	}

	var opts []func(*awsconfig.LoadOptions) error
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})

	return &S3Adapter{
		client:     client,
		bucketName: bucketName,
		prefix:     "statuses",
	}, nil
}

// NewWithClient creates an S3Adapter with an injected S3ClientAPI (useful for testing or custom mocks).
func NewWithClient(client S3ClientAPI, bucketName string) *S3Adapter {
	if bucketName == "" {
		bucketName = "status-page-bucket"
	}
	return &S3Adapter{
		client:     client,
		bucketName: bucketName,
		prefix:     "statuses",
	}
}

func (s *S3Adapter) objectKey(tenant string, product string) string {
	return fmt.Sprintf("%s/%s/%s.json", s.prefix, tenant, product)
}

func (s *S3Adapter) Init(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err == nil {
		return nil
	}

	// Try creating the bucket if it doesn't exist
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		var bfe *s3types.BucketAlreadyOwnedByYou
		var bae *s3types.BucketAlreadyExists
		if errors.As(err, &bfe) || errors.As(err, &bae) {
			return nil
		}
		// If permission or other error, return it or log
		return fmt.Errorf("could not create bucket %s: %w", s.bucketName, err)
	}
	return nil
}

func (s *S3Adapter) GetStatus(ctx context.Context, tenant string, product string) (*domain.Status, error) {
	key := s.objectKey(tenant, product)
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get s3 object %s: %w", key, err)
	}
	defer resp.Body.Close()

	var st domain.Status
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &st); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status JSON: %w", err)
	}
	if st.Features == nil {
		st.Features = make(map[string]domain.FeatureStatus)
	}

	return &st, nil
}

func (s *S3Adapter) ListStatusesByTenant(ctx context.Context, tenant string) ([]domain.Status, error) {
	prefix := fmt.Sprintf("%s/%s/", s.prefix, tenant)
	return s.listByPrefix(ctx, prefix)
}

func (s *S3Adapter) ListAllStatuses(ctx context.Context) ([]domain.Status, error) {
	prefix := fmt.Sprintf("%s/", s.prefix)
	return s.listByPrefix(ctx, prefix)
}

func (s *S3Adapter) GetOutdatedStatuses(ctx context.Context, olderThan time.Duration) ([]domain.Status, error) {
	all, err := s.ListAllStatuses(ctx)
	if err != nil {
		return nil, err
	}

	threshold := time.Now().UTC().Add(-olderThan)
	var outdated []domain.Status
	for _, st := range all {
		if st.LastUpdated.Before(threshold) || st.LastUpdated.IsZero() {
			outdated = append(outdated, st)
		}
	}
	return outdated, nil
}

func (s *S3Adapter) SaveStatus(ctx context.Context, status *domain.Status) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	key := s.objectKey(status.Tenant, status.Product)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("failed to write status to s3 key %s: %w", key, err)
	}
	return nil
}

func (s *S3Adapter) GetActiveIncidents(ctx context.Context, tenant string) ([]domain.Incident, error) {
	prefixes := []string{
		fmt.Sprintf("incidents/%s/", tenant),
	}
	if tenant != "*" {
		prefixes = append(prefixes, "incidents/*/")
	}

	var results []domain.Incident
	for _, pfx := range prefixes {
		resp, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucketName),
			Prefix: aws.String(pfx),
		})
		if err != nil {
			continue
		}
		for _, item := range resp.Contents {
			getResp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(s.bucketName),
				Key:    item.Key,
			})
			if err != nil {
				continue
			}
			bodyBytes, _ := io.ReadAll(getResp.Body)
			getResp.Body.Close()

			var inc domain.Incident
			if err := json.Unmarshal(bodyBytes, &inc); err == nil && inc.Active {
				results = append(results, inc)
			}
		}
	}
	return results, nil
}

func (s *S3Adapter) SaveIncident(ctx context.Context, incident *domain.Incident) error {
	data, err := json.MarshalIndent(incident, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal incident: %w", err)
	}
	key := fmt.Sprintf("incidents/%s/%s.json", incident.Tenant, incident.ID)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	return err
}

func (s *S3Adapter) listByPrefix(ctx context.Context, prefix string) ([]domain.Status, error) {
	var results []domain.Status
	var continuationToken *string

	for {
		resp, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(s.bucketName),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list s3 objects: %w", err)
		}

		for _, item := range resp.Contents {
			if !strings.HasSuffix(*item.Key, ".json") {
				continue
			}

			getResp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(s.bucketName),
				Key:    item.Key,
			})
			if err != nil {
				continue
			}

			bodyBytes, err := io.ReadAll(getResp.Body)
			getResp.Body.Close()
			if err != nil {
				continue
			}

			var st domain.Status
			if err := json.Unmarshal(bodyBytes, &st); err == nil {
				if st.Features == nil {
					st.Features = make(map[string]domain.FeatureStatus)
				}
				results = append(results, st)
			}
		}

		if resp.IsTruncated != nil && *resp.IsTruncated {
			continuationToken = resp.NextContinuationToken
		} else {
			break
		}
	}

	return results, nil
}

func (s *S3Adapter) Close() error {
	return nil
}
