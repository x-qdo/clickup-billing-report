package server

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

// Common headers for JSON responses
var jsonHeaders = map[string]string{
	"Content-Type":                     "application/json",
	"Access-Control-Allow-Origin":      "*",
	"Access-Control-Allow-Methods":     "GET, POST, PUT, DELETE, OPTIONS",
	"Access-Control-Allow-Headers":     "Content-Type, Authorization, X-Amz-Date, X-Api-Key, X-Amz-Security-Token, Cookie",
	"Access-Control-Allow-Credentials": "true",
}

// ErrorResponse defines the structure for JSON error responses.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// JSONResponse sends a JSON response with the given status code and payload.
// Returns APIGatewayProxyResponse directly.
func JSONResponse(statusCode int, payload interface{}) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal JSON response payload")
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    jsonHeaders,
			Body:       `{"error": "Internal Server Error", "message": "Failed to generate response"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    jsonHeaders,
		Body:       string(body),
	}, nil
}

// ErrorJSONResponse sends a JSON error response.
// Returns APIGatewayProxyResponse directly.
func ErrorJSONResponse(statusCode int, errorType string, message string) (events.APIGatewayProxyResponse, error) {
	payload := ErrorResponse{
		Error:   errorType,
		Message: message,
	}
	// Log the error being sent back
	logrus.WithFields(logrus.Fields{
		"status_code": statusCode,
		"error_type":  errorType,
		"message":     message,
	}).Warn("Sending error response")

	// Use JSONResponse to construct the final response
	return JSONResponse(statusCode, payload)
}

// OptionsResponse handles CORS preflight requests.
// Returns APIGatewayProxyResponse directly.
func OptionsResponse() (events.APIGatewayProxyResponse, error) {
	// Log CORS preflight request
	logrus.Debug("Handling CORS preflight OPTIONS request")
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    jsonHeaders,
		Body:       "",
	}, nil
}
