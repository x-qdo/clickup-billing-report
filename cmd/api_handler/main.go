package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"

	"github.com/x-qdo/clickup-billing-report/internal/auth"
	"github.com/x-qdo/clickup-billing-report/internal/config"
	"github.com/x-qdo/clickup-billing-report/internal/job"
	"github.com/x-qdo/clickup-billing-report/internal/report"
	"github.com/x-qdo/clickup-billing-report/internal/server"
	"github.com/x-qdo/clickup-billing-report/internal/storage"
)

var authenticator *auth.Authenticator
var reportService *report.Service
var jobService *job.Service
var appConf *config.AppConfig // Store loaded config
var awsCfg aws.Config

// Initialize config and services once during cold start
func init() {
	var err error
	ctx := context.Background()

	// Load AWS config first
	awsCfg, err = awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		panic(fmt.Sprintf("FATAL: Failed to load AWS config: %v", err))
	}

	// Create a temporary logger for init phase
	initLogger := logrus.New()
	initLogger.SetLevel(logrus.InfoLevel) // Adjust level as needed
	initLogger.SetFormatter(&logrus.JSONFormatter{})
	logEntry := initLogger.WithField("service", "api-handler-init")

	// Create the DynamoDB store instance
	store, err := storage.NewDynamoDBStore(awsCfg, initLogger)
	if err != nil {
		panic(fmt.Sprintf("FATAL: Failed to initialize DynamoDB store: %v", err))
	}

	// Load application config, injecting the store
	appConf, err = config.LoadConfig(ctx, store)
	if err != nil {
		logEntry.WithError(err).Error("Non-fatal error loading config during init")
		// We might still be able to proceed if only part of the config failed,
		// but critical failures (like missing secrets) should cause panic earlier.
		if appConf == nil {
			panic(fmt.Sprintf("FATAL: Failed to load critical config: %v", err))
		}
	}
	if appConf.Store == nil {
		panic("FATAL: DataStore is nil after config load")
	}

	// Initialize services with the loaded config
	authenticator = auth.NewAuthenticator(appConf)
	reportService = report.NewService(appConf)
	jobService = job.NewService(awsCfg, appConf.Store, appConf.Logger)
	appConf.Logger.Info("API handler initialized successfully")
}

// lambdaAdapter adapts the HTTP handler (router) for Lambda execution.
func lambdaAdapter(coreHandler http.Handler) func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		log := appConf.Logger.WithFields(logrus.Fields{
			"lambdaRequestId": event.RequestContext.RequestID,
			"httpMethod":      event.HTTPMethod,
			"path":            event.Path,
			"sourceIp":        event.RequestContext.Identity.SourceIP,
			"userAgent":       event.RequestContext.Identity.UserAgent,
		})
		log.Debug("Handling Lambda request via adapter")

		// Handle CORS preflight requests directly before routing
		if event.HTTPMethod == "OPTIONS" {
			log.Debug("Handling CORS preflight OPTIONS request")
			// Note: This assumes CORS headers are handled globally or by the specific handler later.
			// The server.OptionsResponse provides basic CORS headers. Adjust if needed.
			return server.OptionsResponse()
		}

		// Convert API Gateway event to http.Request
		r, err := server.NewRequestFromEvent(ctx, event)
		if err != nil {
			log.WithError(err).Error("Failed to create http.Request from event")
			return server.ErrorJSONResponse(http.StatusInternalServerError, "InternalError", "Failed to process request")
		}

		// Use the response writer adapter
		w := server.NewResponseWriter()

		// Serve the request using the main router
		coreHandler.ServeHTTP(w, r)

		// Convert the captured response to API Gateway format
		response := w.ToAPIGatewayProxyResponse()
		log.WithField("responseStatusCode", response.StatusCode).Info("Responding to Lambda request")
		return response, nil
	}
}

// --- Auth Handler Logic ---

