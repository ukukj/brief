package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFetch_ParsesForecast(t *testing.T) {
	raw, err := os.ReadFile("testdata/vilage_fcst.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("serviceKey") != "wkey" {
			t.Errorf("serviceKey not forwarded: %q", r.URL.Query().Get("serviceKey"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:0800" } // fcstDate:fcstTime 선택 기준

	f, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.TempC != 14 {
		t.Errorf("TempC = %d, want 14", f.TempC)
	}
	if f.Sky != "구름많음" {
		t.Errorf("Sky = %q, want 구름많음", f.Sky)
	}
	if f.MinTempC != 11 || f.MaxTempC != 23 {
		t.Errorf("min/max = %d/%d, want 11/23", f.MinTempC, f.MaxTempC)
	}
	if f.PrecipMaxPOP != 30 {
		t.Errorf("PrecipMaxPOP = %d, want 30", f.PrecipMaxPOP)
	}
	if f.PrecipKind != "" {
		t.Errorf("PrecipKind = %q, want \"\" (no rain — fixture PTY=0)", f.PrecipKind)
	}
	if len(f.PrecipRanges) != 0 {
		t.Errorf("PrecipRanges = %+v, want empty (no rain — fixture PTY=0)", f.PrecipRanges)
	}
}

func TestFetch_APIErrorCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"03","resultMsg":"NODATA_ERROR"},"body":{}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on non-00 resultCode")
	}
}

func TestFetch_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>nope</html>"))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	if _, err := c.Fetch(context.Background()); err == nil {
		t.Fatal("expected error on malformed (non-JSON) body")
	}
}

