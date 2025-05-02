package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sirupsen/logrus"
	"github.com/x-qdo/clickup-billing-report/internal/config"
)

// DynamoDBStore implements the DataStore interface using AWS DynamoDB.
type DynamoDBStore struct {
	client          *dynamodb.Client
	clientsTable    string
	developersTable string
	settingsTable   string
	sessionsTable   string
	log             *logrus.Entry
}

// NewDynamoDBStore creates a new DynamoDBStore instance.
// It now returns config.DataStore interface type.
// AWS Config is loaded externally and passed in.
func NewDynamoDBStore(awsCfg aws.Config, logger *logrus.Logger) (config.DataStore, error) {
	dynamoClient := dynamodb.NewFromConfig(awsCfg)
	logEntry := logger.WithField("component", "DynamoDBStore")

	store := &DynamoDBStore{
		client:          dynamoClient,
		clientsTable:    config.GetTableName("DYNAMODB_CLIENTS_TABLE", config.DefaultClientsTableName),
		developersTable: config.GetTableName("DYNAMODB_DEVELOPERS_TABLE", config.DefaultDevelopersTableName),
		settingsTable:   config.GetTableName("DYNAMODB_SETTINGS_TABLE", config.DefaultSettingsTableName),
		sessionsTable:   config.GetTableName("DYNAMODB_SESSIONS_TABLE", config.DefaultSessionsTableName),
		log:             logEntry,
	}
	logEntry.WithFields(logrus.Fields{
		"clientsTable":    store.clientsTable,
		"developersTable": store.developersTable,
		"settingsTable":   store.settingsTable,
		"sessionsTable":   store.sessionsTable,
	}).Info("DynamoDB table names configured")

	return store, nil
}

func (s *DynamoDBStore) GetClient(ctx context.Context, name string) (*config.Client, error) {
	s.log.WithField("client_name", name).Debug("Getting client")
	key, err := attributevalue.Marshal(name)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal key: %w", err)
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.clientsTable),
		Key:       map[string]types.AttributeValue{"Name": key},
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.log.WithError(err).Error("Failed to get client from DynamoDB")
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	if result.Item == nil {
		s.log.WithField("client_name", name).Warn("Client not found")
		return nil, config.ErrNotFound
	}

	var client config.Client
	err = attributevalue.UnmarshalMap(result.Item, &client)
	if err != nil {
		s.log.WithError(err).Error("Failed to unmarshal client data")
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	s.log.WithField("client_name", name).Info("Client retrieved successfully")
	return &client, nil
}

func (s *DynamoDBStore) ListClients(ctx context.Context) ([]config.Client, error) {
	s.log.Debug("Listing clients")
	input := &dynamodb.ScanInput{
		TableName: aws.String(s.clientsTable),
	}

	var clients []config.Client
	paginator := dynamodb.NewScanPaginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to scan clients table page")
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}
		var pageClients []config.Client
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageClients)
		if err != nil {
			s.log.WithError(err).Error("Failed to unmarshal client list page")
			return nil, fmt.Errorf("failed to unmarshal items: %w", err)
		}
		clients = append(clients, pageClients...)
	}

	s.log.WithField("count", len(clients)).Info("Clients listed successfully")
	return clients, nil
}

func (s *DynamoDBStore) SaveClient(ctx context.Context, client config.Client) error {
	s.log.WithField("client_name", client.Name).Debug("Saving client")
	client.UpdatedAt = time.Now().UTC()
	if client.CreatedAt.IsZero() {
		client.CreatedAt = client.UpdatedAt
	}

	av, err := attributevalue.MarshalMap(client)
	if err != nil {
		return fmt.Errorf("failed to marshal client: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.clientsTable),
		Item:      av,
		// Optional: Add ConditionExpression to prevent overwriting existing items if needed
		// ConditionExpression: aws.String("attribute_not_exists(Name)"), // Example: only save if new
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		// TODO: Handle specific errors like ConditionalCheckFailedException
		s.log.WithError(err).WithField("client_name", client.Name).Error("Failed to save client")
		return fmt.Errorf("failed to put item: %w", err)
	}

	s.log.WithField("client_name", client.Name).Info("Client saved successfully")
	return nil
}

func (s *DynamoDBStore) DeleteClient(ctx context.Context, name string) error {
	s.log.WithField("client_name", name).Debug("Deleting client")
	key, err := attributevalue.Marshal(name)
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.clientsTable),
		Key:       map[string]types.AttributeValue{"Name": key},
		// Optional: Add ConditionExpression to ensure item exists before deleting
		// ConditionExpression: aws.String("attribute_exists(Name)"),
	}

	_, err = s.client.DeleteItem(ctx, input)
	if err != nil {
		// TODO: Handle specific errors like ConditionalCheckFailedException if needed
		s.log.WithError(err).WithField("client_name", name).Error("Failed to delete client")
		return fmt.Errorf("failed to delete item: %w", err)
	}

	s.log.WithField("client_name", name).Info("Client deleted successfully")
	return nil
}

