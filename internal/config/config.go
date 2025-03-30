package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// --- DataStore Interface (Moved from storage package) ---

// DataStore defines the interface for accessing application configuration and state.
type DataStore interface {
	// Client operations
	GetClient(ctx context.Context, name string) (*Client, error)
	ListClients(ctx context.Context) ([]Client, error)
	SaveClient(ctx context.Context, client Client) error
	DeleteClient(ctx context.Context, name string) error

	// Developer operations
	GetDeveloper(ctx context.Context, name string) (*Developer, error)
	ListDevelopers(ctx context.Context) ([]Developer, error)
	SaveDeveloper(ctx context.Context, developer Developer) error
	DeleteDeveloper(ctx context.Context, name string) error

	// Global Settings operations
	GetGlobalSettings(ctx context.Context, key string) (*GlobalSettings, error)
	SaveGlobalSettings(ctx context.Context, settings GlobalSettings) error

	// Session operations
	GetSession(ctx context.Context, sessionID string) (*SessionState, error)
	SaveSession(ctx context.Context, session SessionState) error
	DeleteSession(ctx context.Context, sessionID string) error
}

// Common storage errors (can be defined here or elsewhere if needed broadly)
var (
	ErrNotFound      = NewStorageError("item not found")
	ErrAlreadyExists = NewStorageError("item already exists")
)

type StorageError struct {
	message string
}

func NewStorageError(message string) *StorageError {
	return &StorageError{message: message}
}

func (e *StorageError) Error() string {
	return e.message
}

// --- DynamoDB Table Names ---
const (
	DefaultClientsTableName    = "ClickUpReporter-Clients"
	DefaultDevelopersTableName = "ClickUpReporter-Developers"
	DefaultSettingsTableName   = "ClickUpReporter-Settings"
	DefaultSessionsTableName   = "ClickUpReporter-Sessions"
	DefaultDebugPort           = "5000"
)

// --- Configuration Structs ---

// Client represents the configuration for a specific client.
type Client struct {
	Name             string    `dynamodbav:"Name" json:"name"` // Primary Key for Clients table
	ClickUpListID    string    `dynamodbav:"ClickUpListID" json:"clickup_list_id"`
	ClickUpTeamID    string    `dynamodbav:"ClickUpTeamID" json:"clickup_team_id"`
	ContractIncluded float64   `dynamodbav:"ContractIncluded" json:"contract_included"`
	TogglSyncEnabled bool      `dynamodbav:"TogglSyncEnabled" json:"toggl_sync_enabled"`
	TogglWorkspaceID string    `dynamodbav:"TogglWorkspaceID,omitempty" json:"toggl_workspace_id,omitempty"`
	CreatedAt        time.Time `dynamodbav:"CreatedAt" json:"created_at"`
	UpdatedAt        time.Time `dynamodbav:"UpdatedAt" json:"updated_at"`
}

// Developer represents a developer and their billing coefficient.
type Developer struct {
	Name        string    `dynamodbav:"Name" json:"name"` // Primary Key for Developers table
	Coefficient float64   `dynamodbav:"Coefficient" json:"coefficient"`
	ClickUpID   string    `dynamodbav:"ClickUpID,omitempty" json:"clickup_id,omitempty"` // Optional: ClickUp User ID if needed for filtering
	CreatedAt   time.Time `dynamodbav:"CreatedAt" json:"created_at"`
	UpdatedAt   time.Time `dynamodbav:"UpdatedAt" json:"updated_at"`
}

