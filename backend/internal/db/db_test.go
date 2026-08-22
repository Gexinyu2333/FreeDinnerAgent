package db

import (
	"context"
	"testing"
)

func TestConnectRejectsInvalidDatabaseURL(t *testing.T) {
	if _, err := Connect(context.Background(), "://not-a-valid-url"); err == nil {
		t.Fatal("Connect should reject invalid database URLs before opening a pool")
	}
}
