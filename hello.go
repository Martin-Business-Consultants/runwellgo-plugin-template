// Package hello is the smallest Runwell plugin, the example docs/plugins.md
// walks through: a note on every client's page (a panel), a page in the
// sidebar with a small form kept in its own storage, and an ear for one
// event. It asks for its storage and to read records, nothing else.
package hello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Martin-Business-Consultants/runwellgo/internal/plugin"
)

// Module is the plugin, for the core to register (internal/core/plugins.go).
func Module() *plugin.Module {
	return &plugin.Module{
		Manifest: plugin.Manifest{
			Key:         "hello",
			Name:        "Hello",
			Version:     "1.0.0",
			Author:      "Runwell",
			Description: "A note on every client's page, and a page of its own with a greeting you can change. The example plugin.",
			Permissions: plugin.Grants{Storage: true, RecordsRead: true},
			Hooks:       []string{"client.created"},
			Routes: []plugin.Route{
				{Method: "GET", Path: "/client", Export: "client"},
				{Method: "GET", Path: "/greeting", Export: "greeting"},
				{Method: "POST", Path: "/greeting", Export: "save"},
			},
			Screens: []plugin.Screen{
				{ID: "client", Title: "Hello", Source: "/client", Text: "text", Detail: &plugin.Detail{Items: "items"}},
				{ID: "greeting", Title: "Hello", Source: "/greeting", Text: "text",
					Form: &plugin.Form{Record: "greeting", Save: "/greeting", Submit: "Save greeting", Fields: []plugin.Field{
						{Name: "message", Label: "Greeting", Required: true, Placeholder: "Hello"},
						{Name: "note", Label: "Note", Type: "textarea", Help: "Shown under the greeting on every client's page."},
					}},
					Table: &plugin.Table{Rows: "history", Empty: "Nobody has changed the greeting yet.", Columns: []plugin.Column{
						{Key: "message", Label: "Greeting", Sub: "note"}, {Key: "by", Label: "By"}, {Key: "at", Label: "When", Time: true},
					}},
				},
			},
			Nav:    &plugin.NavPage{Label: "Hello", Page: "greeting", Icon: "hand"},
			Panels: []plugin.Panel{{Slot: "client_panel", Label: "Hello", Screen: "client"}},
		},
		Handlers: map[string]plugin.Handler{"client": client, "greeting": greeting, "save": save},
		OnEvent:  onEvent,
	}
}

// Greeting is what the form keeps, under "greeting".
type Greeting struct {
	Message string `json:"message"`
	Note    string `json:"note"`
	By      string `json:"by"`
	At      string `json:"at"`
}

func current(ctx context.Context, api *plugin.API) Greeting {
	g := Greeting{Message: "Hello"}
	if raw, ok, err := api.KVGet(ctx, "greeting"); err == nil && ok {
		_ = json.Unmarshal([]byte(raw), &g) // a damaged value: the default greeting
	}
	return g
}

// client is the panel on a client's page: the greeting, the client by
// name, and whether this plugin saw it arrive.
func client(ctx context.Context, api *plugin.API, req *plugin.Request) (*plugin.Response, error) {
	c, err := api.Record(ctx, req.Query.Get("record"))
	if r := plugin.Answer(err); r != nil {
		return r, nil
	} else if err != nil {
		return nil, err
	}
	g := current(ctx, api)
	text := fmt.Sprintf("%s, %s!", g.Message, c["name"])
	if g.Note != "" {
		text += " " + g.Note
	}
	items := []map[string]any{{"label": "Status", "value": c["status"], "badge": true}}
	if at, ok, _ := api.KVGet(ctx, fmt.Sprintf("welcomed:%v", c["id"])); ok {
		items = append(items, map[string]any{"label": "Welcomed", "value": at, "time": true})
	}
	return plugin.JSON(http.StatusOK, map[string]any{"text": text, "items": items}), nil
}

// greeting is the plugin's page: the greeting as a form, and who changed
// it before.
func greeting(ctx context.Context, api *plugin.API, _ *plugin.Request) (*plugin.Response, error) {
	g := current(ctx, api)
	history := []Greeting{}
	kvs, err := api.KVValues(ctx, "history:")
	if err != nil {
		return nil, err
	}
	for i := len(kvs) - 1; i >= 0 && len(history) < 10; i-- {
		var h Greeting
		if json.Unmarshal([]byte(kvs[i].Value), &h) == nil {
			history = append(history, h)
		}
	}
	welcomed, err := api.KVCount(ctx, "welcomed:")
	if err != nil {
		return nil, err
	}
	noun := "new clients"
	if welcomed == 1 {
		noun = "new client"
	}
	text := fmt.Sprintf("Every client's page says this, with the client's name. It has welcomed %d %s since it was switched on.", welcomed, noun)
	return plugin.JSON(http.StatusOK, map[string]any{"text": text, "greeting": map[string]any{"message": g.Message, "note": g.Note}, "history": history}), nil
}

// save keeps a new greeting, and the old one in the history.
func save(ctx context.Context, api *plugin.API, req *plugin.Request) (*plugin.Response, error) {
	var body struct {
		Greeting Greeting `json:"greeting"`
	}
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return plugin.JSON(http.StatusBadRequest, map[string]any{"error": "bad_request", "message": "Send {\"greeting\": {\"message\": …}}."}), nil //nolint:nilerr // the caller's mistake, answered
	}
	g := body.Greeting
	g.Message, g.Note = strings.TrimSpace(g.Message), strings.TrimSpace(g.Note)
	if g.Message == "" {
		return plugin.Invalid(map[string][]string{"message": {"can't be blank"}}), nil
	}
	if len(g.Message) > 100 {
		return plugin.Invalid(map[string][]string{"message": {"is too long (maximum is 100 characters)"}}), nil
	}
	g.By, g.At = req.Actor.Name, time.Now().UTC().Format(time.RFC3339)
	raw, _ := json.Marshal(g)
	if err := api.KVSet(ctx, "greeting", string(raw)); err != nil {
		return nil, err
	}
	if err := api.KVSet(ctx, "history:"+g.At, string(raw)); err != nil {
		return nil, err
	}
	return plugin.JSON(http.StatusOK, map[string]any{"message": "Greeting saved.", "greeting": g}), nil
}

// onEvent hears client.created and remembers when.
func onEvent(ctx context.Context, api *plugin.API, _ string, data json.RawMessage) error {
	var ev struct {
		SubjectID  int64  `json:"subject_id"`
		OccurredAt string `json:"occurred_at"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return err
	}
	return api.KVSet(ctx, fmt.Sprintf("welcomed:%d", ev.SubjectID), ev.OccurredAt)
}
