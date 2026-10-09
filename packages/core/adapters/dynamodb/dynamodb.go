package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"status-page/packages/core/domain"
	"status-page/packages/core/ports"
)

// DynamoDBClientAPI defines the interface for interacting with DynamoDB.
type DynamoDBClientAPI interface {
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
	CreateTable(ctx context.Context, params *dynamodb.CreateTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.CreateTableOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

type DynamoDBAdapter struct {
	client    DynamoDBClientAPI
	tableName string
}

var _ ports.StoragePort = (*DynamoDBAdapter)(nil)

type ddbRecord struct {
	PK                   string `dynamodbav:"PK"` // TENANT#<tenant>
	SK                   string `dynamodbav:"SK"` // PRODUCT#<product>
	Tenant               string `dynamodbav:"tenant"`
	Product              string `dynamodbav:"product"`
	CurrentState         string `dynamodbav:"current_state"`
	Message              string `dynamodbav:"message"`
	FeaturesJSON         string `dynamodbav:"features_json"`
	RegionalFeaturesJSON string `dynamodbav:"regional_features_json,omitempty"`
	CheckResultsJSON     string `dynamodbav:"check_results_json"`
	LastUpdatedUnix      int64  `dynamodbav:"last_updated"`
	ClaimedUntilUnix     int64  `dynamodbav:"claimed_until,omitempty"`
	ClaimedBy            string `dynamodbav:"claimed_by,omitempty"`
}

// New creates a new DynamoDBAdapter using AWS SDK config.
func New(ctx context.Context, tableName string, endpoint string, region string) (*DynamoDBAdapter, error) {
	if tableName == "" {
		tableName = "status_records"
	}

	var opts []func(*awsconfig.LoadOptions) error
	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config for dynamodb: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return &DynamoDBAdapter{
		client:    client,
		tableName: tableName,
	}, nil
}

// NewWithClient creates a DynamoDBAdapter with an injected DynamoDBClientAPI.
func NewWithClient(client DynamoDBClientAPI, tableName string) *DynamoDBAdapter {
	if tableName == "" {
		tableName = "status_records"
	}
	return &DynamoDBAdapter{
		client:    client,
		tableName: tableName,
	}
}

func (d *DynamoDBAdapter) Init(ctx context.Context) error {
	_, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(d.tableName),
	})
	if err == nil {
		return nil
	}

	var notFound *ddbtypes.ResourceNotFoundException
	if errors.As(err, &notFound) {
		_, createErr := d.client.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName: aws.String(d.tableName),
			KeySchema: []ddbtypes.KeySchemaElement{
				{AttributeName: aws.String("PK"), KeyType: ddbtypes.KeyTypeHash},
				{AttributeName: aws.String("SK"), KeyType: ddbtypes.KeyTypeRange},
			},
			AttributeDefinitions: []ddbtypes.AttributeDefinition{
				{AttributeName: aws.String("PK"), AttributeType: ddbtypes.ScalarAttributeTypeS},
				{AttributeName: aws.String("SK"), AttributeType: ddbtypes.ScalarAttributeTypeS},
			},
			BillingMode: ddbtypes.BillingModePayPerRequest,
		})
		if createErr != nil {
			return fmt.Errorf("failed to create dynamodb table %s: %w", d.tableName, createErr)
		}
		return nil
	}

	return fmt.Errorf("failed to describe dynamodb table %s: %w", d.tableName, err)
}

func (d *DynamoDBAdapter) GetStatus(ctx context.Context, tenant string, product string) (*domain.Status, error) {
	pk := fmt.Sprintf("TENANT#%s", tenant)
	sk := fmt.Sprintf("PRODUCT#%s", product)

	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": &ddbtypes.AttributeValueMemberS{Value: pk},
			"SK": &ddbtypes.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item from dynamodb: %w", err)
	}

	if len(out.Item) == 0 {
		return nil, nil
	}

	var rec ddbRecord
	if err := attributevalue.UnmarshalMap(out.Item, &rec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dynamodb item: %w", err)
	}

	return recordToStatus(&rec)
}

func (d *DynamoDBAdapter) ListStatusesByTenant(ctx context.Context, tenant string) ([]domain.Status, error) {
	pk := fmt.Sprintf("TENANT#%s", tenant)

	out, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.tableName),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: pk},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query statuses by tenant from dynamodb: %w", err)
	}

	var statuses []domain.Status
	for _, item := range out.Items {
		var rec ddbRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err == nil {
			if st, err := recordToStatus(&rec); err == nil {
				statuses = append(statuses, *st)
			}
		}
	}

	return statuses, nil
}

