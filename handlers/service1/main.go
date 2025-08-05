package main

import (
	"context"
	"encoding/json"
	"net/http"

	"observability/logs"
	"observability/middleware"
	"observability/tracer"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var ELASTIC_APM_SERVICE_NAME = "test-service"
var ELASTIC_APM_API_KEY = "RUU2ZmE1Z0JCMnI4REdVdi1zVDk6aS1sZlp0RVJhemJ4YU15djc0cmhGQQ=="
var ELASTIC_APM_ENDPOINT = "https://my-observability-project-b29ff9.apm.us-central1.gcp.elastic.cloud:443"

func main() {
	l, err := logs.NewOtelLoggerBuilder().
		WithEndpointUrl(ELASTIC_APM_ENDPOINT).
		WithServiceName(ELASTIC_APM_SERVICE_NAME).
		WithAuthHeader(ELASTIC_APM_API_KEY).
		Build(context.Background())
	if err != nil {
		panic(err)
	}

	cleanup := tracer.InitTracer()
	defer cleanup()

	mux := http.NewServeMux()
	mux.Handle("/test", TestHandler(l))
	mux.Handle("/test-error", TestErrorHandler(l))
	mux.Handle("/greet", GreetHandler(l))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	wrapped := middleware.TraceMiddleware(ELASTIC_APM_SERVICE_NAME, l)(mux)

	l.Info(nil, "Starting server on :8080")
	http.ListenAndServe(":8080", wrapped)
}

func TestHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		tracer := otel.Tracer("test-service")
		ctx, span := tracer.Start(ctx, "client-request")
		defer span.End()

		req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/test-error", nil)
		if err != nil {
			l.Error(span, "Failed to create outbound request", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "Request creation failed", http.StatusInternalServerError)
			return
		}

		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			l.Error(span, "Outbound request failed", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "Outbound call failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		l.Info(span, "Outbound request completed", map[string]interface{}{
			"status": resp.Status,
		})

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Log triggered!"))
	}
}

func TestErrorHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Propagate context from incoming request
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		tracer := otel.Tracer("test-service")

		ctx, span := tracer.Start(ctx, "Handle /test-error")
		defer span.End()

		l.Info(span, "Started /test-error handler")

		dbCtx, dbSpan := tracer.Start(ctx, "DB Query Error")
		l.Info(dbSpan, "Attempting DB Query...")
		l.Error(dbSpan, "DB connection failed: timeout")
		dbSpan.End()

		bizCtx, bizSpan := tracer.Start(dbCtx, "Business Logic Error")
		l.Info(bizSpan, "Running business logic...")

		l.Errorf(bizSpan, "Validation failed for user input: %v", "missing email")
		bizSpan.End()

		_, apiSpan := tracer.Start(bizCtx, "External API Error")
		l.Info(apiSpan, "Calling third-party API...")
		l.Error(apiSpan, "External API responded with 500 Internal Server Error")
		apiSpan.End()
		l.Error(span, "Error scenario completed", map[string]interface{}{
			"error": "Simulated error for testing"})

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error scenario completed! Logs captured."))
	}
}

type GreetRequest struct {
	Name         string       `json:"Name"`
	Surname      string       `json:"Surname"`
	Contact      Contact      `json:"Contact"`
	Address      Address      `json:"Address"`
	Metadata     Metadata     `json:"Metadata"`
	Preferences  Preferences  `json:"Preferences"`
	CustomFields CustomFields `json:"CustomFields"`
}

type Contact struct {
	Email           string      `json:"Email"`
	Phone           string      `json:"Phone"`
	PreferredMethod string      `json:"PreferredMethod"`
	Social          SocialMedia `json:"Social"`
}

type SocialMedia struct {
	Twitter  string `json:"Twitter"`
	LinkedIn string `json:"LinkedIn"`
}

type Address struct {
	Street      string      `json:"Street"`
	City        string      `json:"City"`
	State       string      `json:"State"`
	PostalCode  string      `json:"PostalCode"`
	Country     string      `json:"Country"`
	Coordinates Coordinates `json:"Coordinates"`
}

type Coordinates struct {
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
}

type Metadata struct {
	Timestamp string   `json:"Timestamp"`
	RequestID string   `json:"RequestID"`
	Tags      []string `json:"Tags"`
	Flags     Flags    `json:"Flags"`
	Source    Source   `json:"Source"`
}

type Flags struct {
	Urgent        bool `json:"Urgent"`
	TestMode      bool `json:"TestMode"`
	IncludeExtras bool `json:"IncludeExtras"`
	RetryEnabled  bool `json:"RetryEnabled"`
}

type Source struct {
	IP      string `json:"IP"`
	Device  string `json:"Device"`
	Browser string `json:"Browser"`
}

