package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	aws_config "github.com/aws/a
	"github.com/sirupsen/logrus"
	"github.com/your-org/clickup-reporter/internal/auth"
	"github.com/your-org/clickup-reporter/internal/config"
	"github.com/your-org/clickup-reporter/internal/report"
	"github.com/your-org/clickup-reporter/internal/server"
	"github.com/your-org/clickup-reporter/internal/storage"
)

var authenticator *auth.Authenticator
var reportService *report.Service
var appConf *config.AppConfig // Store loaded config

// Initialize config, authenticator, and service once during cold start
func init() {
	var err error
	ctx := context.Background()

	// Load AWS config first
	awsCfg, err := aws_config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(fmt.Sprintf("FATAL: Failed to load AWS config: %v", err))
	}

	// Create a temporary logger for init phase
	initLogger := logrus.New()
	initLogger.SetLevel(logrus.InfoLevel)
	initLogger.SetFormatter(&logrus.JSONFormatter{})
	logEntry := initLogger.WithField("service", "report-handler-init")

	// Create the DynamoDB store instance
	store, err := storage.NewDynamoDBStore(awsCfg, initLogger)
	if err != nil {
		panic(fmt.Sprintf("FATAL: Failed to initialize DynamoDB store: %v", err))
	}

	// Load application config, injecting the store
	appConf, err = config.LoadConfig(ctx, store)
	if err != nil {
		logEntry.WithError(err).Error("Non-fatal error loading config during init")
		if appConf == nil {
			panic(fmt.Sprintf("FATAL: Failed to load config: %v", err))
		}
	}
	if appConf.Store == nil {
		panic("FATAL: DataStore is nil after config load")
	}

	// Initialize services with the loaded config
	authenticator = auth.NewAuthenticator(appConf)
	reportService = report.NewService(appConf)
	appConf.Logger.Info("Report handler initialized successfully")
}

// lambdaAdapter adapts the HTTP handler for Lambda execution.
func lambdaAdapter(coreHandler http.HandlerFunc) func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		log := appConf.Logger.WithFields(logrus.Fields{
			"lambdaRequestId": event.RequestContext.RequestID,
			"httpMethod":      event.HTTPMethod,
			"path":            event.Path,
		})
		log.Info("Handling Lambda request via adapter")

		// Handle CORS preflight requests directly
		if event.HTTPMethod == "OPTIONS" {
			log.Debug("Handling CORS preflight OPTIONS request")
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

		// Call the core HTTP handler
		coreHandler(w, r)

		// Convert the captured response to API Gateway format
		response := w.ToAPIGatewayProxyResponse()
		log.WithField("responseStatusCode", response.StatusCode).Info("Responding to Lambda request")
		return response, nil
	}
}

// serveReportRequest is the core HTTP handler logic.
func serveReportRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	log.Info("Report handler core logic processing request")

	// --- Authentication Check ---
	session, token, err := authenticator.GetSessionFromRequest(r)
	if err != nil {
		log.WithError(err).Error("Failed to get session from request")
		http.Error(w, "Could not process session", http.StatusInternalServerError)
		return
	}
	if session == nil || token == "" {
		log.Warn("User not authenticated or session invalid/expired")
		// Redirect to login or return 401
		// For API endpoints, 401 is more appropriate
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	userLogger := log.WithFields(logrus.Fields{"user_id": session.UserID}) // Add user ID to logs for this request
	userLogger.Info("User authenticated")
	// ---------------------------

	// --- Routing ---
	switch {
	case strings.HasSuffix(r.URL.Path, "/report/timetrack") && r.Method == "POST":
		handleGenerateTimeTrackReport(w, r, token, userLogger)

	case strings.HasSuffix(r.URL.Path, "/report/billable") && r.Method == "POST":
		handleGenerateBillableReport(w, r, token, userLogger)

	default:
		log.Warn("Report handler received request for unknown path or method")
		http.NotFound(w, r)
	}
}