// serveAuthRequest handles authentication related requests (/auth/*).
func serveAuthRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	log.Info("Auth handler logic processing request")

	// Routing based on the full path managed by the main mux
	switch r.URL.Path {
	case "/auth/clickup": // Start OAuth flow
		if r.Method != "GET" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		authURL := authenticator.StartAuth(w, r) // StartAuth sets cookies on 'w'
		log.WithField("redirect_url", authURL).Info("Redirecting user to ClickUp for authentication")

		// Set redirect header and status code directly on the ResponseWriter
		w.Header().Set("Location", authURL)
		w.WriteHeader(http.StatusFound) // 302 Redirect

	case "/auth/callback": // Handle OAuth callback
		if r.Method != "GET" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		err := authenticator.HandleCallback(w, r) // HandleCallback sets cookies on 'w'
		if err != nil {
			log.WithError(err).Error("OAuth callback failed")
			// Return a user-friendly error page or JSON
			http.Error(w, fmt.Sprintf("Authentication callback failed: %v", err), http.StatusInternalServerError)
			return
		}

		log.Info("OAuth callback successful, redirecting to reports page")
		redirectTarget := "/"
		http.Redirect(w, r, redirectTarget, http.StatusFound)

	case "/auth/logout": // Handle logout
		// Typically POST, but GET might be used for simple link-based logout
		if r.Method != "POST" && r.Method != "GET" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		err := authenticator.Logout(w, r) // Logout clears cookies on 'w'
		if err != nil {
			// Log error but usually proceed with redirect anyway
			log.WithError(err).Error("Error during logout process")
		}

		log.Info("User logged out, redirecting to home page")
		// Redirect user to the home/login page after logout
		// TODO: Make the redirect target configurable (e.g., from appConf.Global)
		redirectTarget := "/"
		//if appConf.Global.LogoutRedirectURL != "" {
		//	redirectTarget = appConf.Global.LogoutRedirectURL
		//}
		http.Redirect(w, r, redirectTarget, http.StatusFound) // Redirect to root or configured URL

	case "/auth/me": // Check authentication status
		if r.Method != "GET" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		setCorsHeaders(w)

		session, sessionID, err := authenticator.GetSessionFromRequest(r)
		if err != nil {
			log.WithError(err).Error("Failed to get session from request")
			respondWithJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"authenticated": false,
				"error":         "Invalid session",
			})
			return
		}

		if session == nil || sessionID == "" {
			log.Info("No valid session found")
			respondWithJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"authenticated": false,
			})
			return
		}

		log.WithField("user_id", session.UserID).Info("User authentication status checked")
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": true,
			"user": map[string]interface{}{
				"id":       session.UserID,
				"username": session.UserName,
			},
		})

	default:
		// This case should ideally not be reached if the main router is configured correctly
		log.WithField("actual_path", r.URL.Path).Warn("Auth handler received request for unexpected sub-path")
		http.NotFound(w, r)
	}
}

// serveDemoRequest handles demo report requests (/report/demo).
func serveDemoRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	log.Info("Demo handler logic processing request")

	// --- Authentication Check ---
	session, token, err := authenticator.GetSessionFromRequest(r)
	if err != nil {
		log.WithError(err).Error("Failed to get session from request")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "SessionError", Message: "Could not process session"})
		return
	}
	if session == nil || token == "" {
		log.Warn("User not authenticated or session invalid/expired for demo request")
		respondWithJSON(w, http.StatusUnauthorized, server.ErrorResponse{Error: "Unauthorized", Message: "Authentication required"})
		return
	}
	userLogger := log.WithFields(logrus.Fields{"user_id": session.UserID})
	userLogger.Info("User authenticated for demo request")
	// ---------------------------

	// Assuming demo report is generated via GET for simplicity
	if r.Method != "GET" {
		respondWithJSON(w, http.StatusMethodNotAllowed, server.ErrorResponse{Error: "MethodNotAllowed", Message: "Method Not Allowed"})
		return
	}

	userLogger.Info("Generating demo report (placeholder)")

	// TODO: Implement demo generation logic using demoService
	// 1. Call demo generation service (fetch tasks, format MD)
	//    mdContent, err := demoService.GenerateDemoMarkdown(r.Context(), token)
	//    if err != nil {
	//        userLogger.WithError(err).Error("Failed to generate demo markdown")
	//        respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "DemoGenerationFailed", Message: fmt.Sprintf("Error generating demo: %v", err)})
	//        return
	//    }
	// 2. (Optional) Call Wiki service
	// 3. (Optional) Call Slack service
	// 4. Format and return response (e.g., JSON with markdown content)

	// Placeholder response
	responsePayload := map[string]string{
		"message":       "Demo report generation not yet implemented.",
		"title":         "Demo Report (Placeholder)",
		"md":            "# Demo Report\n\n*   Task 1\n*   Task 2",
		"slack_message": "@here Demo time! (Placeholder)",
	}

	respondWithJSON(w, http.StatusOK, responsePayload) // Or http.StatusNotImplemented
}

// --- Report Handler Logic ---

