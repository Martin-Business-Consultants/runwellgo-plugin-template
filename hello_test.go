package hello

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Martin-Business-Consultants/runwellgo/v3/plugin"
	"github.com/Martin-Business-Consultants/runwellgo/v3/plugintest"
)

// env is Runwell as Hello sees it: plugintest's stand-in, with one client.
type env struct{ plugintest.Env }

func (env) Records(_ context.Context, q plugin.Query) ([]map[string]any, error) {
	if q.Type == "Client" && q.ID == 7 {
		return []map[string]any{{"type": "Client", "id": int64(7), "name": "Harbor Dental", "status": "active"}}, nil
	}
	return nil, nil
}

var ted = &plugin.Caller{ID: 1, Name: "Ted", Role: "owner"}

func serve(t *testing.T, h *plugin.Host, method, export, body string, query map[string][]string) map[string]any {
	t.Helper()
	res, err := h.Serve(t.Context(), "hello", export, &plugin.Request{Method: method, Query: query, Body: []byte(body), Actor: ted}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(res.Body, &out); err != nil {
		t.Fatalf("%s: %s", err, res.Body)
	}
	return out
}

// A new greeting shows on every client's page, and the page counts the
// clients it saw arrive.
func TestGreetingOnClientsPage(t *testing.T) {
	h, _ := plugintest.Setup(t, env{}, Module())
	if got := serve(t, h, "GET", "client", "", map[string][]string{"record": {"Client:7"}})["text"]; got != "Hello, Harbor Dental!" {
		t.Fatalf("the panel: %v", got)
	}
	serve(t, h, "POST", "save", `{"greeting":{"message":"Welcome","note":"Glad you're here."}}`, nil)
	if got := serve(t, h, "GET", "client", "", map[string][]string{"record": {"Client:7"}})["text"]; got != "Welcome, Harbor Dental! Glad you're here." {
		t.Fatalf("after saving: %v", got)
	}
	h.Emit(t.Context(), "client.created", map[string]any{"subject_type": "Client", "subject_id": 7, "occurred_at": "2026-10-19T10:00:00Z"})
	if got := serve(t, h, "GET", "greeting", "", nil)["text"].(string); !strings.Contains(got, "welcomed 1 new client ") {
		t.Fatalf("the page: %v", got)
	}
}

// A blank greeting is refused, field by field.
func TestBlankGreeting(t *testing.T) {
	h, _ := plugintest.Setup(t, env{}, Module())
	res, err := h.Serve(t.Context(), "hello", "save", &plugin.Request{Method: "POST", Body: []byte(`{"greeting":{"message":" "}}`), Actor: ted}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != 422 || !strings.Contains(string(res.Body), "can't be blank") {
		t.Fatalf("%d %s", res.Status, res.Body)
	}
}
