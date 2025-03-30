package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// responseWriterShim is REMOVED. Use HttpResponseWriter instead.

// NewRequestFromEvent creates a minimal *http.Request from an APIGatewayProxyRequest.
// This is useful for functions expecting *http.Request (like cookie handling).
func NewRequestFromEvent(ctx context.Context, event events.APIGatewayProxyRequest) (*http.Request, error) {
	// Reconstruct URL (approximation)
	// Prefer X-Forwarded-Proto and Host from headers if available (common with ALB/API GW)
	scheme := event.Headers["x-forwarded-proto"]
	if scheme == "" {
		scheme = event.Headers["X-Forwarded-Proto"] // Check case variation
	}
	if scheme == "" {
		// Fallback if headers aren't present (e.g., direct Lambda invocation?)
		if event.RequestContext.DomainName != "" {
			scheme = "https" // Assume https if domain is known
		} else {
			scheme = "http" // Default fallback
		}
	}

	host := event.Headers["host"]
	if host == "" {
		host = event.Headers["Host"] // Check case variation
	}
	if host == "" {
		host = event.RequestContext.DomainName // Fallback to domain name from context
	}
	// If still no host, we might be in trouble, but proceed anyway
	if host == "" {
		host = "lambda.internal" // Placeholder host
	}

	path := event.Path
	rawQuery := ""
	// Use MultiValueQueryStringParameters for better query reconstruction
	if len(event.MultiValueQueryStringParameters) > 0 {
		query := url.Values{}
		for k, v := range event.MultiValueQueryStringParameters {
			for _, iv := range v {
				query.Add(k, iv) // Use Add to handle multiple values for the same key
			}
		}
		rawQuery = query.Encode()
	} else if len(event.QueryStringParameters) > 0 { // Fallback to single value params
		query := url.Values{}
		for k, v := range event.QueryStringParameters {
			query.Set(k, v)
		}
		rawQuery = query.Encode()
	}

	fullURL := fmt.Sprintf("%s://%s%s", scheme, host, path)
	if rawQuery != "" {
		fullURL += "?" + rawQuery
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reconstructed URL '%s': %w", fullURL, err)
	}

	// Create request body reader
	var bodyReader strings.Reader
	if event.IsBase64Encoded {
		// Note: Base64 decoding should ideally happen before calling this function
		// if the body is needed, as this function doesn't handle it.
		// For now, pass the encoded string if necessary.
		// Consider adding decoding here if required by consumers of the *http.Request.
		bodyReader = *strings.NewReader(event.Body) // Pass potentially encoded body
	} else {
		bodyReader = *strings.NewReader(event.Body)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, event.HTTPMethod, u.String(), &bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request from event: %w", err)
	}

	// Add headers (including cookies) using MultiValueHeaders
	for k, v := range event.MultiValueHeaders {
		// http.Header is map[string][]string, direct assignment works
		req.Header[http.CanonicalHeaderKey(k)] = v
	}
	// If MultiValueHeaders is empty, try Headers (single value)
	if len(event.MultiValueHeaders) == 0 {
		for k, v := range event.Headers {
			req.Header.Set(k, v) // Use Set for single value map
		}
	}

	// Set remote address (approximation)
	req.RemoteAddr = event.RequestContext.Identity.SourceIP

	return req, nil
}

// MergeHeaders is REMOVED. Headers are handled by HttpResponseWriter.