func TestFetch_MultiDay_TMNTMX_TargetDay(t *testing.T) {
	// 기상청 05:00 발표는 ~3일치를 반환한다. TMN/TMX는 날짜별로 1건씩 존재.
	// 가드 없이 덮어쓰면 마지막 날(≈+2d)의 min/max가 남아 오늘 값과 어긋난다.
	// 이 테스트는 대상일(20260518)의 TMN=18 / TMX=28 이 선택되어야 함을 고정한다.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00","resultMsg":"OK"},` +
			`"body":{"items":{"item":[` +
			`{"category":"TMP","fcstDate":"20260518","fcstTime":"1700","fcstValue":"27"},` +
			`{"category":"SKY","fcstDate":"20260518","fcstTime":"1700","fcstValue":"1"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"1700","fcstValue":"0"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"1700","fcstValue":"20"},` +
			`{"category":"TMN","fcstDate":"20260518","fcstTime":"0600","fcstValue":"18"},` +
			`{"category":"TMX","fcstDate":"20260518","fcstTime":"1500","fcstValue":"28"},` +
			`{"category":"TMN","fcstDate":"20260519","fcstTime":"0600","fcstValue":"17"},` +
			`{"category":"TMX","fcstDate":"20260519","fcstTime":"1500","fcstValue":"25"},` +
			`{"category":"TMN","fcstDate":"20260520","fcstTime":"0600","fcstValue":"16"},` +
			`{"category":"TMX","fcstDate":"20260520","fcstTime":"1500","fcstValue":"22"}` +
			`]}}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:1700" }

	f, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.TempC != 27 {
		t.Errorf("TempC = %d, want 27", f.TempC)
	}
	if f.Sky != "맑음" {
		t.Errorf("Sky = %q, want 맑음", f.Sky)
	}
	if f.MinTempC != 18 {
		t.Errorf("MinTempC = %d, want 18 (target day, not +2d)", f.MinTempC)
	}
	if f.MaxTempC != 28 {
		t.Errorf("MaxTempC = %d, want 28 (target day, not +2d)", f.MaxTempC)
	}
}

func TestFetch_RequestsPrevDay2300(t *testing.T) {
	// 기상청은 시간이 지나면 당일 TMN(일최저)을 발표에서 제외하므로,
	// 실행 시각과 무관하게 "전날 23:00 발표"를 받아야 한다.
	// 이 테스트는 base_date=<전날>, base_time=2300 요청을 고정한다.
	var gotBaseDate, gotBaseTime string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBaseDate = r.URL.Query().Get("base_date")
		gotBaseTime = r.URL.Query().Get("base_time")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00","resultMsg":"OK"},` +
			`"body":{"items":{"item":[` +
			`{"category":"TMP","fcstDate":"20260518","fcstTime":"0800","fcstValue":"14"}` +
			`]}}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:0800" }

	if _, err := c.Fetch(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBaseDate != "20260517" {
		t.Errorf("base_date = %q, want 20260517 (prev day)", gotBaseDate)
	}
	if gotBaseTime != "2300" {
		t.Errorf("base_time = %q, want 2300", gotBaseTime)
	}
}

func TestFetch_SlotMiss_KeepsDailySummary(t *testing.T) {
	// 슬롯(want=23:00)에 매칭되는 TMP/SKY 항목은 없지만,
	// 당일 다른 시간대의 POP/PTY는 존재 → TempC/Sky는 zero, 일일 종합은 채워진다.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00","resultMsg":"OK"},` +
			`"body":{"items":{"item":[` +
			`{"category":"TMP","fcstDate":"20260518","fcstTime":"0800","fcstValue":"14"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"0800","fcstValue":"30"},` +
			`{"category":"SKY","fcstDate":"20260518","fcstTime":"0800","fcstValue":"3"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"0800","fcstValue":"0"}` +
			`]}}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:2300" } // 응답에 없는 슬롯 → TMP/SKY 미스

	f, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.TempC != 0 || f.Sky != "" {
		t.Errorf("slot miss TempC/Sky = %d/%q, want 0/\"\"", f.TempC, f.Sky)
	}
	if f.PrecipMaxPOP != 30 {
		t.Errorf("PrecipMaxPOP = %d, want 30 (from 08:00 item)", f.PrecipMaxPOP)
	}
	if f.PrecipKind != "" || len(f.PrecipRanges) != 0 {
		t.Errorf("expected no rain (PTY=0), got Kind=%q Ranges=%+v", f.PrecipKind, f.PrecipRanges)
	}
}

func TestFetch_Precip_SingleContinuousRange(t *testing.T) {
	// PTY=1(비) 시간대 08–10 연속 + 그 외는 PTY=0 → 단일 구간 [8,10], 최대 POP=70.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00","resultMsg":"OK"},` +
			`"body":{"items":{"item":[` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"0700","fcstValue":"0"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"0700","fcstValue":"20"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"0800","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"0800","fcstValue":"60"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"0900","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"0900","fcstValue":"70"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"1000","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"1000","fcstValue":"50"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"1100","fcstValue":"0"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"1100","fcstValue":"30"}` +
			`]}}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:0800" }

	f, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.PrecipKind != "비" {
		t.Errorf("PrecipKind = %q, want 비", f.PrecipKind)
	}
	if f.PrecipMaxPOP != 70 {
		t.Errorf("PrecipMaxPOP = %d, want 70", f.PrecipMaxPOP)
	}
	if len(f.PrecipRanges) != 1 || f.PrecipRanges[0] != (HourRange{Start: 8, End: 10}) {
		t.Errorf("PrecipRanges = %+v, want [{8 10}]", f.PrecipRanges)
	}
}

func TestFetch_Precip_MultipleRanges(t *testing.T) {
	// 09–10 비 + 11 멈춤 + 14–15 비 → 두 구간 [9,10], [14,15].
	items := []string{
		`{"category":"PTY","fcstDate":"20260518","fcstTime":"0800","fcstValue":"0"}`,
		`{"category":"POP","fcstDate":"20260518","fcstTime":"0800","fcstValue":"10"}`,
		`{"category":"PTY","fcstDate":"20260518","fcstTime":"0900","fcstValue":"1"}`,
		`{"category":"POP","fcstDate":"20260518","fcstTime":"0900","fcstValue":"50"}`,
		`{"category":"PTY","fcstDate":"20260518","fcstTime":"1000","fcstValue":"1"}`,
		`{"category":"POP","fcstDate":"20260518","fcstTime":"1000","fcstValue":"60"}`,
		`{"category":"PTY","fcstDate":"20260518","fcstTime":"1100","fcstValue":"0"}`,
		`{"category":"POP","fcstDate":"20260518","fcstTime":"1100","fcstValue":"30"}`,
		`{"category":"PTY","fcstDate":"20260518","fcstTime":"1400","fcstValue":"1"}`,
		`{"category":"POP","fcstDate":"20260518","fcstTime":"1400","fcstValue":"55"}`,
		`{"category":"PTY","fcstDate":"20260518","fcstTime":"1500","fcstValue":"1"}`,
		`{"category":"POP","fcstDate":"20260518","fcstTime":"1500","fcstValue":"45"}`,
		`{"category":"PTY","fcstDate":"20260518","fcstTime":"1600","fcstValue":"0"}`,
		`{"category":"POP","fcstDate":"20260518","fcstTime":"1600","fcstValue":"20"}`,
	}
	body := `{"response":{"header":{"resultCode":"00","resultMsg":"OK"},"body":{"items":{"item":[` +
		joinComma(items) + `]}}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:0900" }

	f, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.PrecipKind != "비" {
		t.Errorf("PrecipKind = %q, want 비", f.PrecipKind)
	}
	if f.PrecipMaxPOP != 60 {
		t.Errorf("PrecipMaxPOP = %d, want 60", f.PrecipMaxPOP)
	}
	want := []HourRange{{Start: 9, End: 10}, {Start: 14, End: 15}}
	if len(f.PrecipRanges) != len(want) ||
		f.PrecipRanges[0] != want[0] || f.PrecipRanges[1] != want[1] {
		t.Errorf("PrecipRanges = %+v, want %+v", f.PrecipRanges, want)
	}
}

func TestFetch_Precip_WindowClipsLateNight(t *testing.T) {
	// 비가 22시와 23시에 예보된 상황.
	// 강수 종합은 08~22시만 보므로 23시 비는 무시된다.
	// 기대: 강수 구간은 22시 한 칸({22,22}), 최대 강수확률도
	// 22시의 80%만 — 23시의 90%는 범위 밖이라 제외된다.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00","resultMsg":"OK"},` +
			`"body":{"items":{"item":[` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"2100","fcstValue":"0"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"2100","fcstValue":"20"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"2200","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"2200","fcstValue":"80"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"2300","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"2300","fcstValue":"90"}` +
			`]}}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:2200" }

	f, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(f.PrecipRanges) != 1 || f.PrecipRanges[0] != (HourRange{Start: 22, End: 22}) {
		t.Errorf("PrecipRanges = %+v, want [{22 22}] (23시는 윈도우 밖)", f.PrecipRanges)
	}
	if f.PrecipMaxPOP != 80 {
		t.Errorf("PrecipMaxPOP = %d, want 80 (23시=90은 윈도우 밖)", f.PrecipMaxPOP)
	}
}

