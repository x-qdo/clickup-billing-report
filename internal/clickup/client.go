package clickup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	apiBaseURL = "https://api.clickup.com/api/v2"
)

// Client manages communication with the ClickUp API.
type Client struct {
	httpClient *http.Client // Can be an oauth2.Client
	baseURL    *url.URL
	log        *logrus.Entry
	token      string // User-specific access token (needed if httpClient isn't oauth2 client)

	// Cache for custom fields (per list)
	customFieldsCache sync.Map // map[string]map[string]*CustomField -> map[listID]map[fieldName]fieldDetails
}

// CustomField holds details about a ClickUp custom field definition.
type CustomField struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	TypeConfig struct {
		// Define specific type configs if needed, e.g., for dropdowns
		Options []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
			Order int    `json:"orderindex"`
		} `json:"options"`
	} `json:"type_config"`
	DateCreated    string `json:"date_created"`
	HideFromGuests bool   `json:"hide_from_guests"`
	Required       bool   `json:"required"`
}

// CustomFieldsResponse is the structure for the API response when fetching field definitions.
type CustomFieldsResponse struct {
	Fields []CustomField `json:"fields"`
}

// NewClient creates a new ClickUp API client.
// Requires an http.Client (often configured with OAuth2).
func NewClient(httpClient *http.Client, logger *logrus.Logger, userToken string) (*Client, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: time.Second * 20} // Increased timeout
	}
	baseURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	logEntry := logger.WithField("component", "ClickUpClient")

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		log:        logEntry,
		token:      userToken, // Store the user-specific token if provided
	}, nil
}

// GetAuthenticatedUserClient creates a client using the user's OAuth token.
func GetAuthenticatedUserClient(ctx context.Context, oauthConf *oauth2.Config, token *oauth2.Token, logger *logrus.Logger) (*Client, error) {
	httpClient := oauthConf.Client(ctx, token)
	// Pass empty token string as httpClient handles auth
	return NewClient(httpClient, logger, "")
}

// --- Request Helper ---
func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.baseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Authorization header ONLY if not using an oauth2 http.Client (token is provided)
	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	c.log.WithFields(logrus.Fields{
		"method": req.Method,
		"url":    req.URL.String(),
	}).Debug("Executing ClickUp API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		default:
		}
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check for API errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(resp.Body)
		c.log.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"url":         req.URL.String(),
			"error_body":  string(errorBody),
		}).Error("ClickUp API request failed")

		// Try to parse ClickUp specific error
		var apiErr ErrorResponse
		if json.Unmarshal(errorBody, &apiErr) == nil && apiErr.Error != "" {
			return resp, fmt.Errorf("api error: %s (status code: %d, ecode: %s)", apiErr.Error, resp.StatusCode, apiErr.ErrorCode)
		}
		// Generic HTTP error
		return resp, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode response body if v is provided
	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
		} else {
			decErr := json.NewDecoder(resp.Body).Decode(v)
			if decErr == io.EOF {
				decErr = nil // ignore EOF errors caused by empty response body
			}
			if decErr != nil {
				err = fmt.Errorf("failed to decode response body: %w", decErr)
			}
		}
		if err != nil {
			return resp, err // Return response even on decode error for potential inspection
		}
	}

	c.log.WithField("status_code", resp.StatusCode).Debug("ClickUp API request successful")
	return resp, nil
}

// --- Custom Field Handling ---

