package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	aws_config "github.com/aws/a
	"github.com/sirupsen/logrus"
	"github.com/your-org/clickup-reporter/internal/auth"
	"github.com/your-org/clickup-reporter/internal/config"
	"github.com/your-org/clickup-reporter/internal/server"
	"github.com/your-org/clickup-reporter/internal/storage" // Import storage package
)

var authenticator *auth.Authenticator
var appConf *config.AppConfig // Store loaded config

// Initialize config and authenticator once during cold start
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
	initLogger.SetLevel(logrus.InfoLevel) // Or desired level
	initLogger.SetFormatter(&logrus.JSONFormatter{})
	logEntry := initLogger.WithField("service", "auth-handler-init")

	// Create the DynamoDB store instance
	store, err := storage.NewDynamoDBStore(awsCfg, initLogger)
	if err != nil {
		panic(fmt.Sprintf("FATAL: Failed to initialize DynamoDB store: %v", err))
	}

	// Load application config, injecting the store
	appConf, err = config.LoadConfig(ctx, store)
	if err != nil {
		// Log non-fatal config errors, but panic on fatal ones
		logEntry.WithError(err).Error("Non-fatal error loading config during init")
		// Check if appConf is nil, which indicates a fatal error within LoadConfig
		if appConf == nil {
			panic(fmt.Sprintf("FATAL: Failed to load config: %v", err))
		}
	}
	if appConf.Store == nil {
		panic("FATAL: DataStore is nil after config load")
	}

	// Initialize authenticator with the loaded config
	authenticator = auth.NewAuthenticator(appConf)
	appConf.Logger.Info("Auth handler initialized successfully")
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

// serveAuthRequest is the core HTTP handler logic.
func serveAuthRequest(w http.ResponseWriter, r *http.Request) {
	log := appConf.Logger.WithFields(logrus.Fields{
		"method": r.Method,
		"path":   r.URL.Path,
	})
	log.Info("Auth handler core logic processing request")

	// Simple routing based on path suffix
	switch {
	case strings.HasSuffix(r.URL.Path, "/auth/clickup"): // Start OAuth flow
		if r.Method != "GET" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		authURL := authenticator.StartAuth(w, r) // StartAuth sets cookies on 'w'
		log.WithField("redirect_url", authURL).Info("Redirecting user to ClickUp for authentication")

		// Set redirect header and status code directly on the ResponseWriter
		w.Header().Set("Location", authURL)
		w.WriteHeader(http.StatusFound) // 302 Redirect

	case strings.HasSuffix(r.URL.Path, "/auth/callback"): // Handle OAuth callback
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
		// TODO: Make the redirect target configurable
		http.Redirect(w, r, "/report", http.StatusFound) // Use standard http.Redirect

	case strings.HasSuffix(r.URL.Path, "/auth/logout"): // Handle logout
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
		http.Redirect(w, r, "/", http.StatusFound) // Redirect to root

	default:
		log.Warn("Auth handler received request for unknown path")
		http.NotFound(w, r)
	}
}

func main() {
	// Ensure config is loaded (init should handle this, but double-check)
	if appConf == nil {
		panic("FATAL: AppConfig is nil at main execution")
	}
	log := appConf.Logger

	// Check Debug Mode
	if appConf.Global.DebugMode {
		// --- HTTP Server Mode ---
		addr := ":" + appConf.Global.DebugPort
		log.Infof("Starting Auth HTTP server in debug mode on %s", addr)

		// Create a simple mux or use http.HandleFunc
		mux := http.NewServeMux()
		mux.HandleFunc("/auth/clickup", serveAuthRequest)
		mux.HandleFunc("/auth/callback", serveAuthRequest)
		mux.HandleFunc("/auth/logout", serveAuthRequest)
		// Add a root handler for basic check
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" { // Handle 404 for other paths in debug mode
				http.NotFound(w, r)
				return
			}
			fmt.Fprintln(w, "Auth Handler Debug Server Running")
		})

		server := &http.Server{
			Addr:         addr,
			Handler:      mux, // Use the mux
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
		log.Info("Starting Auth Lambda handler")
		// Wrap the core handler with the Lambda adapter
		lambdaHandler := lambdaAdapter(serveAuthRequest)
		lambda.Start(lambdaHandler)
	}
}