// --- Developer Operations ---

func (s *DynamoDBStore) GetDeveloper(ctx context.Context, name string) (*config.Developer, error) {
	s.log.WithField("developer_name", name).Debug("Getting developer")
	key, err := attributevalue.Marshal(name)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal developer key: %w", err)
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.developersTable),
		Key:       map[string]types.AttributeValue{"Name": key},
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.log.WithError(err).Error("Failed to get developer from DynamoDB")
		return nil, fmt.Errorf("failed to get developer item: %w", err)
	}

	if result.Item == nil {
		s.log.WithField("developer_name", name).Warn("Developer not found")
		return nil, config.ErrNotFound // Use error from config package
	}

	var dev config.Developer
	err = attributevalue.UnmarshalMap(result.Item, &dev)
	if err != nil {
		s.log.WithError(err).Error("Failed to unmarshal developer data")
		return nil, fmt.Errorf("failed to unmarshal developer item: %w", err)
	}

	s.log.WithField("developer_name", name).Info("Developer retrieved successfully")
	return &dev, nil
}

func (s *DynamoDBStore) ListDevelopers(ctx context.Context) ([]config.Developer, error) {
	s.log.Debug("Listing developers")
	input := &dynamodb.ScanInput{
		TableName: aws.String(s.developersTable),
	}

	var developers []config.Developer
	paginator := dynamodb.NewScanPaginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.log.WithError(err).Error("Failed to scan developers table page")
			return nil, fmt.Errorf("failed to scan developers table: %w", err)
		}
		var pageDevelopers []config.Developer
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageDevelopers)
		if err != nil {
			s.log.WithError(err).Error("Failed to unmarshal developer list page")
			return nil, fmt.Errorf("failed to unmarshal developer items: %w", err)
		}
		developers = append(developers, pageDevelopers...)
	}

	s.log.WithField("count", len(developers)).Info("Developers listed successfully")
	return developers, nil
}

func (s *DynamoDBStore) SaveDeveloper(ctx context.Context, developer config.Developer) error {
	s.log.WithField("developer_name", developer.Name).Debug("Saving developer")
	developer.UpdatedAt = time.Now().UTC()
	if developer.CreatedAt.IsZero() {
		developer.CreatedAt = developer.UpdatedAt
	}

	av, err := attributevalue.MarshalMap(developer)
	if err != nil {
		return fmt.Errorf("failed to marshal developer: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.developersTable),
		Item:      av,
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		s.log.WithError(err).WithField("developer_name", developer.Name).Error("Failed to save developer")
		return fmt.Errorf("failed to put developer item: %w", err)
	}

	s.log.WithField("developer_name", developer.Name).Info("Developer saved successfully")
	return nil
}

func (s *DynamoDBStore) DeleteDeveloper(ctx context.Context, name string) error {
	s.log.WithField("developer_name", name).Debug("Deleting developer")
	key, err := attributevalue.Marshal(name)
	if err != nil {
		return fmt.Errorf("failed to marshal developer key: %w", err)
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.developersTable),
		Key:       map[string]types.AttributeValue{"Name": key},
	}

	_, err = s.client.DeleteItem(ctx, input)
	if err != nil {
		s.log.WithError(err).WithField("developer_name", name).Error("Failed to delete developer")
		return fmt.Errorf("failed to delete developer item: %w", err)
	}

	s.log.WithField("developer_name", name).Info("Developer deleted successfully")
	return nil
}

// --- Global Settings Operations ---

func (s *DynamoDBStore) GetGlobalSettings(ctx context.Context, key string) (*config.GlobalSettings, error) {
	s.log.WithField("settings_key", key).Debug("Getting global settings")
	dbKey, err := attributevalue.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings key: %w", err)
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.settingsTable),
		Key:       map[string]types.AttributeValue{"SettingKey": dbKey},
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.log.WithError(err).Error("Failed to get global settings from DynamoDB")
		return nil, fmt.Errorf("failed to get settings item: %w", err)
	}

	if result.Item == nil {
		s.log.WithField("settings_key", key).Warn("Global settings not found")
		return nil, config.ErrNotFound // Use error from config package
	}

	var settings config.GlobalSettings
	err = attributevalue.UnmarshalMap(result.Item, &settings)
	if err != nil {
		s.log.WithError(err).Error("Failed to unmarshal global settings data")
		return nil, fmt.Errorf("failed to unmarshal settings item: %w", err)
	}

	s.log.WithField("settings_key", key).Info("Global settings retrieved successfully")
	return &settings, nil
}