// getCustomFieldsMapForList retrieves or loads custom fields for a specific list ID.
func (c *Client) getCustomFieldsMapForList(ctx context.Context, listID string) (map[string]*CustomField, error) {
	cacheKey := listID
	if cachedFields, ok := c.customFieldsCache.Load(cacheKey); ok {
		if fieldsMap, ok := cachedFields.(map[string]*CustomField); ok {
			c.log.WithField("list_id", listID).Debug("Custom fields cache hit")
			return fieldsMap, nil
		}
	}

	c.log.WithField("list_id", listID).Info("Loading custom fields from ClickUp API")
	fieldsMap, err := c.loadCustomFieldsFromAPI(ctx, listID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.customFieldsCache.Store(cacheKey, fieldsMap)
	c.log.WithField("list_id", listID).Debug("Custom fields stored in cache")
	return fieldsMap, nil
}

// loadCustomFieldsFromAPI fetches custom field definitions directly from the ClickUp API.
func (c *Client) loadCustomFieldsFromAPI(ctx context.Context, listID string) (map[string]*CustomField, error) {
	path := fmt.Sprintf("/list/%s/field", listID)
	req, err := c.newRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create custom fields request: %w", err)
	}

	var fieldsResp CustomFieldsResponse
	_, err = c.do(req, &fieldsResp)
	if err != nil {
		return nil, fmt.Errorf("failed to execute/decode custom fields request: %w", err)
	}

	fieldsMap := make(map[string]*CustomField)
	for i := range fieldsResp.Fields {
		field := &fieldsResp.Fields[i] // Take pointer to avoid loop variable capture issues
		fieldsMap[field.Name] = field
		c.log.WithFields(logrus.Fields{
			"list_id": listID,
			"id":      field.ID,
			"name":    field.Name,
			"type":    field.Type,
		}).Debug("Loaded ClickUp custom field definition")
	}

	return fieldsMap, nil
}

// GetCustomFieldID retrieves the ID of a custom field by its name for a given list.
func (c *Client) GetCustomFieldID(ctx context.Context, listID, fieldName string) (string, error) {
	fieldsMap, err := c.getCustomFieldsMapForList(ctx, listID)
	if err != nil {
		return "", fmt.Errorf("failed to get custom fields for list %s: %w", listID, err)
	}

	field, ok := fieldsMap[fieldName]
	if !ok {
		c.log.WithFields(logrus.Fields{"list_id": listID, "field_name": fieldName}).Error("Custom field definition not found by name")
		return "", fmt.Errorf("custom field definition '%s' not found in list '%s'", fieldName, listID)
	}

	return field.ID, nil
}

// --- API Methods ---

// GetUser fetches details for the authenticated user.
func (c *Client) GetUser(ctx context.Context) (*User, error) {
	c.log.Info("Fetching authenticated user details")
	req, err := c.newRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}

	var userResp UserResponse
	_, err = c.do(req, &userResp)
	if err != nil {
		return nil, err
	}

	return &userResp.User, nil
}

// GetTasksOptions defines parameters for fetching tasks.
type GetTasksOptions struct {
	IncludeClosed bool
	Subtasks      bool
	Assignees     []string // List of user IDs
	// Add other filters like Statuses, Tags, DueDateGt, etc. if needed
}

// GetTasks fetches tasks from a list, handling pagination.
func (c *Client) GetTasks(ctx context.Context, listID string, opts *GetTasksOptions) ([]Task, error) {
	var allTasks []Task
	page := 0
	lastPage := false

	for !lastPage {
		path := fmt.Sprintf("/list/%s/task", listID)
		params := url.Values{}
		params.Set("page", strconv.Itoa(page))
		if opts != nil {
			params.Set("include_closed", strconv.FormatBool(opts.IncludeClosed))
			params.Set("subtasks", strconv.FormatBool(opts.Subtasks))
			if len(opts.Assignees) > 0 {
				// ClickUp expects assignees[]=1&assignees[]=2 format
				for _, id := range opts.Assignees {
					params.Add("assignees[]", id)
				}
			}
			// Add other options here
		}

		fullPath := path + "?" + params.Encode()
		c.log.WithFields(logrus.Fields{"list_id": listID, "page": page, "params": params}).Info("Fetching tasks page")

		req, err := c.newRequest(ctx, "GET", fullPath, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create tasks request for page %d: %w", page, err)
		}

		var tasksResp TasksResponse
		_, err = c.do(req, &tasksResp)
		if err != nil {
			return nil, fmt.Errorf("failed to execute/decode tasks request for page %d: %w", page, err)
		}

		allTasks = append(allTasks, tasksResp.Tasks...)
		lastPage = tasksResp.LastPage
		page++

		// Add a small delay to avoid rate limiting, especially if fetching many pages
		if !lastPage {
			time.Sleep(200 * time.Millisecond)
		}
	}

	c.log.WithFields(logrus.Fields{"list_id": listID, "total_tasks": len(allTasks)}).Info("Finished fetching tasks")
	return allTasks, nil
}

