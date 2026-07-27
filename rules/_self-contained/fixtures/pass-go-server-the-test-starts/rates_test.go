package scenario

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestFetchReturnsTheRatesTheServiceSent(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"EUR":0.92,"GBP":0.84}`)
	}))
	defer service.Close()

	got, err := Fetch(context.Background(), service.URL)
	if err != nil {
		t.Fatalf("Fetch(%q) = %v; want no error", service.URL, err)
	}

	want := Table{"EUR": 0.92, "GBP": 0.84}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Fetch(%q) = %v; want %v", service.URL, got, want)
	}
}

func TestFetchReportsAServiceThatRefuses(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(service.Close)

	_, err := Fetch(context.Background(), service.URL)
	if err == nil {
		t.Fatalf("Fetch(%q) = nil error; want the refusal reported", service.URL)
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("Fetch(%q) = %q; want the status in the message", service.URL, err)
	}
}
