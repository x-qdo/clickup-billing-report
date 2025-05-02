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
		statusCode: 0,
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
		rw.WriteHeader(http.StatusOK)
	}
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
}

// ToAPIGatewayProxyResponse converts the captured response data into the format
// expected by AWS API Gateway Lambda proxy integration.
func (rw *HttpResponseWriter) ToAPIGatewayProxyResponse() events.APIGatewayProxyResponse {
	multiValueHeaders := make(map[string][]string)
	for key, values := range rw.header {
		multiValueHeaders[key] = values
	}

	finalStatusCode := rw.statusCode
	if !rw.wroteHeader {
		finalStatusCode = http.StatusOK
		logrus.WithField("finalStatusCode", finalStatusCode).Debug("WriteHeader never called, defaulting status code")
	}

	// Construct the response object
	response := events.APIGatewayProxyResponse{
		StatusCode:        finalStatusCode,
		Headers:           convertMultiToSingleValueHeaders(multiValueHeaders),
		MultiValueHeaders: multiValueHeaders,
		Body:              rw.buffer.String(),
		IsBase64Encoded:   false,
	}

	return response
}

// convertMultiToSingleValueHeaders takes the MultiValueHeaders and creates
// a single-value map, typically taking the first value if multiple exist.
func convertMultiToSingleValueHeaders(multi map[string][]string) map[string]string {
	single := make(map[string]string)
	for key, values := range multi {
		if len(values) > 0 {
			single[key] = values[0]
		}
	}
	return single
}
