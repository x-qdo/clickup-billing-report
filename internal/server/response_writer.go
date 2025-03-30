package server

import (
	"bytes"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

// HttpResponseWriter acts as an http.ResponseWriter that captures the response
// details (status code, headers, body) to be converted into an
// events.APIGatewayProxyResponse.
type HttpResponseWriter struct {
	header      http.Header
	buffer      bytes.Buffer
	statusCode  int
	wroteHeader bool
}

// NewResponseWriter creates a new HttpResponseWriter.
func NewResponseWriter() *HttpResponseWriter {
	return &HttpResponseWriter{
		header:     make(http.Header),
		statusCode: 0, // Explicitly 0, will default to 200 if Write is called first
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (rw *HttpResponseWriter) Header() http.Header {
	return rw.header
}

// Write writes the data to the internal buffer.
// If WriteHeader has not yet been called, Write calls WriteHeader(http.StatusOK)
// before writing the data.
func (rw *HttpResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK) // Default to 200 OK if Write is called before WriteHeader
	}
	// Write to the internal buffer
	return rw.buffer.Write(b)
}

// WriteHeader sends an HTTP response header with the provided status code.
// If WriteHeader is called multiple times, the subsequent calls are ignored.
func (rw *HttpResponseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		logrus.Warn("http.ResponseWriter.WriteHeader called multiple times for the same request.")
		return
	}
	rw.statusCode = statusCode
	rw.wroteHeader = true
	// Headers set via rw.Header() are stored in rw.header and applied in ToAPIGatewayProxyResponse
}

// ToAPIGatewayProxyResponse converts the captured response data into the format
// expected by AWS API Gateway Lambda proxy integration.
func (rw *HttpResponseWriter) ToAPIGatewayProxyResponse() events.APIGatewayProxyResponse {
	// Convert http.Header (map[string][]string) to APIGateway's MultiValueHeaders
	multiValueHeaders := make(map[string][]string)
	for key, values := range rw.header {
		multiValueHeaders[key] = values
	}

	// Determine final status code
	finalStatusCode := rw.statusCode
	if !rw.wroteHeader {
		// If WriteHeader was never called, default to 200 OK
		finalStatusCode = http.StatusOK
		logrus.WithField("finalStatusCode", finalStatusCode).Debug("WriteHeader never called, defaulting status code")
	}

	// Construct the response object
	response := events.APIGatewayProxyResponse{
		StatusCode:        finalStatusCode,
		Headers:           convertMultiToSingleValueHeaders(multiValueHeaders), // For single-value compatibility
		MultiValueHeaders: multiValueHeaders,
		Body:              rw.buffer.String(),
		IsBase64Encoded:   false, // Assuming text/json body, adjust if handling binary
	}

	return response
}

// convertMultiToSingleValueHeaders takes the MultiValueHeaders and creates
// a single-value map, typically taking the first value if multiple exist.
func convertMultiToSingleValueHeaders(multi map[string][]string) map[string]string {
	single := make(map[string]string)
	for key, values := range multi {
		if len(values) > 0 {
			single[key] = values[0] // Take the first value
		}
	}
	return single
}
