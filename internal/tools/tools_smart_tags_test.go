package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/api"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
)

func TestListTagsIncludesSmartTagDiscoveryGuidance(t *testing.T) {
	server := newKlickTippTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tag" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]string{
			"21": "Customers",
			"22": "Webinar",
		})
	})
	defer server.Close()

	tool := findTool(t, build(newTestClient(server.URL), config.Config{}), "list_tags")
	result, err := tool.run(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("list_tags returned error: %v", err)
	}

	payload := result.(map[string]any)
	discovery := payload["discovery"].(map[string]any)
	if discovery["manual_tag_catalog_supported"] != true {
		t.Fatalf("expected manual tag catalog support")
	}
	if discovery["smart_tag_catalog_supported"] != false {
		t.Fatalf("expected smart tag catalog to be unsupported")
	}
	if note := discovery["note"].(string); !strings.Contains(note, "Smart Tags do not currently have an independently discoverable catalog") {
		t.Fatalf("unexpected discovery note: %s", note)
	}
}

func TestGetTagSupportsManualAndSmartTagByKnownID(t *testing.T) {
	server := newKlickTippTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tag":
			writeJSON(t, w, map[string]string{
				"21": "Customers",
				"22": "Webinar",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/tag/21":
			writeJSON(t, w, map[string]any{
				"tagid": "21",
				"name":  "Customers",
				"type":  "manual",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/tag/9001":
			writeJSON(t, w, map[string]any{
				"tagid":      "9001",
				"name":       "Bali",
				"isSmartTag": true,
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	tool := findTool(t, build(newTestClient(server.URL), config.Config{}), "get_tag")

	manual, err := tool.run(context.Background(), map[string]any{"tagid": 21})
	if err != nil {
		t.Fatalf("manual get_tag returned error: %v", err)
	}
	manualPayload := manual.(map[string]any)
	manualManagement := manualPayload["management"].(map[string]any)
	if manualManagement["mode"] != "manual" {
		t.Fatalf("expected manual mode, got %v", manualManagement["mode"])
	}

	smart, err := tool.run(context.Background(), map[string]any{"tagid": 9001})
	if err != nil {
		t.Fatalf("smart get_tag returned error: %v", err)
	}
	smartPayload := smart.(map[string]any)
	if smartPayload["name"] != "Bali" {
		t.Fatalf("expected smart tag name Bali, got %v", smartPayload["name"])
	}
	smartManagement := smartPayload["management"].(map[string]any)
	if smartManagement["mode"] != "system_managed" {
		t.Fatalf("expected system_managed mode, got %v", smartManagement["mode"])
	}
	if smartManagement["read_only"] != true {
		t.Fatalf("expected smart tag read_only true, got %v", smartManagement["read_only"])
	}
}

func TestGetContactAddsNormalizedTagContext(t *testing.T) {
	server := newKlickTippTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/subscriber/123" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"subscriberid": "123",
			"email":        "test@example.com",
			"manual_tags": map[string]any{
				"21": "1759753677",
			},
			"smart_tags": map[string]any{
				"17": "1758821945",
				"18": "1758822131",
			},
		})
	})
	defer server.Close()

	tool := findTool(t, build(newTestClient(server.URL), config.Config{}), "get_contact")
	result, err := tool.run(context.Background(), map[string]any{"subscriberid": 123})
	if err != nil {
		t.Fatalf("get_contact returned error: %v", err)
	}

	payload := result.(map[string]any)
	tagContext := payload["tag_context"].(map[string]any)
	assertStringSlice(t, tagContext["manual_tag_ids"], []string{"21"})
	assertStringSlice(t, tagContext["smart_tag_ids"], []string{"17", "18"})
	assertStringSlice(t, tagContext["all_tag_ids"], []string{"17", "18", "21"})
	if tagContext["smart_tag_catalog_supported"] != false {
		t.Fatalf("expected smart tag catalog to be unsupported")
	}
}

func TestSearchTaggedContactsSupportsManualAndSmartTags(t *testing.T) {
	var seenTagIDs []string
	server := newKlickTippTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/subscriber/tagged" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		seenTagIDs = append(seenTagIDs, fmt.Sprint(payload["tagid"]))
		writeJSON(t, w, map[string]any{
			"tagid": payload["tagid"],
			"ids":   []string{"123", "456"},
		})
	})
	defer server.Close()

	tool := findTool(t, build(newTestClient(server.URL), config.Config{}), "search_tagged_contacts")

	for _, tagID := range []any{21, 9001} {
		if _, err := tool.run(context.Background(), map[string]any{"tagid": tagID}); err != nil {
			t.Fatalf("search_tagged_contacts(%v) returned error: %v", tagID, err)
		}
	}

	if len(seenTagIDs) != 2 || seenTagIDs[0] != "21" || seenTagIDs[1] != "9001" {
		t.Fatalf("unexpected tag IDs sent to search_tagged_contacts: %v", seenTagIDs)
	}
}

func TestKnownTagLookup404ExplainsSmartTagDiscoveryLimitation(t *testing.T) {
	server := newKlickTippTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tag/9999":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte(`{"message":"not found"}`)); err != nil {
				t.Fatalf("write response: %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/tag":
			writeJSON(t, w, map[string]string{
				"21": "Customers",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	tool := findTool(t, build(newTestClient(server.URL), config.Config{}), "get_tag")
	_, err := tool.run(context.Background(), map[string]any{"tagid": 9999})
	if err == nil {
		t.Fatal("expected get_tag to return an error")
	}
	if !strings.Contains(err.Error(), "Smart Tags cannot currently be discovered independently") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGetTagOmitsManagementWhenTypeCannotBeDerived(t *testing.T) {
	server := newKlickTippTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tag":
			writeJSON(t, w, map[string]string{
				"21": "Customers",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/tag/9001":
			writeJSON(t, w, map[string]any{
				"tagid": "9001",
				"name":  "Bali",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	tool := findTool(t, build(newTestClient(server.URL), config.Config{}), "get_tag")
	result, err := tool.run(context.Background(), map[string]any{"tagid": 9001})
	if err != nil {
		t.Fatalf("get_tag returned error: %v", err)
	}

	payload := result.(map[string]any)
	if _, exists := payload["management"]; exists {
		t.Fatalf("expected management to be omitted when type cannot be derived, got %v", payload["management"])
	}
}

func newTestClient(baseURL string) *api.Client {
	return api.NewClient(config.Config{
		BaseURL:      baseURL,
		AuthMode:     "partner",
		Username:     "tester",
		DeveloperKey: "aa",
		CustomerKey:  "customer",
	})
}

func newKlickTippTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := strings.TrimSpace(r.Header.Get("X-Un")); got != "tester" {
			t.Fatalf("expected X-Un header, got %q", got)
		}
		if strings.TrimSpace(r.Header.Get("X-Ci")) == "" {
			t.Fatal("expected X-Ci header to be set")
		}
		handler(w, r)
	}))
}

func findTool(t *testing.T, tools []toolSpec, name string) toolSpec {
	t.Helper()
	for _, tool := range tools {
		if tool.name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found", name)
	return toolSpec{}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func assertStringSlice(t *testing.T, raw any, want []string) {
	t.Helper()
	switch items := raw.(type) {
	case []string:
		compareStringSlices(t, items, want)
	case []any:
		got := make([]string, 0, len(items))
		for _, item := range items {
			got = append(got, fmt.Sprint(item))
		}
		compareStringSlices(t, got, want)
	default:
		t.Fatalf("expected []string or []any, got %T", raw)
	}
}

func compareStringSlices(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %v want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected items: got %v want %v", got, want)
		}
	}
}