type Preferences struct {
	Language      string        `json:"Language"`
	Timezone      string        `json:"Timezone"`
	Theme         string        `json:"Theme"`
	Notifications Notifications `json:"Notifications"`
	Accessibility Accessibility `json:"Accessibility"`
}

type Notifications struct {
	Email bool `json:"Email"`
	SMS   bool `json:"SMS"`
	Push  bool `json:"Push"`
}

type Accessibility struct {
	HighContrast bool   `json:"HighContrast"`
	FontSize     string `json:"FontSize"`
}

type CustomFields struct {
	ReferralCode  string         `json:"ReferralCode"`
	LoyaltyPoints int            `json:"LoyaltyPoints"`
	VIPStatus     string         `json:"VIPStatus"`
	Subscriptions []Subscription `json:"Subscriptions"`
}

type Subscription struct {
	Name   string `json:"Name"`
	Active bool   `json:"Active"`
}

type GreetResponse struct {
	Message string `json:"Message"`
}
type Response struct {
	Name         string       `json:"Name"`
	Surname      string       `json:"Surname"`
	Contact      Contact      `json:"Contact"`
	Address      Address      `json:"Address"`
	Metadata     Metadata     `json:"Metadata"`
	Preferences  Preferences  `json:"Preferences"`
	CustomFields CustomFields `json:"CustomFields"`
}

func GreetHandler(l logs.OtelLogging) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		tracer := otel.Tracer("test-service")
		_, span := tracer.Start(ctx, "Handle /greet")
		defer span.End()

		var req GreetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			l.Error(span, "Invalid JSON", err.Error())
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		l.LogJson(span, "request.body", req)

		// Create multiple sample responses
		var responses = []Response{
			{
				Name:    "Alice",
				Surname: "Johnson",
				Contact: Contact{
					Email:           "alice@example.com",
					Phone:           "+1234567890",
					PreferredMethod: "Email",
					Social: SocialMedia{
						Twitter:  "@alice",
						LinkedIn: "alice-johnson",
					},
				},
				Address: Address{
					Street:     "123 Main St",
					City:       "Metropolis",
					State:      "CA",
					PostalCode: "90210",
					Country:    "USA",
					Coordinates: Coordinates{
						Latitude:  34.0522,
						Longitude: -118.2437,
					},
				},
				Metadata: Metadata{
					Timestamp: "2025-08-04T20:50:00Z",
					RequestID: "req-789",
					Tags:      []string{"premium", "verified"},
					Flags: Flags{
						Urgent:        true,
						TestMode:      false,
						IncludeExtras: true,
						RetryEnabled:  true,
					},
					Source: Source{
						IP:      "192.168.1.1",
						Device:  "iPhone 14",
						Browser: "Safari",
					},
				},
				Preferences: Preferences{
					Language: "en",
					Timezone: "PST",
					Theme:    "dark",
					Notifications: Notifications{
						Email: true,
						SMS:   false,
						Push:  true,
					},
					Accessibility: Accessibility{
						HighContrast: false,
						FontSize:     "medium",
					},
				},
				CustomFields: CustomFields{
					ReferralCode:  "REF123",
					LoyaltyPoints: 1500,
					VIPStatus:     "Gold",
					Subscriptions: []Subscription{
						{Name: "newsletter", Active: true},
					},
				},
			},
			{
				Name:    "Bob",
				Surname: "Smith",
				Contact: Contact{
					Email:           "bob@example.com",
					Phone:           "+9876543210",
					PreferredMethod: "Phone",
					Social: SocialMedia{
						Twitter:  "@bobsmith",
						LinkedIn: "bob-smith",
					},
				},
				Address: Address{
					Street:     "456 Elm St",
					City:       "Gotham",
					State:      "NY",
					PostalCode: "10001",
					Country:    "USA",
					Coordinates: Coordinates{
						Latitude:  40.7128,
						Longitude: -74.0060,
					},
				},
				Metadata: Metadata{
					Timestamp: "2025-08-04T21:00:00Z",
					RequestID: "req-790",
					Tags:      []string{"standard"},
					Flags: Flags{
						Urgent:        false,
						TestMode:      true,
						IncludeExtras: false,
						RetryEnabled:  false,
					},
					Source: Source{
						IP:      "10.0.0.2",
						Device:  "Android",
						Browser: "Chrome",
					},
				},
				Preferences: Preferences{
					Language: "es",
					Timezone: "EST",
					Theme:    "light",
					Notifications: Notifications{
						Email: false,
						SMS:   true,
						Push:  false,
					},
					Accessibility: Accessibility{
						HighContrast: true,
						FontSize:     "large",
					},
				},
				CustomFields: CustomFields{
					ReferralCode:  "REF456",
					LoyaltyPoints: 800,
					VIPStatus:     "Silver",
					Subscriptions: []Subscription{
						{Name: "alerts", Active: false},
					},
				},
			},
		}

		// Log the full slice of responses
		l.LogJson(span, "response.body", responses)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responses)
	}
}
