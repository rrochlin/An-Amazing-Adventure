package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func assertPanicsWithEnvAbsent(t *testing.T, envVar string, fn func()) {
	t.Helper()
	t.Setenv(envVar, "")
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic for missing %s, but handler did not panic", envVar)
			return
		}
		msg := ""
		switch v := r.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		}
		if !strings.Contains(msg, envVar) {
			t.Errorf("panic message %q does not mention %s", msg, envVar)
		}
	}()
	fn()
}

func makeWSReq(connID string) events.APIGatewayWebsocketProxyRequest {
	return events.APIGatewayWebsocketProxyRequest{
		RequestContext: events.APIGatewayWebsocketProxyRequestContext{
			ConnectionID: connID,
		},
	}
}

// ---- Required env var tests ----
// ws-disconnect requires: CONNECTIONS_TABLE

func TestHandlerDisconnect_MissingCONNECTIONS_TABLE_Panics(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	assertPanicsWithEnvAbsent(t, "CONNECTIONS_TABLE", func() {
		handler(context.Background(), makeWSReq("conn-1")) //nolint:errcheck
	})
}

// fakeConnections stands in for DynamoDB.
type fakeConnections struct {
	deleted []string
	err     error
}

func (f *fakeConnections) DeleteConnection(_ context.Context, id string) error {
	f.deleted = append(f.deleted, id)
	return f.err
}

func useDB(t *testing.T, db connections, err error) {
	t.Helper()
	orig := newDBClient
	newDBClient = func(context.Context) (connections, error) { return db, err }
	t.Cleanup(func() { newDBClient = orig })
}

func TestHandlerDisconnect_DeletesTheConnection(t *testing.T) {
	fake := &fakeConnections{}
	useDB(t, fake, nil)
	resp, err := handler(context.Background(), makeWSReq("conn-to-clean-up"))
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("got %d %v", resp.StatusCode, err)
	}
	if len(fake.deleted) != 1 || fake.deleted[0] != "conn-to-clean-up" {
		t.Fatalf("deleted %v", fake.deleted)
	}
}

// Disconnect always returns 200: API Gateway ignores the response, and a
// non-200 would only cause retries.
func TestHandlerDisconnect_AlwaysReturns200(t *testing.T) {
	for name, tc := range map[string]struct {
		db      connections
		initErr error
		connID  string
	}{
		"delete fails":        {&fakeConnections{err: errors.New("throttled")}, nil, "conn-1"},
		"db client fails":     {nil, errors.New("no config"), "conn-1"},
		"empty connection id": {&fakeConnections{}, nil, ""},
	} {
		t.Run(name, func(t *testing.T) {
			useDB(t, tc.db, tc.initErr)
			resp, err := handler(context.Background(), makeWSReq(tc.connID))
			if err != nil || resp.StatusCode != 200 {
				t.Fatalf("got %d %v", resp.StatusCode, err)
			}
		})
	}
}
