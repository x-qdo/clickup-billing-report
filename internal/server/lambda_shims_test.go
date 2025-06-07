package server

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// Helper function to read the body and close it
func readBody(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}
	defer r.Body.Close()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func TestNewRequestFromEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("BasicRequest", func(t *testing.T) {
		event := events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/testpath",
			RequestContext: events.APIGatewayProxyRequestContext{
				DomainName: "example.com", // For URL reconstruction
			},
		}
		req, err := NewRequestFromEvent(ctx, event)
		if err != nil {
			t.Fatalf("NewRequestFromEvent failed: %v", err)
		}

		if req.Method != "GET" {
			t.Errorf("expected method GET, got %s", req.Method)
		}
		if req.URL.Path != "/testpath" {
			t.Errorf("expected path /testpath, got %s", req.URL.Path)
		}
	})

	t.Run("QueryParameters", func(t *testing.T) {
		t.Run("SingleValue", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/query",
				QueryStringParameters: map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
				RequestContext: events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}

			query := req.URL.Query()
			if query.Get("param1") != "value1" {
				t.Errorf("expected param1=value1, got %s", query.Get("param1"))
			}
			if query.Get("param2") != "value2" {
				t.Errorf("expected param2=value2, got %s", query.Get("param2"))
			}
			// url.Values.Encode() sorts keys, so this should be stable
			expectedEncoded := "param1=value1&param2=value2"
			if query.Encode() != expectedEncoded {
				t.Errorf("expected encoded query '%s', got '%s'", expectedEncoded, query.Encode())
			}
		})

		t.Run("MultiValue", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/multiquery",
				MultiValueQueryStringParameters: map[string][]string{
					"paramA": {"valA1", "valA2"},
					"paramB": {"valB1"},
				},
				RequestContext: events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}

			query := req.URL.Query()
			if !reflect.DeepEqual(query["paramA"], []string{"valA1", "valA2"}) {
				t.Errorf("expected paramA=[valA1, valA2], got %v", query["paramA"])
			}
			if !reflect.DeepEqual(query["paramB"], []string{"valB1"}) { // Query()["key"] returns a slice
				t.Errorf("expected paramB=[valB1], got %v", query["paramB"])
			}
			if query.Get("paramB") != "valB1" { // Get returns first value
				t.Errorf("expected paramB.Get()=valB1, got %s", query.Get("paramB"))
			}
		})
		t.Run("MultiValueTakesPrecedence", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/query",
				QueryStringParameters: map[string]string{
					"param1": "ignored",
				},
				MultiValueQueryStringParameters: map[string][]string{
					"param1": {"actualValue"},
				},
				RequestContext: events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}
			if req.URL.Query().Get("param1") != "actualValue" {
				t.Errorf("expected param1=actualValue from MultiValue, got %s", req.URL.Query().Get("param1"))
			}
		})
	})

	t.Run("Headers", func(t *testing.T) {
		t.Run("SingleValueHeaders", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/headers",
				Headers: map[string]string{
					"Content-Type":    "application/json",
					"X-Custom-Header": "value1",
				},
				RequestContext: events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}

			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type: application/json, got %s", req.Header.Get("Content-Type"))
			}
			if req.Header.Get("X-Custom-Header") != "value1" {
				t.Errorf("expected X-Custom-Header: value1, got %s", req.Header.Get("X-Custom-Header"))
			}
		})
		t.Run("MultiValueHeaders", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/multiheaders",
				MultiValueHeaders: map[string][]string{
					"Accept-Encoding": {"gzip", "deflate"},
					"X-Api-Key":       {"key1"},
				},
				RequestContext: events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}
			// http.Header canonicalizes keys, so use http.CanonicalHeaderKey for direct map access if needed,
			// or rely on Get/Values which handle canonicalization.
			if !reflect.DeepEqual(req.Header["Accept-Encoding"], []string{"gzip", "deflate"}) {
				t.Errorf("expected Accept-Encoding: [gzip, deflate], got %v", req.Header["Accept-Encoding"])
			}
			if !reflect.DeepEqual(req.Header["X-Api-Key"], []string{"key1"}) {
				t.Errorf("expected X-Api-Key: [key1], got %v", req.Header["X-Api-Key"])
			}
			if req.Header.Get("X-Api-Key") != "key1" { // Get returns first value
				t.Errorf("expected X-Api-Key.Get(): key1, got %s", req.Header.Get("X-Api-Key"))
			}
		})
		t.Run("MultiValueHeadersTakesPrecedence", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/headers",
				Headers: map[string]string{
					"X-Ignored": "should-not-see",
				},
				MultiValueHeaders: map[string][]string{
					"X-Actual": {"actual-value"},
				},
				RequestContext: events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}
			if req.Header.Get("X-Actual") != "actual-value" {
				t.Errorf("expected X-Actual header, got %s", req.Header.Get("X-Actual"))
			}
			if req.Header.Get("X-Ignored") != "" {
				t.Errorf("expected X-Ignored header to be empty, got %s", req.Header.Get("X-Ignored"))
			}
		})
	})

	t.Run("RequestBody", func(t *testing.T) {
		t.Run("PlainText", func(t *testing.T) {
			bodyContent := "Hello, world!"
			event := events.APIGatewayProxyRequest{
				HTTPMethod:      "POST",
				Path:            "/bodytest",
				Body:            bodyContent,
				IsBase64Encoded: false,
				RequestContext:  events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}

			readBodyStr, err := readBody(req)
			if err != nil {
				t.Fatalf("Failed to read request body: %v", err)
			}
			if readBodyStr != bodyContent {
				t.Errorf("expected body '%s', got '%s'", bodyContent, readBodyStr)
			}
		})

		t.Run("Base64Encoded", func(t *testing.T) {
			originalBody := "Hello, base64 world!"
			encodedBody := base64.StdEncoding.EncodeToString([]byte(originalBody))
			event := events.APIGatewayProxyRequest{
				HTTPMethod:      "POST",
				Path:            "/b64body",
				Body:            encodedBody,
				IsBase64Encoded: true,
				RequestContext:  events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}

			readBodyStr, err := readBody(req)
			if err != nil {
				t.Fatalf("Failed to read request body: %v", err)
			}
			if readBodyStr != originalBody {
				t.Errorf("expected decoded body '%s', got '%s'", originalBody, readBodyStr)
			}
		})

		t.Run("Base64EncodedMalformed", func(t *testing.T) {
			malformedBody := "this is not valid base64 %&^"
			event := events.APIGatewayProxyRequest{
				HTTPMethod:      "POST",
				Path:            "/b64malformed",
				Body:            malformedBody,
				IsBase64Encoded: true,
				RequestContext:  events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			_, err := NewRequestFromEvent(ctx, event)
			if err == nil {
				t.Fatalf("Expected an error for malformed base64 body, but got nil")
			}
			if !strings.Contains(err.Error(), "failed to decode base64 body") {
				t.Errorf("Expected error message to contain 'failed to decode base64 body', got: %v", err)
			}
		})

		t.Run("EmptyBody", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod:      "POST",
				Path:            "/emptybody",
				Body:            "",
				IsBase64Encoded: false,
				RequestContext:  events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}
			readBodyStr, err := readBody(req)
			if err != nil {
				t.Fatalf("Failed to read request body: %v", err)
			}
			if readBodyStr != "" {
				t.Errorf("expected empty body, got '%s'", readBodyStr)
			}
		})

		t.Run("EmptyBodyBase64Encoded", func(t *testing.T) {
			event := events.APIGatewayProxyRequest{
				HTTPMethod:      "POST",
				Path:            "/emptybodyb64",
				Body:            "", // base64 encoding of "" is ""
				IsBase64Encoded: true,
				RequestContext:  events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}
			readBodyStr, err := readBody(req)
			if err != nil {
				t.Fatalf("Failed to read request body: %v", err)
			}
			if readBodyStr != "" {
				t.Errorf("expected empty decoded body, got '%s'", readBodyStr)
			}
		})

		t.Run("Base64EncodedFormURLEncoded", func(t *testing.T) {
			originalFormBody := "report_date=2025-06&team_id=123"
			encodedBody := base64.StdEncoding.EncodeToString([]byte(originalFormBody))
			event := events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/b64form",
				Headers: map[string]string{
					"Content-Type": "application/x-www-form-urlencoded",
				},
				Body:            encodedBody,
				IsBase64Encoded: true,
				RequestContext:  events.APIGatewayProxyRequestContext{DomainName: "test.host"},
			}
			req, err := NewRequestFromEvent(ctx, event)
			if err != nil {
				t.Fatalf("NewRequestFromEvent failed: %v", err)
			}

			// Check Content-Type header is preserved
			if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("expected Content-Type header 'application/x-www-form-urlencoded', got '%s'", req.Header.Get("Content-Type"))
			}

			// Parse the form
			err = req.ParseForm()
			if err != nil {
				t.Fatalf("Failed to parse form: %v", err)
			}

			if val := req.FormValue("report_date"); val != "2025-06" {
				t.Errorf("expected form value for 'report_date' to be '2025-06', got '%s'", val)
			}
			if val := req.FormValue("team_id"); val != "123" {
				t.Errorf("expected form value for 'team_id' to be '123', got '%s'", val)
			}

			// Also check r.PostForm
			if val := req.PostFormValue("report_date"); val != "2025-06" {
				t.Errorf("expected PostForm value for 'report_date' to be '2025-06', got '%s'", val)
			}

			// Verify the body can still be read and matches the original decoded form body
			// Note: ParseForm consumes the body, so for this check, we'd ideally create a new request
			// or use a mechanism to re-read it if the http.Request allowed it.
			// However, the primary goal here is to ensure ParseForm works.
			// For a simple string reader, it's consumed.
			// If we wanted to re-read, we'd need to re-initialize the body reader in NewRequestFromEvent
			// or pass a NopCloser that allows multiple reads if the underlying reader supports it.
			// Given the current structure, we'll focus on ParseForm's success.
		})
	})

	t.Run("URLReconstruction", func(t *testing.T) {
		testCases := []struct {
			name           string
			event          events.APIGatewayProxyRequest
			expectedScheme string
			expectedHost   string
			expectedPath   string
		}{
			{
				name: "X-Forwarded-Proto and Host headers",
				event: events.APIGatewayProxyRequest{
					HTTPMethod: "GET", Path: "/p1",
					Headers:        map[string]string{"X-Forwarded-Proto": "https", "Host": "custom.example.com"},
					RequestContext: events.APIGatewayProxyRequestContext{DomainName: "api.gateway.domain"},
				},
				expectedScheme: "https", expectedHost: "custom.example.com", expectedPath: "/p1",
			},
			{
				name: "x-forwarded-proto and host headers (lowercase)",
				event: events.APIGatewayProxyRequest{
					HTTPMethod: "GET", Path: "/p2",
					Headers:        map[string]string{"x-forwarded-proto": "http", "host": "lower.example.com"},
					RequestContext: events.APIGatewayProxyRequestContext{DomainName: "api.gateway.domain"},
				},
				expectedScheme: "http", expectedHost: "lower.example.com", expectedPath: "/p2",
			},
			{
				name: "Only RequestContext.DomainName (implies https and host)",
				event: events.APIGatewayProxyRequest{
					HTTPMethod: "GET", Path: "/p3",
					Headers:        map[string]string{},
					RequestContext: events.APIGatewayProxyRequestContext{DomainName: "rc.example.com"},
				},
				expectedScheme: "https", expectedHost: "rc.example.com", expectedPath: "/p3",
			},
			{
				name: "No headers, no DomainName (fallback)",
				event: events.APIGatewayProxyRequest{
					HTTPMethod: "GET", Path: "/p4",
					Headers:        map[string]string{},
					RequestContext: events.APIGatewayProxyRequestContext{},
				},
				expectedScheme: "http", expectedHost: "lambda.internal", expectedPath: "/p4",
			},
			{
				name: "X-Forwarded-Proto only, fallback host from RequestContext",
				event: events.APIGatewayProxyRequest{
					HTTPMethod: "GET", Path: "/p5",
					Headers:        map[string]string{"X-Forwarded-Proto": "https"},
					RequestContext: events.APIGatewayProxyRequestContext{DomainName: "fallback.host.com"},
				},
				expectedScheme: "https", expectedHost: "fallback.host.com", expectedPath: "/p5",
			},
			{
				name: "Host header only, scheme from RequestContext.DomainName presence",
				event: events.APIGatewayProxyRequest{
					HTTPMethod: "GET", Path: "/p6",
					Headers:        map[string]string{"Host": "header.host.com"},
					RequestContext: events.APIGatewayProxyRequestContext{DomainName: "fallback.domain.com"},
				},
				expectedScheme: "https", expectedHost: "header.host.com", expectedPath: "/p6",
			},
			{
				name: "Host header only, no DomainName (fallback scheme http)",
				event: events.APIGatewayProxyRequest{
					HTTPMethod: "GET", Path: "/p7",
					Headers:        map[string]string{"Host": "header.host.com"},
					RequestContext: events.APIGatewayProxyRequestContext{},
				},
				expectedScheme: "http", expectedHost: "header.host.com", expectedPath: "/p7",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := NewRequestFromEvent(ctx, tc.event)
				if err != nil {
					t.Fatalf("NewRequestFromEvent failed: %v", err)
				}
				if req.URL.Scheme != tc.expectedScheme {
					t.Errorf("expected scheme %s, got %s", tc.expectedScheme, req.URL.Scheme)
				}
				if req.URL.Host != tc.expectedHost {
					t.Errorf("expected host %s, got %s", tc.expectedHost, req.URL.Host)
				}
				if req.URL.Path != tc.expectedPath {
					t.Errorf("expected path %s, got %s", tc.expectedPath, req.URL.Path)
				}
				// Check full URL string (without query parameters, which are tested separately)
				expectedFullURL := tc.expectedScheme + "://" + tc.expectedHost + tc.expectedPath
				if req.URL.String() != expectedFullURL {
					t.Errorf("expected full URL (sans query) %s, got %s", expectedFullURL, req.URL.String())
				}
			})
		}
	})

	t.Run("RemoteAddr", func(t *testing.T) {
		sourceIP := "192.0.2.100"
		event := events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/remote",
			RequestContext: events.APIGatewayProxyRequestContext{
				Identity:   events.APIGatewayRequestIdentity{SourceIP: sourceIP},
				DomainName: "test.host",
			},
		}
		req, err := NewRequestFromEvent(ctx, event)
		if err != nil {
			t.Fatalf("NewRequestFromEvent failed: %v", err)
		}
		if req.RemoteAddr != sourceIP {
			t.Errorf("expected RemoteAddr %s, got %s", sourceIP, req.RemoteAddr)
		}
	})

	t.Run("ContextPropagation", func(t *testing.T) {
		type ctxKey string
		const testKey ctxKey = "testKey"
		testValue := "testValue"

		parentCtx := context.WithValue(context.Background(), testKey, testValue)

		event := events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/ctx",
			RequestContext: events.APIGatewayProxyRequestContext{
				DomainName: "test.host",
			},
		}
		req, err := NewRequestFromEvent(parentCtx, event)
		if err != nil {
			t.Fatalf("NewRequestFromEvent failed: %v", err)
		}

		if val := req.Context().Value(testKey); val != testValue {
			t.Errorf("expected context value '%s' for key '%s', got '%v'", testValue, testKey, val)
		}
	})

	t.Run("URLParseErrorScenario", func(t *testing.T) {
		// This test attempts to trigger the url.Parse error within NewRequestFromEvent.
		// It's tricky because the URL construction logic is quite robust.
		// One way to make url.Parse fail is to provide a scheme with invalid characters.
		// The current logic prioritizes X-Forwarded-Proto.
		event := events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/test",
			Headers: map[string]string{
				"X-Forwarded-Proto": "http@!:", // Invalid scheme character
				"Host":              "example.com",
			},
		}
		_, err := NewRequestFromEvent(ctx, event)
		if err == nil {
			t.Fatal("Expected NewRequestFromEvent to fail due to URL parsing error, but it succeeded")
		}
		if !strings.Contains(err.Error(), "failed to parse reconstructed URL") {
			t.Errorf("Expected error message to contain 'failed to parse reconstructed URL', got: %v", err)
		}
	})
}