func (d *DynamoDBAdapter) ListAllStatuses(ctx context.Context) ([]domain.Status, error) {
	out, err := d.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(d.tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan dynamodb table: %w", err)
	}

	var statuses []domain.Status
	for _, item := range out.Items {
		var rec ddbRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err == nil {
			if st, err := recordToStatus(&rec); err == nil {
				statuses = append(statuses, *st)
			}
		}
	}

	return statuses, nil
}

func (d *DynamoDBAdapter) GetOutdatedStatuses(ctx context.Context, olderThan time.Duration) ([]domain.Status, error) {
	thresholdUnix := time.Now().UTC().Add(-olderThan).Unix()

	out, err := d.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(d.tableName),
		FilterExpression: aws.String("last_updated < :threshold OR attribute_not_exists(last_updated)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":threshold": &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(thresholdUnix, 10)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan outdated statuses from dynamodb: %w", err)
	}

	var statuses []domain.Status
	for _, item := range out.Items {
		var rec ddbRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err == nil {
			if st, err := recordToStatus(&rec); err == nil {
				statuses = append(statuses, *st)
			}
		}
	}

	return statuses, nil
}

func (d *DynamoDBAdapter) SaveStatus(ctx context.Context, status *domain.Status) error {
	featJSON, err := json.Marshal(status.Features)
	if err != nil {
		return fmt.Errorf("failed to marshal features: %w", err)
	}

	checkJSON, err := json.Marshal(status.CheckResults)
	if err != nil {
		return fmt.Errorf("failed to marshal check results: %w", err)
	}

	var regionalJSON string
	if len(status.RegionalFeatures) > 0 {
		b, err := json.Marshal(status.RegionalFeatures)
		if err != nil {
			return fmt.Errorf("failed to marshal regional features: %w", err)
		}
		regionalJSON = string(b)
	}

	var lastUpdatedUnix int64
	if !status.LastUpdated.IsZero() {
		lastUpdatedUnix = status.LastUpdated.UTC().Unix()
	}

	rec := ddbRecord{
		PK:                   fmt.Sprintf("TENANT#%s", status.Tenant),
		SK:                   fmt.Sprintf("PRODUCT#%s", status.Product),
		Tenant:               status.Tenant,
		Product:              status.Product,
		CurrentState:         string(status.CurrentState),
		Message:              status.Message,
		FeaturesJSON:         string(featJSON),
		RegionalFeaturesJSON: regionalJSON,
		CheckResultsJSON:     string(checkJSON),
		LastUpdatedUnix:      lastUpdatedUnix,
	}

	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal ddb record: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to put item to dynamodb: %w", err)
	}

	return nil
}

// ClaimStatus attempts to acquire an exclusive evaluation lease on a status item for leaseDuration.
func (d *DynamoDBAdapter) ClaimStatus(ctx context.Context, tenant string, product string, workerID string, leaseDuration time.Duration) (bool, error) {
	nowUnix := time.Now().UTC().Unix()
	untilUnix := time.Now().UTC().Add(leaseDuration).Unix()

	pk := fmt.Sprintf("TENANT#%s", tenant)
	sk := fmt.Sprintf("PRODUCT#%s", product)

	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": &ddbtypes.AttributeValueMemberS{Value: pk},
			"SK": &ddbtypes.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:    aws.String("SET claimed_until = :until, claimed_by = :worker_id"),
		ConditionExpression: aws.String("attribute_not_exists(claimed_until) OR claimed_until <= :now OR claimed_by = :worker_id"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":until":     &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(untilUnix, 10)},
			":worker_id": &ddbtypes.AttributeValueMemberS{Value: workerID},
			":now":       &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(nowUnix, 10)},
		},
	})

	if err != nil {
		var condErr *ddbtypes.ConditionalCheckFailedException
		var transCondErr *ddbtypes.TransactionCanceledException
		if errors.As(err, &condErr) || errors.As(err, &transCondErr) {
			return false, nil
		}
		return false, fmt.Errorf("failed to claim status in dynamodb for %s/%s: %w", tenant, product, err)
	}

	return true, nil
}

