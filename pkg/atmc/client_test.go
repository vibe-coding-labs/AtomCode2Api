package atmc

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:13456")
	if c.BaseURL != "http://localhost:13456" {
		t.Errorf("expected base url http://localhost:13456, got %s", c.BaseURL)
	}
}

func TestNewClientTrailingSlash(t *testing.T) {
	c := NewClient("http://localhost:13456/")
	if c.BaseURL != "http://localhost:13456" {
		t.Errorf("expected http://localhost:13456 (no trailing slash), got %s", c.BaseURL)
	}
}
