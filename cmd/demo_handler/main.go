package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	aws_config "github.com/aws/a
	"github.com/sirupsen/logrus"
	"github.com/your-org/clickup-reporter/internal/auth" // Needed for auth check
	"github.com/your-org/clickup-reporter/internal/config"
	"github.com/your-org/clickup-reporter/internal/server"
	"github.com/your-org/clickup-reporter/internal/storage"
	// Import other necessary services like demo, wiki, slack when implemented
)

var authenticator *auth.Authenticator
var appConf *config.AppConfig // Store loaded config
// var demoService *demo.Service // Placeholder for demo service

// Initialize config and services once during cold start
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
	logEntry := initLogger.WithField("service", "demo-handler-init")

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
	// demoService = demo.NewService(appConf) // Uncomment when demo service exists
	appConf.Logger.Info("Demo handler initialized successfully")
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

// serveDemoRequest is the core HTTP handler logic for demo reports.
func serveDemoRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	log.Info("Demo handler core logic processing request")

	// --- Authentication Check ---
	session, token, err := authenticator.GetSessionFromRequest(r)
	if err != nil {
		log.WithError(err).Error("Failed to get session from request")
		http.Error(w, "Could not process session", http.StatusInternalServerError)
		return
	}
	if session == nil || token == "" {
		log.Warn("User not authenticated or session invalid/expired")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	userLogger := log.WithFields(logrus.Fields{"user_id": session.UserID})
	userLogger.Info("User authenticated")
	// ---------------------------

	// --- Routing & Logic ---
	// Assuming demo report is generated via GET for simplicity, like Python version
	if r.Method != "GET" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userLogger.Info("Generating demo report (placeholder)")

	// TODO: Implement demo generation logic using demoService
	// 1. Call demo generation service (fetch tasks, format MD)
	//    mdContent, err := demoService.GenerateDemoMarkdown(r.Context(), token)
	//    if err != nil { ... handle error ... }
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

	w.Header().Set("Content-Type", "application/json")
	// Add CORS headers if needed for debug mode
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie")

	w.WriteHeader(http.StatusOK) // Or http.StatusNotImplemented
	err = json.NewEncoder(w).Encode(responsePayload)
	if err != nil {
		userLogger.WithError(err).Error("Failed to encode demo response")
	}
}

func main() {
	if appConf == nil {
		panic("FATAL: AppConfig is nil at main execution")
	}
	log := appConf.Logger

	if appConf.Global.DebugMode {
		// --- HTTP Server Mode ---
		addr := ":" + appConf.Global.DebugPort
		log.Infof("Starting Demo HTTP server in debug mode on %s", addr)

		mux := http.NewServeMux()
		// Register the core handler for the demo path
		mux.HandleFunc("/report/demo", serveDemoRequest) // Assuming path from python app
		// Add a root handler for basic check
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprintln(w, "Demo Handler Debug Server Running")
		})

		server := &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("HTTP server ListenAndServe error")
		}
		log.Info("HTTP server stopped gracefully.")

	} else {
		// --- Lambda Mode ---
		log.Info("Starting Demo Lambda handler")
		lambdaHandler := lambdaAdapter(serveDemoRequest)
		lambda.Start(lambdaHandler)
	}
}
