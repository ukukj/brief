package news

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func rssServer(t *testing.T, items string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` +
			`<rss version="2.0"><channel><title>t</title><link>http://x</link><description>d</description>` +
			items + `</channel></rss>`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func item(title, link string) string {
	return `<item><title>` + title + `</title><link>` + link + `</link></item>`
}

func titles(hs []Headline) []string {
	out := make([]string, len(hs))
	for i, h := range hs {
		out[i] = h.Title
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFetch_RoundRobin(t *testing.T) {
	econ := rssServer(t, item("E1", "http://e/1")+item("E2", "http://e/2")+item("E3", "http://e/3"))
	it := rssServer(t, item("I1", "http://i/1")+item("I2", "http://i/2"))

	c := New([]string{econ.URL, it.URL}, 3)
	hs, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := titles(hs); !eq(got, []string{"E1", "I1", "E2"}) {
		t.Errorf("round-robin = %v, want [E1 I1 E2]", got)
	}
}

func TestFetch_DedupeNormalizedTitle(t *testing.T) {
	econ := rssServer(t, item("(종합) 같은 기사", "http://e/1")+item("[속보] 같은 기사", "http://e/2")+item("다른 기사", "http://e/3"))
	it := rssServer(t, item("아이티 기사", "http://i/1"))

	c := New([]string{econ.URL, it.URL}, 3)
	hs, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := titles(hs)
	if len(got) != 3 || got[0] != "(종합) 같은 기사" || got[1] != "아이티 기사" || got[2] != "다른 기사" {
		t.Errorf("dedupe round-robin = %v", got)
	}
}

func TestFetch_BoilerplateExcluded(t *testing.T) {
	econ := rssServer(t,
		item("[표] 코스피 종목별 시세", "http://e/0")+
			item("코스피 2,600선 마감…외국인 순매도", "http://e/1")+
			item("[부고] 홍길동씨 별세", "http://e/2")+
			item("진짜 경제 기사", "http://e/3"))
	it := rssServer(t, item("진짜 IT 기사", "http://i/1"))

	c := New([]string{econ.URL, it.URL}, 3)
	hs, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, h := range hs {
		if h.Title != "진짜 경제 기사" && h.Title != "진짜 IT 기사" {
			t.Errorf("boilerplate leaked: %q", h.Title)
		}
	}
	if got := titles(hs); !eq(got, []string{"진짜 경제 기사", "진짜 IT 기사"}) {
		t.Errorf("after filter = %v, want [진짜 경제 기사, 진짜 IT 기사]", got)
	}
}

func TestFetch_PartialFailureOneFeed(t *testing.T) {
	it := rssServer(t, item("I1", "http://i/1")+item("I2", "http://i/2")+item("I3", "http://i/3"))
	c := New([]string{"http://127.0.0.1:1/dead.xml", it.URL}, 3)
	hs, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("partial failure must not error: %v", err)
	}
	if got := titles(hs); !eq(got, []string{"I1", "I2", "I3"}) {
		t.Errorf("partial = %v, want [I1 I2 I3]", got)
	}
}

func TestFetch_AllFeedsFail(t *testing.T) {
	c := New([]string{"http://127.0.0.1:1/a.xml", "http://127.0.0.1:1/b.xml"}, 3)
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("expected error when all feeds fail")
	}
}
