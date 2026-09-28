package dynamodb_test

import (
	"context"
	"testing"
	"time"

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
