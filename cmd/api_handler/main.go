package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"

	"github.com/x-qdo/clickup-billing-report/internal/auth"
	"github.com/x-qdo/clickup-billing-report/internal/config"
	"github.com/x-qdo/clickup-billing-report/internal/report"
	"github.com/x-qdo/clickup-billing-report/internal/server"
	"github.com/x-qdo/clickup-billing-report/internal/storage"
)

var authenticator *auth.Authenticator
var reportService *report.Service
var appConf *config.AppConfig // Store loaded config
// var demoService *demo.Service // Placeholder for demo service

// Initialize config and services once during cold start
func init() {
	var err error
	ctx := context.Background()

	// Load AWS config first
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
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
	// demoService = demo.NewService(appConf) // Uncomment when demo service exists
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
		// Redirect user to the main application page after successful login
		// TODO: Make the redirect target configurable (e.g., from appConf.Global)
		redirectTarget := "/report/timetrack" // Default redirect target
		//if appConf.Global.AuthSuccessRedirectURL != "" {
		//	redirectTarget = appConf.Global.AuthSuccessRedirectURL
		//}
		http.Redirect(w, r, redirectTarget, http.StatusFound) // Use standard http.Redirect

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
		http.NotFound(w, r) // Or respondWithJSON(w, http.StatusNotFound, ...)
	}
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

// respondWithJSON is a helper for sending JSON responses in the HTTP server context.
// It also handles basic CORS headers, which might be needed for debug mode or specific setups.
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		// Log the error internally
		logrus.WithError(err).Error("Failed to marshal JSON response payload")
		// Send a generic error response back to the client
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for production
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "Internal Server Error", "message": "Failed to generate response"}`)) // Ignore write error here
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Add CORS headers - adjust origin and headers for production environments
	w.Header().Set("Access-Control-Allow-Origin", "*") // Example: Use appConf.Global.AllowedOrigin
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie") // Add any custom headers needed

	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		// Log write error, but headers and status are already sent.
		logrus.WithError(err).Error("Failed to write JSON response body")
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
		// Check for specific error types if needed (e.g., ClickUp API errors)
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ReportGenerationFailed", Message: fmt.Sprintf("Error generating time tracking report: %v", err)})
		return
	}

	log.Info("Successfully generated time tracking report")
	respondWithJSON(w, http.StatusOK, reportOutput)
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

	listID := params.Get("list_id")
	refreshInvoicedStr := params.Get("refresh_invoiced")

	if listID == "" {
		log.Warn("Missing 'list_id' parameter")
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "MissingParam", Message: "Missing 'list_id' parameter"})
		return
	}

	refreshInvoiced := false
	if refreshInvoicedStr == "on" || refreshInvoicedStr == "true" {
		refreshInvoiced = true
		log.Info("Refresh Invoiced Hours flag is set for billable report")
	}

	input := report.BillableReportInput{
		ListID:          listID,
		RefreshInvoiced: refreshInvoiced,
		ClickUpToken:    token,
	}

	reportOutput, err := reportService.GenerateBillableReport(ctx, input)
	if err != nil {
		log.WithError(err).Error("Failed to generate billable report")
		// Check for specific error types if needed
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ReportGenerationFailed", Message: fmt.Sprintf("Error generating billable report: %v", err)})
		return
	}

	log.Info("Successfully generated billable report")
	respondWithJSON(w, http.StatusOK, reportOutput)
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
	// Note: HandleFunc registers for the exact path. Use Handle for prefixes if needed,
	// but for these specific auth actions, exact paths are fine.
	mux.HandleFunc("/auth/clickup", serveAuthRequest)
	mux.HandleFunc("/auth/callback", serveAuthRequest)
	mux.HandleFunc("/auth/logout", serveAuthRequest)

	// Register Report routes
	mux.HandleFunc("/report/demo", serveDemoRequest)        // Specific demo report
	mux.HandleFunc("/report/timetrack", serveReportRequest) // Other reports handled by serveReportRequest
	mux.HandleFunc("/report/billable", serveReportRequest)  // Other reports handled by serveReportRequest
	// Add more /report/* routes here if needed, pointing to serveReportRequest or new specific funcs

	// Add a root handler for basic health check or info page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Prevent accidental matches for paths not explicitly handled
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		log.Debug("Root path '/' accessed")
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "API Handler Running")
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
			WriteTimeout: 60 * time.Second, // Longer write timeout for report generation
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