type ddbIncidentRecord struct {
	PK          string   `dynamodbav:"PK"` // INCIDENT#<tenant>
	SK          string   `dynamodbav:"SK"` // ID#<id>
	ID          string   `dynamodbav:"id"`
	Tenant      string   `dynamodbav:"tenant"`
	Product     string   `dynamodbav:"product"`
	Products    []string `dynamodbav:"products,omitempty"`
	CheckIDs    []string `dynamodbav:"check_ids,omitempty"`
	Features    []string `dynamodbav:"features,omitempty"`
	Title       string   `dynamodbav:"title"`
	Severity    string   `dynamodbav:"severity"`
	State       string   `dynamodbav:"state"`
	Message     string   `dynamodbav:"message"`
	Active      bool     `dynamodbav:"active"`
	CreatedAt   int64    `dynamodbav:"created_at"`
	UpdatedAt   int64    `dynamodbav:"updated_at"`
	UpdatesJSON string   `dynamodbav:"updates_json"`
}

func (d *DynamoDBAdapter) GetActiveIncidents(ctx context.Context, tenant string) ([]domain.Incident, error) {
	tenants := []string{tenant}
	if tenant != "*" {
		tenants = append(tenants, "*")
	}

	var incidents []domain.Incident
	for _, t := range tenants {
		pk := fmt.Sprintf("INCIDENT#%s", t)
		out, err := d.client.Query(ctx, &dynamodb.QueryInput{
			TableName:              aws.String(d.tableName),
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
				":pk": &ddbtypes.AttributeValueMemberS{Value: pk},
			},
		})
		if err != nil {
			continue
		}
		for _, itm := range out.Items {
			var rec ddbIncidentRecord
			if err := attributevalue.UnmarshalMap(itm, &rec); err == nil && rec.Active {
				inc := domain.Incident{
					ID:        rec.ID,
					Tenant:    rec.Tenant,
					Product:   rec.Product,
					Products:  rec.Products,
					CheckIDs:  rec.CheckIDs,
					Features:  rec.Features,
					Title:     rec.Title,
					Severity:  domain.IncidentSeverity(rec.Severity),
					State:     domain.IncidentState(rec.State),
					Message:   rec.Message,
					Active:    rec.Active,
					CreatedAt: time.Unix(rec.CreatedAt, 0).UTC(),
					UpdatedAt: time.Unix(rec.UpdatedAt, 0).UTC(),
					Updates:   []domain.IncidentUpdate{},
				}
				if len(inc.Products) == 0 && inc.Product != "" {
					inc.Products = []string{inc.Product}
				}
				if rec.UpdatesJSON != "" {
					_ = json.Unmarshal([]byte(rec.UpdatesJSON), &inc.Updates)
				}
				incidents = append(incidents, inc)
			}
		}
	}
	return incidents, nil
}

func (d *DynamoDBAdapter) SaveIncident(ctx context.Context, incident *domain.Incident) error {
	updatesJSON, _ := json.Marshal(incident.Updates)

	prod := incident.Product
	if prod == "" && len(incident.Products) > 0 {
		prod = incident.Products[0]
	}

	rec := ddbIncidentRecord{
		PK:          fmt.Sprintf("INCIDENT#%s", incident.Tenant),
		SK:          fmt.Sprintf("ID#%s", incident.ID),
		ID:          incident.ID,
		Tenant:      incident.Tenant,
		Product:     prod,
		Products:    incident.Products,
		CheckIDs:    incident.CheckIDs,
		Features:    incident.Features,
		Title:       incident.Title,
		Severity:    string(incident.Severity),
		State:       string(incident.State),
		Message:     incident.Message,
		Active:      incident.Active,
		CreatedAt:   incident.CreatedAt.Unix(),
		UpdatedAt:   incident.UpdatedAt.Unix(),
		UpdatesJSON: string(updatesJSON),
	}
	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return err
	}
	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})
	return err
}

func (d *DynamoDBAdapter) Close() error {
	return nil
}

func recordToStatus(rec *ddbRecord) (*domain.Status, error) {
	var lastUpdated time.Time
	if rec.LastUpdatedUnix > 0 {
		lastUpdated = time.Unix(rec.LastUpdatedUnix, 0).UTC()
	}

	st := &domain.Status{
		Tenant:       rec.Tenant,
		Product:      rec.Product,
		CurrentState: domain.StatusState(rec.CurrentState),
		Message:      rec.Message,
		LastUpdated:  lastUpdated,
		Features:     make(map[string]domain.FeatureStatus),
	}

	if rec.FeaturesJSON != "" {
		_ = json.Unmarshal([]byte(rec.FeaturesJSON), &st.Features)
	}
	if rec.RegionalFeaturesJSON != "" {
		_ = json.Unmarshal([]byte(rec.RegionalFeaturesJSON), &st.RegionalFeatures)
	}
	if rec.CheckResultsJSON != "" {
		_ = json.Unmarshal([]byte(rec.CheckResultsJSON), &st.CheckResults)
	}

	return st, nil
}
