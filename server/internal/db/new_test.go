package db_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rrochlin/an-amazing-adventure/internal/db"
)

// http-games and world-gen never use the connections table, so a client
// without CONNECTIONS_TABLE must work until a connection method is called.
// No AWS call is made: the check runs first.
func TestNewWithoutConnectionsTable(t *testing.T) {
	t.Setenv("SESSIONS_TABLE", "test-sessions")
	t.Setenv("CONNECTIONS_TABLE", "")
	t.Setenv("AWS_REGION", "us-west-2")
	c, err := db.New(context.Background())
	if err != nil {
		t.Fatalf("db.New without CONNECTIONS_TABLE: %v", err)
	}
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), "CONNECTIONS_TABLE") {
			t.Fatalf("a connection method should panic naming CONNECTIONS_TABLE, got %v", r)
		}
	}()
	_, _ = c.GetConnection(context.Background(), "conn-1")
}
