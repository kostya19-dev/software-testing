package grafana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// tracesPerFile keeps one OTLP request well under the collector body limit.
const tracesPerFile = 100

type jaegerTag struct {
	Key   string          `json:"key"`
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type jaegerSpan struct {
	TraceID       string `json:"traceID"`
	SpanID        string `json:"spanID"`
	OperationName string `json:"operationName"`
	References    []struct {
		RefType string `json:"refType"`
		SpanID  string `json:"spanID"`
	} `json:"references"`
	StartTime int64       `json:"startTime"`
	Duration  int64       `json:"duration"`
	Tags      []jaegerTag `json:"tags"`
	ProcessID string      `json:"processID"`
}

type jaegerTrace struct {
	Spans     []jaegerSpan `json:"spans"`
	Processes map[string]struct {
		ServiceName string      `json:"serviceName"`
		Tags        []jaegerTag `json:"tags"`
	} `json:"processes"`
}

type otlpValue struct {
	StringValue *string  `json:"stringValue,omitempty"`
	IntValue    *string  `json:"intValue,omitempty"`
	DoubleValue *float64 `json:"doubleValue,omitempty"`
	BoolValue   *bool    `json:"boolValue,omitempty"`
}

type otlpAttribute struct {
	Key   string    `json:"key"`
	Value otlpValue `json:"value"`
}

type otlpSpan struct {
	TraceID      string          `json:"traceId"`
	SpanID       string          `json:"spanId"`
	ParentSpanID string          `json:"parentSpanId,omitempty"`
	Name         string          `json:"name"`
	Kind         int             `json:"kind"`
	Start        string          `json:"startTimeUnixNano"`
	End          string          `json:"endTimeUnixNano"`
	Attributes   []otlpAttribute `json:"attributes"`
	Status       struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	} `json:"status"`
}

type otlpScopeSpans struct {
	Scope struct {
		Name string `json:"name"`
	} `json:"scope"`
	Spans []otlpSpan `json:"spans"`
}

type otlpResourceSpans struct {
	Resource struct {
		Attributes []otlpAttribute `json:"attributes"`
	} `json:"resource"`
	ScopeSpans []*otlpScopeSpans `json:"scopeSpans"`
}

type otlpRequest struct {
	ResourceSpans []*otlpResourceSpans `json:"resourceSpans"`
}

var spanKinds = map[string]int{"internal": 1, "server": 2, "client": 3, "producer": 4, "consumer": 5}