func (s *DynamoDBStore) SaveGlobalSettings(ctx context.Context, settings config.GlobalSettings) error {
	s.log.WithField("settings_key", settings.SettingKey).Debug("Saving global settings")
	settings.UpdatedAt = time.Now().UTC()

	// Ensure the primary key is set
	if settings.SettingKey == "" {
		return fmt.Errorf("SettingKey cannot be empty for GlobalSettings")
	}

	// Marshal, excluding fields marked with `dynamodbav:"-"`
	av, err := attributevalue.MarshalMap(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal global settings: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.settingsTable),
		Item:      av,
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		s.log.WithError(err).WithField("settings_key", settings.SettingKey).Error("Failed to save global settings")
		return fmt.Errorf("failed to put settings item: %w", err)
	}

	s.log.WithField("settings_key", settings.SettingKey).Info("Global settings saved successfully")
	return nil
}

// --- Session Operations ---

func (s *DynamoDBStore) GetSession(ctx context.Context, sessionID string) (*config.SessionState, error) {
	s.log.WithField("session_id_prefix", sessionID[:min(len(sessionID), 8)]).Debug("Getting session") // Log prefix only
	key, err := attributevalue.Marshal(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session key: %w", err)
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.sessionsTable),
		Key:       map[string]types.AttributeValue{"SessionID": key},
	}

	result, err := s.client.GetItem(ctx, input)
	if err != nil {
		s.log.WithError(err).Error("Failed to get session from DynamoDB")
		return nil, fmt.Errorf("failed to get session item: %w", err)
	}

	if result.Item == nil {
		s.log.Warn("Session not found")
		return nil, config.ErrNotFound // Use error from config package
	}

	var session config.SessionState
	err = attributevalue.UnmarshalMap(result.Item, &session)
	if err != nil {
		s.log.WithError(err).Error("Failed to unmarshal session data")
		return nil, fmt.Errorf("failed to unmarshal session item: %w", err)
	}

	// Check expiry
	if time.Now().UTC().After(session.ExpiresAt) {
		s.log.Warn("Session found but expired")
		// Optionally delete the expired session here
		// go s.DeleteSession(context.Background(), sessionID) // Delete in background
		return nil, config.ErrNotFound // Treat expired as not found
	}

	s.log.Debug("Session retrieved successfully")
	return &session, nil
}

func (s *DynamoDBStore) SaveSession(ctx context.Context, session config.SessionState) error {
	s.log.WithField("session_id_prefix", session.SessionID[:min(len(session.SessionID), 8)]).Debug("Saving session")
	session.CreatedAt = time.Now().UTC() // Set creation/update time

	// Add TTL attribute for automatic cleanup by DynamoDB
	ttlTimestamp := session.ExpiresAt.Unix()
	av, err := attributevalue.MarshalMap(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Add TTL attribute if expiry is valid
	if !session.ExpiresAt.IsZero() {
		ttlAv, err := attributevalue.Marshal(ttlTimestamp)
		if err != nil {
			s.log.WithError(err).Warn("Failed to marshal TTL attribute for session")
		} else {
			av["TTL"] = ttlAv // DynamoDB TTL attribute name is often 'TTL'
		}
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(s.sessionsTable),
		Item:      av,
	}

	_, err = s.client.PutItem(ctx, input)
	if err != nil {
		s.log.WithError(err).Error("Failed to save session")
		return fmt.Errorf("failed to put session item: %w", err)
	}

	s.log.Debug("Session saved successfully")
	return nil
}

func (s *DynamoDBStore) DeleteSession(ctx context.Context, sessionID string) error {
	s.log.WithField("session_id_prefix", sessionID[:min(len(sessionID), 8)]).Debug("Deleting session")
	key, err := attributevalue.Marshal(sessionID)
	if err != nil {
		return fmt.Errorf("failed to marshal session key: %w", err)
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.sessionsTable),
		Key:       map[string]types.AttributeValue{"SessionID": key},
	}

	_, err = s.client.DeleteItem(ctx, input)
	if err != nil {
		// Check if it's ResourceNotFoundException, which might be okay if TTL already deleted it
		// var rnfe *types.ResourceNotFoundException
		// if errors.As(err, &rnfe) {
		//  s.log.Warn("Session not found during delete, possibly already expired/deleted by TTL")
		//  return nil // Treat as success if not found
		// }
		s.log.WithError(err).Error("Failed to delete session")
		return fmt.Errorf("failed to delete session item: %w", err)
	}

	s.log.Debug("Session deleted successfully")
	return nil
}