// GlobalSettings represents application-wide settings.
type GlobalSettings struct {
	SettingKey          string    `dynamodbav:"SettingKey"` // Primary Key (e.g., "global")
	ClickUpClientID     string    `dynamodbav:"-"`          // Loaded from Env
	ClickUpClientSecret string    `dynamodbav:"-"`          // Loaded from Env, not stored directly
	DefaultDemoListID   string    `dynamodbav:"DefaultDemoListID,omitempty"`
	DefaultDemoTeamID   string    `dynamodbav:"DefaultDemoTeamID,omitempty"`
	SlackBotToken       string    `dynamodbav:"-"` // Loaded from Env, not stored directly
	WikiAPIToken        string    `dynamodbav:"-"` // Loaded from Env, not stored directly
	WikiBaseURL         string    `dynamodbav:"WikiBaseURL,omitempty"`
	TogglAPIToken       string    `dynamodbav:"-"` // Loaded from Env, not stored directly
	SessionSecret       string    `dynamodbav:"-"` // Loaded from Env for signing session cookies/tokens
	BaseURL             string    `dynamodbav:"-"` // Application's base URL (from Env) for callbacks etc.
	DebugMode           bool      `dynamodbav:"-"` // Loaded from Env (DEBUG_MODE)
	DebugPort           string    `dynamodbav:"-"` // Loaded from Env (DEBUG_PORT)
	UpdatedAt           time.Time `dynamodbav:"UpdatedAt,omitempty"`
}

// SessionState represents the data stored for an authenticated user session.
type SessionState struct {
	SessionID    string    `dynamodbav:"SessionID"`    // Primary Key (e.g., secure random string)
	ClickUpToken string    `dynamodbav:"ClickUpToken"` // Encrypted Access Token
	UserID       string    `dynamodbav:"UserID"`       // ClickUp User ID
	UserName     string    `dynamodbav:"UserName"`     // ClickUp User Name
	ExpiresAt    time.Time `dynamodbav:"ExpiresAt"`    // Token expiry
	CreatedAt    time.Time `dynamodbav:"CreatedAt"`
	TTL          int64     `dynamodbav:"TTL,omitempty"` // DynamoDB TTL attribute
}

// --- Report Structs (Input/Output for API handlers) ---

// BillableReportInput defines parameters for the billable report API endpoint.
type BillableReportInput struct {
	ListID          string `json:"list_id"`          // ClickUp List ID to generate the report for
	RefreshInvoiced bool   `json:"refresh_invoiced"` // Flag to update InvoicedHours to match BillableHours
}

// BillableReportTask represents a task included in the billable report API response.
type BillableReportTask struct {
	TaskID          string   `json:"task_id"`
	CustomID        string   `json:"custom_id"`
	Name            string   `json:"name"`
	Priority        string   `json:"priority"`
	Tags            []string `json:"tags"`
	BillableHours   float64  `json:"billable_hours"`
	InvoicedHours   float64  `json:"invoiced_hours"`
	MonthlyReported float64  `json:"monthly_reported"`
	Reporter        string   `json:"reporter"`
	Status          string   `json:"status"`
	URL             string   `json:"url"`
}

// BillableReportTotals holds the sum of hours for the billable report API response.
type BillableReportTotals struct {
	BillableHours   float64 `json:"billable_hours"`
	InvoicedHours   float64 `json:"invoiced_hours"`
	MonthlyReported float64 `json:"monthly_reported"`
}

// BillableReportOutput holds the results of the billable report for the API response.
type BillableReportOutput struct {
	Tasks  []BillableReportTask `json:"tasks"`
	Totals BillableReportTotals `json:"totals"`
}

// --- Environment Variable Loading ---

