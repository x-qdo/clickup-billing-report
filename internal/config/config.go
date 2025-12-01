package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sirupsen/logrus"
)

// DataStore defines the interface for interacting with the configuration storage.
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

	// Job operations
	GetJob(ctx context.Context, jobID string) (*Job, error)
	SaveJob(ctx context.Context, job Job) error
	UpdateJobStatus(ctx context.Context, jobID, status, output, errMsg string) error
}

// Common storage errors (can be defined here or elsewhere if needed broadly)
var (
	ErrNotFound      = NewStorageError("item not found")
	ErrAlreadyExists = NewStorageError("item already exists")
)

type StorageError struct {
	message string
	cause   error // Optional underlying cause
}

func NewStorageError(message string) *StorageError {
	return &StorageError{message: message}
}

func (e *StorageError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// --- DynamoDB Table Names ---
const (
	DefaultClientsTableName    = "ClickUpReporter-Clients"
	DefaultDevelopersTableName = "ClickUpReporter-Developers"
	DefaultSettingsTableName   = "ClickUpReporter-Settings"
	DefaultSessionsTableName   = "ClickUpReporter-Sessions"
	DefaultJobsTableName       = "ClickUpReporter-Jobs"
	DefaultDebugPort           = "5000"
)

// Unwrap returns the underlying cause, allowing errors.Is and errors.As to work.
func (e *StorageError) Unwrap() error {
	return e.cause
}

// --- Data Structures ---

// Client represents a customer or project entity.
type Client struct {
	Name             string    `dynamodbav:"Name" json:"name"` // Primary Key for Clients table
	ClickUpListID    string    `dynamodbav:"ClickUpListID" json:"clickup_list_id"`
	ClickUpTeamID    string    `dynamodbav:"ClickUpTeamID" json:"clickup_team_id"`
	ContractIncluded float64   `dynamodbav:"ContractIncluded" json:"contract_included"` // Included hours/value in contract
	TogglSyncEnabled bool      `dynamodbav:"TogglSyncEnabled" json:"toggl_sync_enabled"`
	TogglWorkspaceID string    `dynamodbav:"TogglWorkspaceID,omitempty" json:"toggl_workspace_id,omitempty"`
	CreatedAt        time.Time `dynamodbav:"CreatedAt" json:"created_at"`
	UpdatedAt        time.Time `dynamodbav:"UpdatedAt" json:"updated_at"`
	// Add other client-specific settings as needed
}

// Developer represents a team member.
type Developer struct {
	Name        string    `dynamodbav:"Name" json:"name"` // Primary Key for Developers table
	Coefficient float64   `dynamodbav:"Coefficient" json:"coefficient"`
	ClickUpID   string    `dynamodbav:"ClickUpID,omitempty" json:"clickup_id,omitempty"` // Optional: ClickUp User ID
	TogglID     string    `dynamodbav:"TogglID,omitempty" json:"toggl_id,omitempty"`     // Optional: Toggl User ID
	CreatedAt   time.Time `dynamodbav:"CreatedAt" json:"created_at"`
	UpdatedAt   time.Time `dynamodbav:"UpdatedAt" json:"updated_at"`
}

// GlobalSettings holds application-wide configuration.
// Secrets are loaded from environment variables, not stored directly in DynamoDB.
type GlobalSettings struct {
	SettingKey          string    `dynamodbav:"SettingKey"` // Primary Key (e.g., "global")
	ClickUpClientID     string    `dynamodbav:"-"`          // Loaded from Env
	ClickUpClientSecret string    `dynamodbav:"-"`          // Loaded from Env, not stored directly
	SessionSecret       string    `dynamodbav:"-"`          // Loaded from Env for session encryption
	BaseURL             string    `dynamodbav:"-"`          // Loaded from Env, base URL of the deployed app
	DefaultDemoListID   string    `dynamodbav:"DefaultDemoListID,omitempty"`
	DefaultDemoTeamID   string    `dynamodbav:"DefaultDemoTeamID,omitempty"`
	SlackBotToken       string    `dynamodbav:"-"` // Loaded from Env, not stored directly
	WikiAPIToken        string    `dynamodbav:"-"` // Loaded from Env, not stored directly
	WikiBaseURL         string    `dynamodbav:"WikiBaseURL,omitempty"`
	TogglAPIToken       string    `dynamodbav:"-"`                   // Loaded from Env, not stored directly
	DebugMode           bool      `dynamodbav:"-"`                   // Loaded from Env (e.g., "DEBUG_MODE=true")
	DebugPort           string    `dynamodbav:"-"`                   // Loaded from Env (e.g., "DEBUG_PORT=8080")
	UpdatedAt           time.Time `dynamodbav:"UpdatedAt,omitempty"` // When DB settings were last updated
}

// SessionState stores user session information, typically linked to an OAuth token.
type SessionState struct {
	SessionID    string    `dynamodbav:"SessionID"`    // Primary Key (e.g., secure random string)
	ClickUpToken string    `dynamodbav:"ClickUpToken"` // Encrypted Access Token
	UserID       string    `dynamodbav:"UserID"`       // ClickUp User ID
	UserName     string    `dynamodbav:"UserName"`     // ClickUp User Name
	ExpiresAt    time.Time `dynamodbav:"ExpiresAt"`    // Token expiry
	CreatedAt    time.Time `dynamodbav:"CreatedAt"`
	TTL          int64     `dynamodbav:"TTL,omitempty"` // DynamoDB TTL attribute
}

// Job status constants
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)

