package parser

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseWithTimeout_NoTimeout(t *testing.T) {
	expectedErr := errors.New("boom")
	err := parseWithTimeout(0, func() error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestParseWithTimeout_TimesOut(t *testing.T) {
	err := parseWithTimeout(10*time.Millisecond, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "parse timeout after") {
		t.Fatalf("expected timeout message, got %v", err)
	}
}

func TestParseWithTimeout_ReturnsParseError(t *testing.T) {
	expectedErr := errors.New("parse failed")
	err := parseWithTimeout(100*time.Millisecond, func() error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}