// LoadGlobalSettingsFromEnv loads sensitive or deployment-specific settings from environment variables.
func LoadGlobalSettingsFromEnv(log *logrus.Entry) (*GlobalSettings, error) {
	clientID := os.Getenv("CLICKUP_CLIENT_ID")
	clientSecret := os.Getenv("CLICKUP_CLIENT_SECRET")
	sessionSecret := os.Getenv("SESSION_SECRET")
	baseURL := os.Getenv("BASE_URL")

	// Basic validation for critical secrets
	if clientID == "" || clientSecret == "" || sessionSecret == "" || baseURL == "" {
		missing := []string{}
		if clientID == "" {
			missing = append(missing, "CLICKUP_CLIENT_ID")
		}
		if clientSecret == "" {
			missing = append(missing, "CLICKUP_CLIENT_SECRET")
		}
		if sessionSecret == "" {
			missing = append(missing, "SESSION_SECRET")
		}
		if baseURL == "" {
			missing = append(missing, "BASE_URL")
		}
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}
	if len(sessionSecret) < 32 {
		log.Warn("SESSION_SECRET should be at least 32 bytes long for optimal security")
	}

	// Load Debug Mode settings
	debugModeStr := os.Getenv("DEBUG_MODE")
	debugMode, _ := strconv.ParseBool(debugModeStr) // Defaults to false if parsing fails or var is empty

	debugPort := os.Getenv("DEBUG_PORT")
	if debugPort == "" {
		debugPort = DefaultDebugPort
	}

	return &GlobalSettings{
		ClickUpClientID:     clientID,
		ClickUpClientSecret: clientSecret,
		SlackBotToken:       os.Getenv("SLACK_BOT_TOKEN"), // Optional
		WikiAPIToken:        os.Getenv("WIKI_API_TOKEN"),  // Optional
		TogglAPIToken:       os.Getenv("TOGGL_API_TOKEN"), // Optional
		SessionSecret:       sessionSecret,
		BaseURL:             baseURL,
		DebugMode:           debugMode,
		DebugPort:           debugPort,
		// Non-secret fields like DefaultDemoListID, WikiBaseURL will be loaded/merged from DB
	}, nil
}

// GetTableName returns the DynamoDB table name from env var or default.
func GetTableName(envVar, defaultValue string) string {
	name := os.Getenv(envVar)
	if name == "" {
		return defaultValue
	}
	return name
}

// --- AppConfig Singleton ---

// AppConfig holds the fully loaded application configuration.
type AppConfig struct {
	Global     *GlobalSettings
	Clients    map[string]Client    // Cache of clients, mapped by Name
	Developers map[string]Developer // Cache of developers, mapped by Name
	Store      DataStore            // Use the interface defined in this package
	Logger     *logrus.Logger
}

var (
	appConfig *AppConfig
	configErr error
	once      sync.Once
)