// parseHTTPFormParams extracts form parameters from an http.Request.
// It handles both URL query parameters and form-urlencoded bodies.
func parseHTTPFormParams(r *http.Request, log *logrus.Entry) (url.Values, error) {
	// Parse form data from body (handles POST, PUT, PATCH)
	// Max memory limit for parsing form data (e.g., 10MB)
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
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal JSON response payload")
		http.Error(w, `{"error": "Internal Server Error", "message": "Failed to create response"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Add CORS headers if needed for debug mode
	w.Header().Set("Access-Control-Allow-Origin", "*") // Be more specific in production
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie")

	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		logrus.WithError(err).Error("Failed to write JSON response body")
	}
}

// handleGenerateTimeTrackReport handles the POST request to generate the time tracking report (HTTP context).
func handleGenerateTimeTrackReport(w http.ResponseWriter, r *http.Request, token string, log *logrus.Entry) {
	log.Info("Handling generate time track report request (HTTP)")
	ctx := r.Context()

	params, err := parseHTTPFormParams(r, log)
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidRequest", Message: err.Error()})
		return
	}

	reportDateStr := params.Get("report_date") // Expected format: "YYYY-MM"
	refreshBillableStr := params.Get("refresh_billable")

	if reportDateStr == "" {
		log.Error("Missing 'report_date' parameter")
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
		log.Info("Refresh Billable Hours flag is set")
	}

	input := report.TimeTrackingInput{
		SelectedMonth:   selectedMonth,
		RefreshBillable: refreshBillable,
		ClickUpToken:    token,
	}

	reportOutput, err := reportService.GenerateTimeTrackingReport(ctx, input)
	if err != nil {
		log.WithError(err).Error("Failed to generate time tracking report")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ReportGenerationFailed", Message: fmt.Sprintf("Error generating report: %v", err)})
		return
	}

	log.Info("Successfully generated time tracking report")
	respondWithJSON(w, http.StatusOK, reportOutput)
}

// handleGenerateBillableReport handles the POST request for the billable report (HTTP context).
func handleGenerateBillableReport(w http.ResponseWriter, r *http.Request, token string, log *logrus.Entry) {
	log.Info("Handling generate billable report request (HTTP)")
	ctx := r.Context()

	params, err := parseHTTPFormParams(r, log)
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "InvalidRequest", Message: err.Error()})
		return
	}

	listID := params.Get("list_id")
	refreshInvoicedStr := params.Get("refresh_invoiced")

	if listID == "" {
		log.Error("Missing 'list_id' parameter")
		respondWithJSON(w, http.StatusBadRequest, server.ErrorResponse{Error: "MissingParam", Message: "Missing 'list_id' parameter"})
		return
	}

	refreshInvoiced := false
	if refreshInvoicedStr == "on" || refreshInvoicedStr == "true" {
		refreshInvoiced = true
		log.Info("Refresh Invoiced Hours flag is set")
	}

	input := report.BillableReportInput{
		ListID:          listID,
		RefreshInvoiced: refreshInvoiced,
		ClickUpToken:    token,
	}

	reportOutput, err := reportService.GenerateBillableReport(ctx, input)
	if err != nil {
		log.WithError(err).Error("Failed to generate billable report")
		respondWithJSON(w, http.StatusInternalServerError, server.ErrorResponse{Error: "ReportGenerationFailed", Message: fmt.Sprintf("Error generating billable report: %v", err)})
		return
	}

	log.Info("Successfully generated billable report")
	respondWithJSON(w, http.StatusOK, reportOutput)
}

func main() {
	if appConf == nil {
		panic("FATAL: AppConfig is nil at main execution")
	}
	log := appConf.Logger

	if appConf.Global.DebugMode {
		// --- HTTP Server Mode ---
		addr := ":" + appConf.Global.DebugPort
		log.Infof("Starting Report HTTP server in debug mode on %s", addr)

		mux := http.NewServeMux()
		// Register the core handler for report paths
		mux.HandleFunc("/report/timetrack", serveReportRequest)
		mux.HandleFunc("/report/billable", serveReportRequest)
		// Add a root handler for basic check
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprintln(w, "Report Handler Debug Server Running")
		})

		server := &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second, // Slightly longer for potential report generation
			WriteTimeout: 60 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("HTTP server ListenAndServe error")
		}
		log.Info("HTTP server stopped gracefully.")

	} else {
		// --- Lambda Mode ---
		log.Info("Starting Report Lambda handler")
		lambdaHandler := lambdaAdapter(serveReportRequest)
		lambda.Start(lambdaHandler)
	}
}
