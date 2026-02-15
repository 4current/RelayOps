package main_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/4current/relayops/internal/ops"
)

func TestInitRuntime(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ops.InitRuntime(ctx); err != nil {
		t.Fatalf("InitRuntime: %v", err)
	}
}

func TestDoctor(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("HOME", tmp)
	_ = os.Setenv("USERPROFILE", tmp)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := ops.Doctor(ctx); err != nil {
		t.Fatalf("Doctor: %v", err)
	}
}