// LoadConfig initializes and returns the application configuration using dependency injection for the store.
// It ensures configuration is loaded only once (singleton pattern).
func LoadConfig(ctx context.Context, store DataStore) (*AppConfig, error) {
	once.Do(func() {
		logger := logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logLevelStr := os.Getenv("LOG_LEVEL")
		logLevel, err := logrus.ParseLevel(logLevelStr)
		if err != nil {
			logLevel = logrus.InfoLevel
		}
		logger.SetLevel(logLevel)
		logEntry := logger.WithField("service", "config-loader")

		logEntry.Info("Loading application configuration...")
		_ = godotenv.Load() // Load .env file if present

		// Load required settings from environment
		globalSettingsEnv, err := LoadGlobalSettingsFromEnv(logEntry)
		if err != nil {
			configErr = fmt.Errorf("failed to load required settings from environment: %w", err)
			logEntry.WithError(configErr).Fatal("Environment settings loading failed") // Fatal if required are missing
			return
		}
		logEntry.Info("Required global settings loaded from environment")
		if globalSettingsEnv.DebugMode {
			logEntry.Warnf("DEBUG MODE ENABLED (Port: %s)", globalSettingsEnv.DebugPort)
		}

		// --- Store is now injected, no need to initialize it here ---
		if store == nil {
			configErr = fmt.Errorf("data store implementation was not provided to LoadConfig")
			logEntry.WithError(configErr).Fatal("Data store is nil")
			return
		}
		logEntry.Info("Data store provided")
		// -----------------------------------------------------------

		// Load non-sensitive global settings from DB
		globalSettingsDB, err := store.GetGlobalSettings(ctx, "global")
		if err != nil && err != ErrNotFound { // Use error defined in this package
			configErr = fmt.Errorf("failed to load global settings from DB: %w", err)
			logEntry.WithError(configErr).Error("DB settings loading failed")
			// Don't return here, allow fallback to env settings
		} else if err == ErrNotFound {
			logEntry.Warn("Global settings not found in DB, using environment/defaults.")
			// Optionally save initial settings from env to DB here if desired
			// globalSettingsEnv.SettingKey = "global"
			// _ = store.SaveGlobalSettings(ctx, *globalSettingsEnv)
		} else {
			logEntry.Info("Loaded global settings from DB")
		}

		// Merge DB settings into Env settings
		finalGlobalSettings := mergeGlobalSettings(globalSettingsEnv, globalSettingsDB)
		logEntry.Info("Merged global settings")

		// Load clients and developers from DB
		clientsList, err := store.ListClients(ctx)
		if err != nil {
			configErr = fmt.Errorf("failed to load clients: %w", err)
			logEntry.WithError(configErr).Error("Failed loading clients from DB")
			// Don't return, allow app to potentially run without clients if needed
		}
		clientsMap := make(map[string]Client)
		for _, c := range clientsList {
			clientsMap[c.Name] = c
		}
		logEntry.WithField("count", len(clientsMap)).Info("Loaded clients from DB")

		developersList, err := store.ListDevelopers(ctx)
		if err != nil {
			configErr = fmt.Errorf("failed to load developers: %w", err)
			logEntry.WithError(configErr).Error("Failed loading developers from DB")
			// Don't return, allow app to potentially run without developers
		}
		developersMap := make(map[string]Developer)
		for _, d := range developersList {
			developersMap[d.Name] = d
		}
		logEntry.WithField("count", len(developersMap)).Info("Loaded developers from DB")

		appConfig = &AppConfig{
			Global:     finalGlobalSettings,
			Clients:    clientsMap,
			Developers: developersMap,
			Store:      store, // Store the injected store
			Logger:     logger,
		}
		logEntry.Info("Application configuration loaded successfully")
	})

	// Return the potentially partial config even if non-fatal errors occurred during loading
	if appConfig == nil && configErr == nil {
		// This should not happen if once.Do completed without fatal errors
		return nil, fmt.Errorf("configuration loading failed silently")
	}
	// Return the config and any non-fatal error encountered
	return appConfig, configErr
}

// GetConfig returns the already loaded configuration.
// Panics if LoadConfig hasn't been called successfully first.
func GetConfig() *AppConfig {
	if appConfig == nil {
		// Configuration must be loaded explicitly via LoadConfig in the main/init function.
		panic("Configuration has not been loaded. Call config.LoadConfig first.")
	}
	return appConfig
}

// mergeGlobalSettings merges settings from DB into the Env-loaded struct.
// Env settings (especially secrets and debug flags) take precedence.
func mergeGlobalSettings(env *GlobalSettings, db *GlobalSettings) *GlobalSettings {
	merged := *env // Start with env settings (secrets, debug flags)

	if db == nil {
		merged.SettingKey = "global" // Ensure key is set even if DB is empty
		return &merged               // No DB settings, return env only
	}

	// Overwrite with DB values only if they are non-empty/non-zero
	// and the corresponding env value is empty/zero (except for keys)
	merged.SettingKey = db.SettingKey // Key always comes from DB if present
	if merged.DefaultDemoListID == "" {
		merged.DefaultDemoListID = db.DefaultDemoListID
	}
	if merged.DefaultDemoTeamID == "" {
		merged.DefaultDemoTeamID = db.DefaultDemoTeamID
	}
	if merged.WikiBaseURL == "" {
		merged.WikiBaseURL = db.WikiBaseURL
	}
	// Keep UpdatedAt from DB
	merged.UpdatedAt = db.UpdatedAt

	return &merged
}
