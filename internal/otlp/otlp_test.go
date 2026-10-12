package otlp

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	collogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"
	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"

	"github.com/mr-andrej/junkyard/internal/model"
	"github.com/mr-andrej/junkyard/internal/pipeline"
)

// collector is a pipeline.Writer that records every entry it receives.
type collector struct {
	mu      sync.Mutex
	entries []model.LogEntry
}

func (c *collector) write(batch []model.LogEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, batch...)
	return nil
}

func (c *collector) all() []model.LogEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]model.LogEntry, len(c.entries))
	copy(out, c.entries)
	return out
}

func strVal(s string) *common.AnyValue {
	return &common.AnyValue{Value: &common.AnyValue_StringValue{StringValue: s}}
}

// sampleRequest builds a small ExportLogsServiceRequest with one record.
func sampleRequest() *collogs.ExportLogsServiceRequest {
	return &collogs.ExportLogsServiceRequest{
		ResourceLogs: []*logs.ResourceLogs{{
			Resource: &resource.Resource{
				Attributes: []*common.KeyValue{
					{Key: "service.name", Value: strVal("smoke")},
					{Key: "service.version", Value: strVal("1.2.3")},
				},
			},
			ScopeLogs: []*logs.ScopeLogs{{
				Scope: &common.InstrumentationScope{
					Name:       "test-scope",
					Attributes: []*common.KeyValue{{Key: "scope.attr", Value: strVal("scoped")}},
				},
				LogRecords: []*logs.LogRecord{{
					TimeUnixNano:   1700000000000000000,
					SeverityNumber: logs.SeverityNumber_SEVERITY_NUMBER_INFO,
					SeverityText:   "INFO",
					Body:           strVal("hello junkyard"),
					Attributes: []*common.KeyValue{
						{Key: "env", Value: strVal("test")},
						{Key: "retries", Value: &common.AnyValue{Value: &common.AnyValue_IntValue{IntValue: 3}}},
					},
					TraceId: []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b},
					SpanId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
				}},
			}},
		}},
	}
}

func checkNormalized(t *testing.T, entries []model.LogEntry) {
	t.Helper()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]

	if e.Service != "smoke" {
		t.Errorf("service: expected %q, got %q", "smoke", e.Service)
	}
	if e.Body != "hello junkyard" {
		t.Errorf("body: expected %q, got %q", "hello junkyard", e.Body)
	}
	if e.SeverityText != "INFO" || e.SeverityNumber != int32(logs.SeverityNumber_SEVERITY_NUMBER_INFO) {
		t.Errorf("severity: got %q/%d", e.SeverityText, e.SeverityNumber)
	}
	if want := time.Unix(0, 1700000000000000000).UTC(); !e.Timestamp.Equal(want) {
		t.Errorf("timestamp: expected %v, got %v", want, e.Timestamp)
	}
	if e.TraceID != "deadbeef000102030405060708090a0b" {
		t.Errorf("trace ID: got %q", e.TraceID)
	}
	if e.SpanID != "0102030405060708" {
		t.Errorf("span ID: got %q", e.SpanID)
	}
	if e.Attributes["env"] != "test" {
		t.Errorf("attribute env: got %q", e.Attributes["env"])
	}
	if e.Attributes["retries"] != "3" {
		t.Errorf("attribute retries: got %q", e.Attributes["retries"])
	}
	if e.Attributes["scope.attr"] != "scoped" {
		t.Errorf("attribute scope.attr: got %q", e.Attributes["scope.attr"])
	}
}