// Job type constants
const (
	JobTypeTimetrack = "timetrack"
	JobTypeBillable  = "billable"
)

// Job represents an async job for report generation.
type Job struct {
	JobID     string    `dynamodbav:"JobID" json:"job_id"`
	Type      string    `dynamodbav:"Type" json:"type"`
	Status    string    `dynamodbav:"Status" json:"status"`
	Input     string    `dynamodbav:"Input" json:"input,omitempty"`
	Output    string    `dynamodbav:"Output,omitempty" json:"result,omitempty"`
	Error     string    `dynamodbav:"Error,omitempty" json:"error,omitempty"`
	UserID    string    `dynamodbav:"UserID" json:"user_id"`
	CreatedAt time.Time `dynamodbav:"CreatedAt" json:"created_at"`
	UpdatedAt time.Time `dynamodbav:"UpdatedAt" json:"updated_at"`
	TTL       int64     `dynamodbav:"TTL,omitempty" json:"-"`
}

// TimetrackJobInput represents input parameters for a timetrack job.
type TimetrackJobInput struct {
	ReportDate      string `json:"report_date"`
	RefreshBillable bool   `json:"refresh_billable"`
	Format          string `json:"format"`
	ClickUpToken    string `json:"clickup_token"`
}

// BillableJobInput represents input parameters for a billable job.
type BillableJobInput struct {
	ClientName      string `json:"client_name"`
	RefreshInvoiced bool   `json:"refresh_invoiced"`
	Format          string `json:"format"`
	ClickUpToken    string `json:"clickup_token"`
}

// JobFileOutput represents the output when job produces a file (Excel).
type JobFileOutput struct {
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
}

// --- Report Structures ---

// BillableReportTask represents a single task row in the billable report.
type BillableReportTask struct {
	TaskID          string   `json:"task_id"`
	CustomID        string   `json:"custom_id"`
	Name            string   `json:"name"`
	Priority        string   `json:"priority"`
	Tags            []string `json:"tags"`
	BillableHours   float64  `json:"billable_hours"`
	InvoicedHours   float64  `json:"invoiced_hours"`
	MonthlyReported float64  `json:"monthly_reported"` // Calculated: Billable - Invoiced
	Reporter        string   `json:"reporter"`
	URL             string   `json:"url"`
}

// BillableReportTotals represents the summary totals for the billable report.
type BillableReportTotals struct {
	BillableHours   float64 `json:"billable_hours"`
	InvoicedHours   float64 `json:"invoiced_hours"`
	MonthlyReported float64 `json:"monthly_reported"`
}

// BillableReportOutput represents the complete data for the billable report.
type BillableReportOutput struct {
	Tasks         []BillableReportTask `json:"tasks"`          // Non-internal tasks meeting criteria
	InternalTasks []BillableReportTask `json:"internal_tasks"` // Tasks tagged as 'internal'
	Totals        BillableReportTotals `json:"totals"`         // Totals for non-internal tasks
}

// --- Config Loading ---

// LoadGlobalSettingsFromEnv loads settings primarily from environment variables.
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

	debugMode := os.Getenv("DEBUG_MODE") == "true"
	debugPort := os.Getenv("DEBUG_PORT")
	if debugMode && debugPort == "" {
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
	}, nil
}

