package otlp

import (
	"context"
	"encoding/hex"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	collogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	common "go.opentelemetry.io/proto/otlp/common/v1"

	"github.com/mr-andrej/junkyard/internal/model"
	"github.com/mr-andrej/junkyard/internal/pipeline"
)

// Normalize converts an OTLP ExportLogsServiceRequest into internal log
// entries. Storage never sees OTLP types beyond this point.
func Normalize(req *collogs.ExportLogsServiceRequest) []model.LogEntry {
	var out []model.LogEntry

	for _, rl := range req.GetResourceLogs() {
		service := ""
		for _, kv := range rl.GetResource().GetAttributes() {
			if kv.GetKey() == "service.name" {
				service = kv.GetValue().GetStringValue()
			}
		}

		for _, sl := range rl.GetScopeLogs() {
			scopeAttrs := attrsToMap(sl.GetScope().GetAttributes())

			for _, lr := range sl.GetLogRecords() {
				var attrs map[string]string
				if len(scopeAttrs)+len(lr.GetAttributes()) > 0 {
					attrs = make(map[string]string, len(scopeAttrs)+len(lr.GetAttributes()))
					for k, v := range scopeAttrs {
						attrs[k] = v
					}
					for _, kv := range lr.GetAttributes() {
						attrs[kv.GetKey()] = valueToString(kv.GetValue())
					}
				}

				out = append(out, model.LogEntry{
					Timestamp:      time.Unix(0, int64(lr.GetTimeUnixNano())).UTC(),
					SeverityNumber: int32(lr.GetSeverityNumber()),
					SeverityText:   lr.GetSeverityText(),
					Body:           valueToString(lr.GetBody()),
					Service:        service,
					Attributes:     attrs,
					TraceID:        hex.EncodeToString(lr.GetTraceId()),
					SpanID:         hex.EncodeToString(lr.GetSpanId()),
				})
			}
		}
	}

	return out
}

func attrsToMap(kvs []*common.KeyValue) map[string]string {
	if len(kvs) == 0 {
		return nil
	}
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		m[kv.GetKey()] = valueToString(kv.GetValue())
	}
	return m
}

func valueToString(v *common.AnyValue) string {
	switch val := v.GetValue().(type) {
	case *common.AnyValue_StringValue:
		return val.StringValue
	case *common.AnyValue_BoolValue:
		return strconv.FormatBool(val.BoolValue)
	case *common.AnyValue_IntValue:
		return strconv.FormatInt(val.IntValue, 10)
	case *common.AnyValue_DoubleValue:
		return strconv.FormatFloat(val.DoubleValue, 'g', -1, 64)
	case *common.AnyValue_BytesValue:
		return hex.EncodeToString(val.BytesValue)
	case *common.AnyValue_ArrayValue:
		parts := make([]string, 0, len(val.ArrayValue.GetValues()))
		for _, item := range val.ArrayValue.GetValues() {
			parts = append(parts, valueToString(item))
		}
		return "[" + strings.Join(parts, ",") + "]"
	case *common.AnyValue_KvlistValue:
		parts := make([]string, 0, len(val.KvlistValue.GetValues()))
		for _, kv := range val.KvlistValue.GetValues() {
			parts = append(parts, kv.GetKey()+"="+valueToString(kv.GetValue()))
		}
		return "{" + strings.Join(parts, ",") + "}"
	default:
		return ""
	}
}

// logsService implements the OTLP/gRPC LogsService.
type logsService struct {
	collogs.UnimplementedLogsServiceServer
	pipe *pipeline.Pipeline
}

func (s *logsService) Export(ctx context.Context, req *collogs.ExportLogsServiceRequest) (*collogs.ExportLogsServiceResponse, error) {
	if err := s.pipe.EnqueueBatch(Normalize(req)); err != nil {
		return nil, status.Error(codes.ResourceExhausted, "ingest queue full, retry later")
	}
	return &collogs.ExportLogsServiceResponse{
		PartialSuccess: &collogs.ExportLogsPartialSuccess{RejectedLogRecords: 0},
	}, nil
}

// NewGRPCServer returns a gRPC server serving the OTLP logs service.
func NewGRPCServer(pipe *pipeline.Pipeline) *grpc.Server {
	srv := grpc.NewServer()
	collogs.RegisterLogsServiceServer(srv, &logsService{pipe: pipe})
	return srv
}

// NewHTTPHandler returns the OTLP/HTTP logs receiver (POST /v1/logs).
func NewHTTPHandler(pipe *pipeline.Pipeline) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/v1/logs", &httpLogsHandler{pipe: pipe})
	return mux
}

type httpLogsHandler struct {
	pipe *pipeline.Pipeline
}

func (h *httpLogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		mediaType = r.Header.Get("Content-Type")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req collogs.ExportLogsServiceRequest
	switch mediaType {
	case "application/x-protobuf":
		err = proto.Unmarshal(body, &req)
	case "application/json":
		err = protojson.Unmarshal(body, &req)
	default:
		http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
		return
	}
	if err != nil {
		http.Error(w, "failed to decode request", http.StatusBadRequest)
		return
	}

	// Whole-batch accept/reject: a rejected batch never lands in
	// partial_success, the client is told to retry everything instead.
	if err := h.pipe.EnqueueBatch(Normalize(&req)); err != nil {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "ingest queue full, retry later", http.StatusServiceUnavailable)
		return
	}

	resp := &collogs.ExportLogsServiceResponse{
		PartialSuccess: &collogs.ExportLogsPartialSuccess{RejectedLogRecords: 0},
	}

	switch mediaType {
	case "application/x-protobuf":
		out, _ := proto.Marshal(resp)
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Write(out)
	default:
		out, _ := protojson.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	}
}