// serveReportRequest handles report generation requests (/report/*, excluding /report/demo).
func serveReportRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	log.Info("Report handler logic processing request")

	// --- Authentication Check ---
	session, token, err := authenticator.GetSessionFromRequest(r)
	if err != nil {
		log.WithError(err).Error("Failed to get session from request")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "SessionError", Message: "Could not process session"})
		return
	}
	if session == nil || token == "" {
		log.Warn("User not authenticated or session invalid/expired for report request")
		respondWithJSON(w, http.StatusUnauthorized, server.ErrorResponse{Error: "Unauthorized", Message: "Authentication required"})
		return
	}
	userLogger := log.WithFields(logrus.Fields{"user_id": session.UserID}) // Add user ID to logs for this request
	userLogger.Info("User authenticated for report request")
	// ---------------------------

	// --- Routing based on path ---
	// Note: The main mux directs requests starting with /report/ here,
	// but we still need to differentiate between specific report types.
	// /report/demo is handled by serveDemoRequest.
	switch r.URL.Path {
	case "/report/timetrack":
		if r.Method == "POST" {
			handleGenerateTimeTrackReport(w, r, token, userLogger)
		} else {
			respondWithJSON(w, http.StatusMethodNotAllowed, server.ErrorResponse{Error: "MethodNotAllowed", Message: "Use POST for timetrack report"})
		}

	case "/report/billable":
		if r.Method == "POST" {
			handleGenerateBillableReport(w, r, token, userLogger)
		} else {
			respondWithJSON(w, http.StatusMethodNotAllowed, server.ErrorResponse{Error: "MethodNotAllowed", Message: "Use POST for billable report"})
		}

	default:
		log.WithField("actual_path", r.URL.Path).Warn("Report handler received request for unknown report path or method")
		http.NotFound(w, r)
	}
}

func serveClientsRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	log.Info("Client list request received")

	setCorsHeaders(w) // Set CORS headers early

	// Handle OPTIONS request for CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// --- Authentication Check ---
	session, _, err := authenticator.GetSessionFromRequest(r)
	if err != nil {
		log.WithError(err).Error("Failed to get session from request")
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "SessionError", "message": "Could not process session"})
		return
	}
	if session == nil {
		log.Warn("User not authenticated for client list request")
		respondWithJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized", "message": "Authentication required"})
		return
	}
	userLogger := log.WithFields(logrus.Fields{"user_id": session.UserID})
	userLogger.Info("User authenticated for client list request")
	// ---------------------------

	if r.Method != http.MethodGet {
		userLogger.Warn("Method not allowed for client list request")
		respondWithJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "MethodNotAllowed", "message": "Method Not Allowed, please use GET"})
		return
	}

	clients, err := appConf.Store.ListClients(r.Context())
	if err != nil {
		userLogger.WithError(err).Error("Failed to list clients from store")
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": "InternalError", "message": "Failed to retrieve client list"})
		return
	}

	if clients == nil {
		// Return an empty list instead of null if no clients are found.
		clients = []config.Client{}
	}

	userLogger.Infof("Successfully retrieved %d clients", len(clients))
	respondWithJSON(w, http.StatusOK, clients)
}

// --- Job Handler Logic ---

// JobCreateRequest represents the request body for creating a job.
type JobCreateRequest struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// serveJobRequest handles job-related requests.
func serveJobRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})

	setCorsHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Authentication check
	session, token, err := authenticator.GetSessionFromRequest(r)
	if err != nil {
		log.WithError(err).Error("Failed to get session from request")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "SessionError", Message: "Could not process session"})
		return
	}
	if session == nil || token == "" {
		log.Warn("User not authenticated for job request")
		respondWithJSON(w, http.StatusUnauthorized, server.ErrorResponse{Error: "Unauthorized", Message: "Authentication required"})
		return
	}

	userLogger := log.WithField("user_id", session.UserID)

	// Route based on path and method
	if r.URL.Path == "/api/jobs" && r.Method == http.MethodPost {
		handleCreateJob(w, r, token, session.UserID, userLogger)
		return
	}

	// Handle GET /api/jobs/{id}
	if strings.HasPrefix(r.URL.Path, "/api/jobs/") && r.Method == http.MethodGet {
		jobID := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
		if jobID == "" {
			respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidRequest", Message: "Job ID required"})
			return
		}
		handleGetJob(w, r, jobID, userLogger)
		return
	}

	http.NotFound(w, r)
}