func GetTableName(envVar, defaultValue string) string {
	name := os.Getenv(envVar)
	if name == "" {
		return defaultValue
	}
	return name
}

type AppConfig struct {
	Global     *GlobalSettings
	Clients    map[string]Client    // Cache of clients, mapped by Name
	Developers map[string]Developer // Cache of developers, mapped by Name
	Store      DataStore            // Use the interface defined in this package
	Logger     *logrus.Logger
}

func mergeGlobalSettings(env *GlobalSettings, db *GlobalSettings) *GlobalSettings {
	merged := *env // Start with env settings (secrets, debug flags)

	if db == nil {
		merged.SettingKey = "global" // Ensure key is set even if DB is empty
		return &merged               // No DB settings, return env only
	}

	// Overwrite with DB values only if they are non-empty/non-zero
	// and the corresponding env value is empty/zero (except for keys)
	merged.SettingKey = db.SettingKey // Always take DB key if present

	if merged.DefaultDemoListID == "" {
		merged.DefaultDemoListID = db.DefaultDemoListID
	}
	if merged.DefaultDemoTeamID == "" {
		merged.DefaultDemoTeamID = db.DefaultDemoTeamID
	}
	if merged.WikiBaseURL == "" {
		merged.WikiBaseURL = db.WikiBaseURL
	}
	// Add other DB-overridable fields here

	merged.UpdatedAt = db.UpdatedAt // Reflect DB update time

	return &merged
}

func LoadConfig(ctx context.Context, store DataStore) (*AppConfig, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logLevelStr := os.Getenv("LOG_LEVEL")
	logLevel, err := logrus.ParseLevel(logLevelStr)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	logEntry := logger.WithField("service", "config-loader")

	if store == nil {
		return nil, fmt.Errorf("DataStore cannot be nil")
	}

	// 1. Load Global Settings from Env
	envSettings, err := LoadGlobalSettingsFromEnv(logEntry)
	if err != nil {
		logEntry.WithError(err).Error("Failed to load critical settings from environment")
		return nil, fmt.Errorf("failed to load environment settings: %w", err)
	}
	logger.SetLevel(logLevel) // Re-apply log level potentially derived from env settings

	// 2. Load Global Settings from DB
	dbSettings, err := store.GetGlobalSettings(ctx, "global")
	if err != nil {
		var itemNotFound *types.ResourceNotFoundException
		if errors.As(err, &itemNotFound) {
			logEntry.Warn("Global settings not found in database, using environment defaults.")
			dbSettings = nil // Treat as empty settings
		} else {
			logEntry.WithError(err).Error("Failed to load global settings from database")
			// Decide if this is fatal or if we can proceed with env settings only
			// return nil, fmt.Errorf("failed to load DB settings: %w", err)
			dbSettings = nil // Proceed with caution
		}
	}

	// 3. Merge Global Settings
	globalConfig := mergeGlobalSettings(envSettings, dbSettings)
	if globalConfig.DebugMode {
		logger.SetLevel(logrus.DebugLevel) // Set debug level if enabled
		logEntry.Warn("Debug mode enabled")
	}

	// 4. Load Clients from DB
	clientsList, err := store.ListClients(ctx)
	if err != nil {
		logEntry.WithError(err).Error("Failed to load clients from database")
		return nil, fmt.Errorf("failed to load clients: %w", err)
	}
	clientsMap := make(map[string]Client)
	for _, c := range clientsList {
		clientsMap[c.Name] = c
	}
	logEntry.Infof("Loaded %d clients", len(clientsMap))

	developersList, err := store.ListDevelopers(ctx)
	if err != nil {
		logEntry.WithError(err).Error("Failed to load developers from database")
		return nil, fmt.Errorf("failed to load developers: %w", err)
	}
	developersMap := make(map[string]Developer)
	for _, d := range developersList {
		developersMap[d.Name] = d
	}
	logEntry.Infof("Loaded %d developers", len(developersMap))

	// 6. Assemble AppConfig
	appConf := &AppConfig{
		Global:     globalConfig,
		Clients:    clientsMap,
		Developers: developersMap,
		Store:      store,
		Logger:     logger,
	}

	logEntry.Info("Application configuration loaded successfully")
	return appConf, nil
}