// ConvertTraces rewrites a Jaeger UI export as OTLP JSON requests. Spans are moved to the dashboard time axis
// (the run starts at Epoch) and tagged with the scenario and the variant, so Grafana finds them next to the metrics.
func ConvertTraces(source, outDir, scenario, variant string, start float64) (int, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return 0, err
	}
	var export struct {
		Data []jaegerTrace `json:"data"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		return 0, err
	}
	shift := int64(0)
	if !math.IsNaN(start) {
		shift = int64(math.Round((Epoch - start) * 1e6))
	}
	extra := []otlpAttribute{stringAttribute("lab6.scenario", scenario), stringAttribute("lab6.variant", variant)}

	for first := 0; first < len(export.Data); first += tracesPerFile {
		request := otlpRequest{}
		resources := map[string]*otlpResourceSpans{}
		scopes := map[string]*otlpScopeSpans{}
		for _, trace := range export.Data[first:min(first+tracesPerFile, len(export.Data))] {
			for _, span := range trace.Spans {
				process := trace.Processes[span.ProcessID]
				resource := resources[process.ServiceName]
				if resource == nil {
					resource = &otlpResourceSpans{}
					resource.Resource.Attributes = append(attributes(process.Tags), stringAttribute("service.name", process.ServiceName))
					resources[process.ServiceName] = resource
					request.ResourceSpans = append(request.ResourceSpans, resource)
				}
				scopeName := tagString(span.Tags, "otel.scope.name")
				scope := scopes[process.ServiceName+"\x00"+scopeName]
				if scope == nil {
					scope = &otlpScopeSpans{}
					scope.Scope.Name = scopeName
					scopes[process.ServiceName+"\x00"+scopeName] = scope
					resource.ScopeSpans = append(resource.ScopeSpans, scope)
				}
				scope.Spans = append(scope.Spans, convertSpan(span, shift, extra))
			}
		}
		body, err := json.Marshal(request)
		if err != nil {
			return 0, err
		}
		name := fmt.Sprintf("%s-%s-%03d.json", scenario, variant, first/tracesPerFile)
		if err := os.WriteFile(filepath.Join(outDir, name), body, 0o644); err != nil {
			return 0, err
		}
	}
	return len(export.Data), nil
}

func convertSpan(span jaegerSpan, shift int64, extra []otlpAttribute) otlpSpan {
	start := (span.StartTime + shift) * 1000
	out := otlpSpan{
		TraceID:    pad(span.TraceID, 32),
		SpanID:     pad(span.SpanID, 16),
		Name:       span.OperationName,
		Kind:       1,
		Start:      strconv.FormatInt(start, 10),
		End:        strconv.FormatInt(start+span.Duration*1000, 10),
		Attributes: append(attributes(span.Tags), extra...),
	}
	for _, ref := range span.References {
		if ref.RefType == "CHILD_OF" {
			out.ParentSpanID = pad(ref.SpanID, 16)
		}
	}
	if kind, ok := spanKinds[tagString(span.Tags, "span.kind")]; ok {
		out.Kind = kind
	}
	switch status := tagString(span.Tags, "otel.status_code"); {
	case status == "ERROR" || tagString(span.Tags, "error") == "true":
		out.Status.Code = 2
		out.Status.Message = tagString(span.Tags, "otel.status_description")
	case status == "OK":
		out.Status.Code = 1
	}
	return out
}

// Tags that Jaeger derives from OTLP span fields are turned back into those fields.
var derivedTags = map[string]bool{
	"span.kind": true, "otel.status_code": true, "otel.status_description": true,
	"otel.scope.name": true, "otel.scope.version": true, "error": true, "internal.span.format": true,
}

func attributes(tags []jaegerTag) []otlpAttribute {
	out := make([]otlpAttribute, 0, len(tags))
	for _, tag := range tags {
		if derivedTags[tag.Key] {
			continue
		}
		var value otlpValue
		switch tag.Type {
		case "bool":
			var b bool
			if json.Unmarshal(tag.Value, &b) != nil {
				continue
			}
			value.BoolValue = &b
		case "int64":
			var n json.Number
			if json.Unmarshal(tag.Value, &n) != nil {
				continue
			}
			s := n.String()
			value.IntValue = &s
		case "float64":
			var f float64
			if json.Unmarshal(tag.Value, &f) != nil {
				continue
			}
			value.DoubleValue = &f
		default:
			var s string
			if json.Unmarshal(tag.Value, &s) != nil {
				s = string(tag.Value)
			}
			value.StringValue = &s
		}
		out = append(out, otlpAttribute{Key: tag.Key, Value: value})
	}
	return out
}

func stringAttribute(key, value string) otlpAttribute {
	return otlpAttribute{Key: key, Value: otlpValue{StringValue: &value}}
}

func tagString(tags []jaegerTag, key string) string {
	for _, tag := range tags {
		if tag.Key == key {
			var s string
			if json.Unmarshal(tag.Value, &s) == nil {
				return s
			}
			return strings.Trim(string(tag.Value), `"`)
		}
	}
	return ""
}

func pad(id string, width int) string {
	if len(id) >= width {
		return id
	}
	return strings.Repeat("0", width-len(id)) + id
}

// PushTraces sends the converted traces to an OTLP/HTTP endpoint, waiting for the collector to come up.
func PushTraces(dir, endpoint string) (int, error) {
	files, err := filepath.Glob(filepath.Join(dir, TracesDir, "*.json"))
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	url := strings.TrimSuffix(endpoint, "/") + "/v1/traces"
	for i, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			return i, err
		}
		attempts := 1
		if i == 0 {
			attempts = 60
		}
		if err := post(client, url, body, attempts); err != nil {
			return i, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
	}
	return len(files), nil
}

func post(client *http.Client, url string, body []byte, attempts int) error {
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second)
		}
		resp, err := client.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			last = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode/100 == 2 {
			return nil
		}
		last = fmt.Errorf("%s", resp.Status)
	}
	return last
}