func handleCreateJob(w http.ResponseWriter, r *http.Request, token, userID string, log *logrus.Entry) {
	log.Info("Handling create job request")
	ctx := r.Context()

	var req JobCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Failed to decode job request body")
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidRequest", Message: "Invalid request body"})
		return
	}

	// Validate job type
	if req.Type != config.JobTypeTimetrack && req.Type != config.JobTypeBillable {
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidJobType", Message: "Job type must be 'timetrack' or 'billable'"})
		return
	}

	// Add token to params for worker to use
	if req.Params == nil {
		req.Params = make(map[string]interface{})
	}
	req.Params["clickup_token"] = token

	// Marshal input to JSON
	inputJSON, err := json.Marshal(req.Params)
	if err != nil {
		log.WithError(err).Error("Failed to marshal job input")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "InternalError", Message: "Failed to create job"})
		return
	}

	// Create the job
	createdJob, err := jobService.CreateJob(ctx, req.Type, string(inputJSON), userID)
	if err != nil {
		log.WithError(err).Error("Failed to create job")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "JobCreationFailed", Message: fmt.Sprintf("Failed to create job: %v", err)})
		return
	}

	log.WithField("job_id", createdJob.JobID).Info("Job created successfully")
	respondWithJSON(w, http.StatusAccepted, map[string]interface{}{
		"job_id": createdJob.JobID,
		"status": createdJob.Status,
	})
}

func handleGetJob(w http.ResponseWriter, r *http.Request, jobID string, log *logrus.Entry) {
	log.WithField("job_id", jobID).Info("Handling get job request")
	ctx := r.Context()

	jobData, err := jobService.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			respondWithJSON(w, http.StatusNotFound, server.ErrorResponse{Error: "NotFound", Message: "Job not found"})
			return
		}
		log.WithError(err).Error("Failed to get job")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "InternalError", Message: "Failed to get job"})
		return
	}

	// Build response based on job status
	response := map[string]interface{}{
		"job_id":     jobData.JobID,
		"type":       jobData.Type,
		"status":     jobData.Status,
		"created_at": jobData.CreatedAt,
		"updated_at": jobData.UpdatedAt,
	}

	if jobData.Status == config.JobStatusCompleted && jobData.Output != "" {
		var result interface{}
		if err := json.Unmarshal([]byte(jobData.Output), &result); err == nil {
			response["result"] = result
		} else {
			response["result"] = jobData.Output
		}
	}

	if jobData.Status == config.JobStatusFailed && jobData.Error != "" {
		response["error"] = jobData.Error
	}

	respondWithJSON(w, http.StatusOK, response)
}

// parseHTTPFormParams extracts form parameters from an http.Request.
// It handles both URL query parameters and form-urlencoded bodies.
func parseHTTPFormParams(r *http.Request, log *logrus.Entry) (url.Values, error) {
	// Parse form data from body (handles POST, PUT, PATCH)
	// Max memory limit for parsing form data (e.g., 10MB)
	// Consider using r.ParseMultipartForm if you expect multipart/form-data
	if err := r.ParseForm(); err != nil {
		log.WithError(err).Error("Failed to parse form data from request")
		return nil, fmt.Errorf("invalid form data")
	}

	// r.Form contains both query parameters and form body parameters
	// r.PostForm contains only form body parameters
	// Use r.Form for simplicity here, as it covers both GET (query) and POST (body)
	log.WithField("form_params", r.Form).Debug("Parsed form parameters")
	return r.Form, nil
}

// setCorsHeaders sets common CORS headers. Adjust origin for production.
func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie, Content-Disposition")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE") // Add methods as needed
}

// respondWithJSON is a helper for sending JSON responses.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal JSON response payload")
		w.Header().Set("Content-Type", "application/json")
		setCorsHeaders(w)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal Server Error", "message": "Failed to generate response"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	setCorsHeaders(w)
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		logrus.WithError(err).Error("Failed to write JSON response body")
	}
}

// respondWithFile sends a binary file response (e.g., Excel).
func respondWithFile(w http.ResponseWriter, code int, data []byte, contentType, filename string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if filename != "" {
		// Suggest filename for download
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}
	setCorsHeaders(w) // Also set CORS for file downloads if needed by frontend
	w.WriteHeader(code)
	_, err := w.Write(data)
	if err != nil {
		logrus.WithError(err).Error("Failed to write file response body")
	}
}