// GetTimeEntriesOptions defines parameters for fetching time entries.
type GetTimeEntriesOptions struct {
	StartDate            time.Time // Use time.Time for clarity
	EndDate              time.Time
	AssigneeIDs          []string
	IncludeTaskTags      bool
	IncludeLocationNames bool
	SpaceID              string
	FolderID             string
	ListID               string
	TaskID               string
}

// GetTimeEntries fetches time entries for a team, handling pagination (if API supports it).
// Note: ClickUp's time entry endpoint pagination is unclear/maybe non-existent. Fetch all in one go for now.
func (c *Client) GetTimeEntries(ctx context.Context, teamID string, opts *GetTimeEntriesOptions) ([]TimeEntry, error) {
	path := fmt.Sprintf("/team/%s/time_entries", teamID)
	params := url.Values{}

	if opts != nil {
		if !opts.StartDate.IsZero() {
			params.Set("start_date", strconv.FormatInt(opts.StartDate.UnixMilli(), 10))
		}
		if !opts.EndDate.IsZero() {
			params.Set("end_date", strconv.FormatInt(opts.EndDate.UnixMilli(), 10))
		}
		if len(opts.AssigneeIDs) > 0 {
			params.Set("assignee", strings.Join(opts.AssigneeIDs, ",")) // Comma-separated string
		}
		params.Set("include_task_tags", strconv.FormatBool(opts.IncludeTaskTags))
		params.Set("include_location_names", strconv.FormatBool(opts.IncludeLocationNames))
		if opts.SpaceID != "" {
			params.Set("space_id", opts.SpaceID)
		}
		if opts.FolderID != "" {
			params.Set("folder_id", opts.FolderID)
		}
		if opts.ListID != "" {
			params.Set("list_id", opts.ListID)
		}
		if opts.TaskID != "" {
			params.Set("task_id", opts.TaskID)
		}
	}

	fullPath := path + "?" + params.Encode()
	c.log.WithFields(logrus.Fields{"team_id": teamID, "params": params}).Info("Fetching time entries")

	req, err := c.newRequest(ctx, "GET", fullPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create time entries request: %w", err)
	}

	var timeResp TimeEntriesResponse
	_, err = c.do(req, &timeResp)
	if err != nil {
		return nil, fmt.Errorf("failed to execute/decode time entries request: %w", err)
	}

	c.log.WithFields(logrus.Fields{"team_id": teamID, "count": len(timeResp.Data)}).Info("Finished fetching time entries")
	return timeResp.Data, nil
}

// UpdateTaskCustomField updates a custom field value for a task.
// The `value` should be the appropriate type for the custom field (e.g., float64 for number, string for text).
func (c *Client) UpdateTaskCustomField(ctx context.Context, taskID, fieldID string, value interface{}) error {
	path := fmt.Sprintf("/task/%s/field/%s", taskID, fieldID)
	c.log.WithFields(logrus.Fields{"task_id": taskID, "field_id": fieldID, "value": value}).Info("Updating task custom field")

	// ClickUp expects numbers as numbers, not strings, in the JSON payload
	requestBody := UpdateTaskFieldRequest{Value: value}

	req, err := c.newRequest(ctx, "POST", path, requestBody)
	if err != nil {
		return fmt.Errorf("failed to create update custom field request: %w", err)
	}

	_, err = c.do(req, nil) // No response body expected on success
	if err != nil {
		return fmt.Errorf("failed to execute update custom field request: %w", err)
	}

	c.log.WithFields(logrus.Fields{"task_id": taskID, "field_id": fieldID}).Info("Task custom field updated successfully")
	return nil
}

// GetSpaces fetches spaces for a team.
// func (c *Client) GetSpaces(ctx context.Context, teamID string) ( /* SpaceResponse, */ error) {
// 	c.log.WithField("team_id", teamID).Info("Fetching spaces")
// 	// TODO: Implement API call
// 	return fmt.Errorf("GetSpaces not implemented")
// }