func TestFetch_Precip_WindowExcludesEarlyMorning(t *testing.T) {
	// 새벽 03–05시에만 비. 활동시간대 윈도우(08–22시) 밖이라 종합에서 제외 →
	// 무강수(PrecipKind="", PrecipRanges 빈, PrecipMaxPOP=0)로 처리.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"header":{"resultCode":"00","resultMsg":"OK"},` +
			`"body":{"items":{"item":[` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"0300","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"0300","fcstValue":"60"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"0400","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"0400","fcstValue":"70"},` +
			`{"category":"PTY","fcstDate":"20260518","fcstTime":"0500","fcstValue":"1"},` +
			`{"category":"POP","fcstDate":"20260518","fcstTime":"0500","fcstValue":"50"}` +
			`]}}}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "wkey", 60, 127)
	c.Now = func() string { return "20260518:0800" }

	f, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.PrecipKind != "" || len(f.PrecipRanges) != 0 {
		t.Errorf("early-morning rain must be excluded: Kind=%q Ranges=%+v", f.PrecipKind, f.PrecipRanges)
	}
	if f.PrecipMaxPOP != 0 {
		t.Errorf("PrecipMaxPOP = %d, want 0 (새벽 POP는 윈도우 밖)", f.PrecipMaxPOP)
	}
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}