// handleGenerateTimeTrackReport handles the POST request to generate the time tracking report.
func handleGenerateTimeTrackReport(w http.ResponseWriter, r *http.Request, token string, log *logrus.Entry) {
	log.Info("Handling generate time track report request")
	ctx := r.Context()

	params, err := parseHTTPFormParams(r, log)
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidRequest", Message: err.Error()})
		return
	}

	reportDateStr := params.Get("report_date") // Expected format: "YYYY-MM"
	refreshBillableStr := params.Get("refresh_billable")
	outputFormat := params.Get("format") // "excel" or empty/other for JSON

	if reportDateStr == "" {
		log.Warn("Missing 'report_date' parameter")
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "MissingParam", Message: "Missing 'report_date' parameter (YYYY-MM)"})
		return
	}

	selectedMonth, err := time.Parse("2006-01", reportDateStr)
	if err != nil {
		log.WithError(err).Errorf("Invalid 'report_date' format: %s", reportDateStr)
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidParam", Message: "Invalid 'report_date' format, use YYYY-MM"})
		return
	}

	refreshBillable := false
	if refreshBillableStr == "on" || refreshBillableStr == "true" {
		refreshBillable = true
		log.Info("Refresh Billable Hours flag is set for time track report")
	}

	input := report.TimeTrackingInput{
		SelectedMonth:   selectedMonth,
		RefreshBillable: refreshBillable,
		ClickUpToken:    token,
	}

	reportOutput, err := reportService.GenerateTimeTrackingReport(ctx, input)
	if err != nil {
		log.WithError(err).Error("Failed to generate time tracking report")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ReportGenerationFailed", Message: fmt.Sprintf("Error generating time tracking report: %v", err)})
		return
	}

	// --- Respond based on format ---
	if outputFormat == "excel" {
		log.Info("Generating Excel format for time tracking report")
		excelFile, err := report.GenerateTimeTrackingExcel(reportOutput, selectedMonth)
		if err != nil {
			log.WithError(err).Error("Failed to generate time tracking Excel file")
			respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ExcelGenerationFailed", Message: fmt.Sprintf("Error generating Excel report: %v", err)})
			return
		}

		// Save Excel to buffer
		var buf bytes.Buffer
		if err := excelFile.Write(&buf); err != nil {
			log.WithError(err).Error("Failed to write Excel file to buffer")
			respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ExcelWriteFailed", Message: "Failed to write Excel file"})
			return
		}

		filename := fmt.Sprintf("time_tracking_report_%s.xlsx", selectedMonth.Format("2006-01"))
		contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		respondWithFile(w, http.StatusOK, buf.Bytes(), contentType, filename)
		log.Info("Successfully sent time tracking report as Excel file")

	} else {
		log.Info("Successfully generated time tracking report (JSON)")
		respondWithJSON(w, http.StatusOK, reportOutput)
	}
}

// handleGenerateBillableReport handles the POST request for the billable report.
func handleGenerateBillableReport(w http.ResponseWriter, r *http.Request, token string, log *logrus.Entry) {
	log.Info("Handling generate billable report request")
	ctx := r.Context()

	params, err := parseHTTPFormParams(r, log)
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidRequest", Message: err.Error()})
		return
	}

	clientName := params.Get("client_name")
	refreshInvoicedStr := params.Get("refresh_invoiced")
	outputFormat := params.Get("format") // "excel" or empty/other for JSON

	if clientName == "" {
		log.Warn("Missing 'client_name' parameter")
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "MissingParam", Message: "Missing 'client_name' parameter"})
		return
	}

	refreshInvoiced := false
	if refreshInvoicedStr == "on" || refreshInvoicedStr == "true" {
		refreshInvoiced = true
		log.Info("Refresh Invoiced Hours flag is set for billable report")
	}

	input := report.BillableReportInput{
		ClientName:      clientName, // Use client name
		RefreshInvoiced: refreshInvoiced,
		ClickUpToken:    token,
	}

	reportOutput, err := reportService.GenerateBillableReport(ctx, input)
	if err != nil {
		log.WithError(err).Error("Failed to generate billable report")
		// Check for specific error types if needed (e.g., client not found)
		var storageErr *config.StorageError
		if errors.As(err, &storageErr) { // Example: Check if it's a known config error type
			respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "ConfigError", Message: err.Error()})
		} else {
			respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ReportGenerationFailed", Message: fmt.Sprintf("Error generating billable report: %v", err)})
		}
		return
	}

	// --- Respond based on format ---
	if outputFormat == "excel" {
		log.Info("Generating Excel format for billable report")
		excelFile, err := report.GenerateBillableExcel(reportOutput, clientName)
		if err != nil {
			log.WithError(err).Error("Failed to generate billable Excel file")
			respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ExcelGenerationFailed", Message: fmt.Sprintf("Error generating Excel report: %v", err)})
			return
		}

		// Save Excel to buffer
		var buf bytes.Buffer
		if err := excelFile.Write(&buf); err != nil {
			log.WithError(err).Error("Failed to write Excel file to buffer")
			respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ExcelWriteFailed", Message: "Failed to write Excel file"})
			return
		}

		// Sanitize client name for filename
		safeClientName := url.PathEscape(clientName)                                                                                          // Basic sanitization
		safeClientName = Mreplace(safeClientName, "/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_") // Replace common invalid chars

		filename := fmt.Sprintf("billable_report_%s_%s.xlsx", safeClientName, time.Now().Format("20060102"))
		contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		respondWithFile(w, http.StatusOK, buf.Bytes(), contentType, filename)
		log.Info("Successfully sent billable report as Excel file")

	} else {
		log.Info("Successfully generated billable report (JSON)")
		respondWithJSON(w, http.StatusOK, reportOutput)
	}
}