// waitForEntries polls until the collector has at least n entries.
func waitForEntries(t *testing.T, c *collector, n int) []model.LogEntry {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if entries := c.all(); len(entries) >= n {
			return entries
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d entries, got %d", n, len(c.all()))
	return nil
}

func TestHTTPProtobufRoundTrip(t *testing.T) {
	c := &collector{}
	p := pipeline.New(16, 1, time.Hour, c.write)
	defer p.Shutdown()

	srv := httptest.NewServer(NewHTTPHandler(p))
	defer srv.Close()

	body, err := proto.Marshal(sampleRequest())
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	resp, err := http.Post(srv.URL+"/v1/logs", "application/x-protobuf", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		t.Fatalf("expected protobuf response, got %q", ct)
	}

	var exportResp collogs.ExportLogsServiceResponse
	respBody, _ := readAll(resp)
	if err := proto.Unmarshal(respBody, &exportResp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := exportResp.GetPartialSuccess().GetRejectedLogRecords(); got != 0 {
		t.Fatalf("expected 0 rejected records, got %d", got)
	}

	checkNormalized(t, waitForEntries(t, c, 1))
}

func TestHTTPJSONRoundTrip(t *testing.T) {
	c := &collector{}
	p := pipeline.New(16, 1, time.Hour, c.write)
	defer p.Shutdown()

	srv := httptest.NewServer(NewHTTPHandler(p))
	defer srv.Close()

	// Note: protojson treats bytes fields per the proto3 JSON mapping, so
	// traceId/spanId are base64 here rather than the hex form the OTLP/JSON
	// spec describes.
	jsonBody := `{
	  "resourceLogs": [{
	    "resource": {"attributes": [{"key": "service.name", "value": {"stringValue": "smoke"}}]},
	    "scopeLogs": [{
	      "scope": {"name": "test-scope", "attributes": [{"key": "scope.attr", "value": {"stringValue": "scoped"}}]},
	      "logRecords": [{
	        "timeUnixNano": "1700000000000000000",
	        "severityNumber": 9,
	        "severityText": "INFO",
	        "body": {"stringValue": "hello junkyard"},
	        "attributes": [
	          {"key": "env", "value": {"stringValue": "test"}},
	          {"key": "retries", "value": {"intValue": "3"}}
	        ],
	        "traceId": "3q2+7wABAgMEBQYHCAkKCw==",
	        "spanId": "AQIDBAUGBwg="
	      }]
	    }]
	  }]
	}`

	resp, err := http.Post(srv.URL+"/v1/logs", "application/json", bytes.NewBufferString(jsonBody))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected JSON response, got %q", ct)
	}

	checkNormalized(t, waitForEntries(t, c, 1))
}

// fullPipeline returns a pipeline whose queue is already full, plus a
// release function that lets the writer finish.
func fullPipeline(t *testing.T) (*pipeline.Pipeline, func()) {
	t.Helper()
	blocked := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once

	p := pipeline.New(2, 1, time.Hour, func(batch []model.LogEntry) error {
		once.Do(func() { close(started) })
		<-blocked
		return nil
	})

	if err := p.Enqueue(model.LogEntry{Body: "writer"}); err != nil {
		t.Fatalf("enqueue writer entry: %v", err)
	}
	<-started

	for i := 0; i < 2; i++ {
		if err := p.Enqueue(model.LogEntry{Body: "queued"}); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	return p, func() {
		close(blocked)
		p.Shutdown()
	}
}

func TestHTTPQueueFullReturns503(t *testing.T) {
	p, release := fullPipeline(t)
	defer release()

	srv := httptest.NewServer(NewHTTPHandler(p))
	defer srv.Close()

	body, err := proto.Marshal(sampleRequest())
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	resp, err := http.Post(srv.URL+"/v1/logs", "application/x-protobuf", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.StatusCode)
	}
	if ra := resp.Header.Get("Retry-After"); ra == "" {
		t.Fatal("expected Retry-After header on 503")
	}
}

func TestGRPCExport(t *testing.T) {
	tests := []struct {
		name      string
		full      bool
		wantCode  codes.Code
		wantEntry bool
	}{
		{"happy path", false, codes.OK, true},
		{"queue full", true, codes.ResourceExhausted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &collector{}
			var p *pipeline.Pipeline
			if tt.full {
				var release func()
				p, release = fullPipeline(t)
				defer release()
			} else {
				p = pipeline.New(16, 1, time.Hour, c.write)
				defer p.Shutdown()
			}

			lis := bufconn.Listen(1024 * 1024)
			srv := NewGRPCServer(p)
			go srv.Serve(lis)
			defer srv.Stop()

			ctx := context.Background()
			conn, err := grpc.NewClient("passthrough:///bufnet",
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
					return lis.DialContext(ctx)
				}),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close()

			client := collogs.NewLogsServiceClient(conn)
			resp, err := client.Export(ctx, sampleRequest())

			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("expected code %v, got %v (err: %v)", tt.wantCode, got, err)
			}

			if tt.wantEntry {
				if resp.GetPartialSuccess().GetRejectedLogRecords() != 0 {
					t.Fatalf("expected 0 rejected records, got %d", resp.GetPartialSuccess().GetRejectedLogRecords())
				}
				checkNormalized(t, waitForEntries(t, c, 1))
			}
		})
	}
}

func readAll(resp *http.Response) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}
