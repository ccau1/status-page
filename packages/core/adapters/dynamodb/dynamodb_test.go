package dynamodb_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	ddbadapter "status-page/packages/core/adapters/dynamodb"
	"status-page/packages/core/domain"
)

type mockDDBClient struct {
	items map[string]map[string]ddbtypes.AttributeValue
}

func newMockDDBClient() *mockDDBClient {
	return &mockDDBClient{
		items: make(map[string]map[string]ddbtypes.AttributeValue),
	}
}

func (m *mockDDBClient) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	return &dynamodb.DescribeTableOutput{}, nil
}

func (m *mockDDBClient) CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error) {
	return &dynamodb.CreateTableOutput{}, nil
}

func (m *mockDDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	pk := params.Item["PK"].(*ddbtypes.AttributeValueMemberS).Value
	sk := params.Item["SK"].(*ddbtypes.AttributeValueMemberS).Value
	key := pk + "#" + sk
	m.items[key] = params.Item
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	pk := params.Key["PK"].(*ddbtypes.AttributeValueMemberS).Value
	sk := params.Key["SK"].(*ddbtypes.AttributeValueMemberS).Value
	key := pk + "#" + sk
	item, ok := m.items[key]
	if !ok {
		return &dynamodb.GetItemOutput{}, nil
	}
	return &dynamodb.GetItemOutput{Item: item}, nil
}

func (m *mockDDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	var matched []map[string]ddbtypes.AttributeValue
	pkVal := params.ExpressionAttributeValues[":pk"].(*ddbtypes.AttributeValueMemberS).Value

	for _, item := range m.items {
		if item["PK"].(*ddbtypes.AttributeValueMemberS).Value == pkVal {
			matched = append(matched, item)
		}
	}
	return &dynamodb.QueryOutput{Items: matched}, nil
}

func (m *mockDDBClient) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	var list []map[string]ddbtypes.AttributeValue
	for _, item := range m.items {
		list = append(list, item)
	}
	return &dynamodb.ScanOutput{Items: list}, nil
}

func (m *mockDDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	pk := params.Key["PK"].(*ddbtypes.AttributeValueMemberS).Value
	sk := params.Key["SK"].(*ddbtypes.AttributeValueMemberS).Value
	key := pk + "#" + sk
	item, ok := m.items[key]
	if !ok {
		item = make(map[string]ddbtypes.AttributeValue)
		item["PK"] = params.Key["PK"]
		item["SK"] = params.Key["SK"]
		m.items[key] = item
	}

	if untilAttr, exists := item["claimed_until"]; exists {
		var untilVal int64
		if n, ok := untilAttr.(*ddbtypes.AttributeValueMemberN); ok {
			untilVal, _ = strconv.ParseInt(n.Value, 10, 64)
		}
		nowVal, _ := strconv.ParseInt(params.ExpressionAttributeValues[":now"].(*ddbtypes.AttributeValueMemberN).Value, 10, 64)
		workerIDVal := params.ExpressionAttributeValues[":worker_id"].(*ddbtypes.AttributeValueMemberS).Value
		var claimedBy string
		if cb, ok := item["claimed_by"].(*ddbtypes.AttributeValueMemberS); ok {
			claimedBy = cb.Value
		}

		if untilVal > nowVal && claimedBy != workerIDVal {
			return nil, &ddbtypes.ConditionalCheckFailedException{
				Message: aws.String("The conditional request failed"),
			}
		}
	}

	item["claimed_until"] = params.ExpressionAttributeValues[":until"]
	item["claimed_by"] = params.ExpressionAttributeValues[":worker_id"]

	return &dynamodb.UpdateItemOutput{}, nil
}

func TestDynamoDBAdapter_SaveAndGet(t *testing.T) {
	client := newMockDDBClient()
	adapter := ddbadapter.NewWithClient(client, "test-table")
	ctx := context.Background()

	now := time.Now().UTC()
	st := &domain.Status{
		Tenant:       "tenant-2",
		Product:      "billing",
		CurrentState: domain.StateOperational,
		LastUpdated:  now,
		Features: map[string]domain.FeatureStatus{
			"invoicing": {
				ID:          "invoicing",
				Name:        "Automated Invoicing",
				State:       domain.StateOperational,
				LastChecked: now,
			},
		},
	}

	if err := adapter.SaveStatus(ctx, st); err != nil {
		t.Fatalf("failed to save to ddb: %v", err)
	}

	fetched, err := adapter.GetStatus(ctx, "tenant-2", "billing")
	if err != nil {
		t.Fatalf("failed to get from ddb: %v", err)
	}
	if fetched == nil {
		t.Fatalf("expected status to be found in ddb")
	}
	if fetched.Tenant != "tenant-2" || fetched.Product != "billing" {
		t.Errorf("unexpected status returned: %+v", fetched)
	}
}

func TestDynamoDBAdapter_ClaimStatus(t *testing.T) {
	client := newMockDDBClient()
	adapter := ddbadapter.NewWithClient(client, "test-table")
	ctx := context.Background()

	// 1. Initial claim should succeed
	claimed, err := adapter.ClaimStatus(ctx, "acme", "api", "worker-1", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error claiming: %v", err)
	}
	if !claimed {
		t.Fatalf("expected worker-1 to acquire lease")
	}

	// 2. Worker 1 renewing should succeed
	claimed, err = adapter.ClaimStatus(ctx, "acme", "api", "worker-1", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error re-claiming: %v", err)
	}
	if !claimed {
		t.Fatalf("expected worker-1 to renew lease")
	}

	// 3. Worker 2 attempting to claim active lease should fail
	claimed, err = adapter.ClaimStatus(ctx, "acme", "api", "worker-2", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error on conflict: %v", err)
	}
	if claimed {
		t.Fatalf("expected worker-2 to be rejected due to active lease")
	}
}