// Mreplace replaces multiple substrings in a string.
func Mreplace(s string, replaces ...string) string {
	if len(replaces)%2 != 0 {
		panic("Mreplace requires pairs of old/new strings")
	}
	for i := 0; i < len(replaces); i += 2 {
		s = strings.ReplaceAll(s, replaces[i], replaces[i+1])
	}
	return s
}

// --- Main Entry Point ---

func main() {
	// Ensure config is loaded (init should handle this, but double-check)
	if appConf == nil {
		panic("FATAL: AppConfig is nil at main execution")
	}
	log := appConf.Logger

	// --- Create the main router ---
	mux := http.NewServeMux()

	// Register Auth routes
	mux.HandleFunc("/auth/clickup", serveAuthRequest)
	mux.HandleFunc("/auth/callback", serveAuthRequest)
	mux.HandleFunc("/auth/logout", serveAuthRequest)
	mux.HandleFunc("/auth/me", serveAuthRequest)

	// Register Report routes
	mux.HandleFunc("/report/demo", serveDemoRequest)        // Specific demo report
	mux.HandleFunc("/report/timetrack", serveReportRequest) // Handles POST for time track (JSON/Excel)
	mux.HandleFunc("/report/billable", serveReportRequest)  // Handles POST for billable (JSON/Excel)

	// Register API routes
	mux.HandleFunc("/api/clients", serveClientsRequest)

	// Register Job routes (async report processing)
	mux.HandleFunc("/api/jobs", serveJobRequest)
	mux.HandleFunc("/api/jobs/", serveJobRequest)

	// --- Serve Static Files (Frontend) ---
	// Serve static assets from /dist/assets for URLs starting with /assets/
	// e.g. /assets/index.js will serve /dist/assets/index.js from the filesystem
	assetsFs := http.FileServer(http.Dir("/dist/assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", assetsFs))

	// Add a root handler for basic health check or info page
	// This also acts as a catch-all for any routes not explicitly defined above.
	// If a request is made to / (root), it serves the index.html from the dist folder.
	// Otherwise, for any other unhandled path, it returns a 404.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If running in debug mode and accessing root, serve index.html from dist
		// This is a common pattern for SPAs.
		// Ensure your frontend router handles client-side routing appropriately.
		http.ServeFile(w, r, "/dist/index.html")
		log.Debug("Root path '/' accessed, serving ./dist/index.html")
		return
	})

	// --- Start Server or Lambda ---
	if appConf.Global.DebugMode {
		// --- HTTP Server Mode ---
		addr := ":" + appConf.Global.DebugPort
		log.Infof("Starting API HTTP server in debug mode on %s", addr)

		server := &http.Server{
			Addr:    addr,
			Handler: mux, // Use the main router
			// Add timeouts for production readiness
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 90 * time.Second, // Increased write timeout for potentially larger Excel generation
			IdleTimeout:  120 * time.Second,
		}

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("HTTP server ListenAndServe error")
		}
		log.Info("HTTP server stopped gracefully.")

	} else {
		// --- Lambda Mode ---
		log.Info("Starting API Lambda handler")
		// Wrap the main router with the Lambda adapter
		lambdaHandler := lambdaAdapter(mux)
		lambda.Start(lambdaHandler)
	}
}
