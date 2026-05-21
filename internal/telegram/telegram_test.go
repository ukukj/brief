package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSend_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/botTKN/sendMessage") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = r.ParseForm()
		if r.FormValue("chat_id") != "CID" {
			t.Errorf("chat_id = %q", r.FormValue("chat_id"))
		}
		if r.FormValue("text") != "본문" {
			t.Errorf("text = %q", r.FormValue("text"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "TKN", "CID")
	if err := c.Send(context.Background(), "본문"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_NotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"chat not found"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "TKN", "CID")
	if err := c.Send(context.Background(), "본문"); err == nil {
		t.Fatal("expected error on ok:false / non-200")
	}
}

func TestSend_HTTPErrorNoTokenLeak(t *testing.T) {
	// baseURL이 닿지 않는 곳 → http 에러. 에러 문자열에 토큰이 없어야 함.
	c := New("http://127.0.0.1:1", "SUPERSECRET", "CID")
	err := c.Send(context.Background(), "본문")
	if err == nil {
		t.Fatal("expected http error")
	}
	if strings.Contains(err.Error(), "SUPERSECRET") {
		t.Errorf("error leaked token: %v", err)
	}
}
