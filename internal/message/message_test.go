package message

import (
	"os"
	"strings"
	"testing"

	"morning-brief/internal/news"
	"morning-brief/internal/weather"
)

func TestFormat_Golden(t *testing.T) {
	f := &weather.Forecast{
		TempC: 14, Sky: "구름많음", MinTempC: 11, MaxTempC: 23,
		PrecipMaxPOP: 30, PrecipKind: "", PrecipRanges: nil,
	}
	hs := []news.Headline{
		{Title: "첫번째 기사", Link: "https://news.example/1"},
		{Title: "두번째 기사", Link: "https://news.example/2"},
	}
	text := Format(f, hs, []string{"weather partial"})
	want, err := os.ReadFile("testdata/golden_full.txt")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if text != string(want) {
		t.Errorf("text mismatch:\n--- got ---\n%s\n--- want ---\n%s", text, string(want))
	}
}

func TestFormat_Precip_SingleRange(t *testing.T) {
	f := &weather.Forecast{
		TempC: 18, Sky: "흐림", MinTempC: 15, MaxTempC: 22,
		PrecipMaxPOP: 70, PrecipKind: "비",
		PrecipRanges: []weather.HourRange{{Start: 8, End: 23}},
	}
	text := Format(f, nil, nil)
	if !strings.Contains(text, "강수 비 08–23시 · 최대 70%\n") {
		t.Errorf("expected single-range precip line, got:\n%s", text)
	}
}

func TestFormat_Precip_MultipleRanges(t *testing.T) {
	f := &weather.Forecast{
		TempC: 18, Sky: "흐림", MinTempC: 15, MaxTempC: 22,
		PrecipMaxPOP: 60, PrecipKind: "비",
		PrecipRanges: []weather.HourRange{
			{Start: 9, End: 11},
			{Start: 14, End: 16},
		},
	}
	text := Format(f, nil, nil)
	if !strings.Contains(text, "강수 비 09–11시, 14–16시 · 최대 60%\n") {
		t.Errorf("expected multi-range precip line, got:\n%s", text)
	}
}

func TestFormat_Precip_None(t *testing.T) {
	f := &weather.Forecast{
		TempC: 20, Sky: "맑음", MinTempC: 14, MaxTempC: 25,
		PrecipMaxPOP: 10, PrecipKind: "", PrecipRanges: nil,
	}
	text := Format(f, nil, nil)
	if !strings.Contains(text, "강수 없음 · 최대 10%\n") {
		t.Errorf("expected no-precip line, got:\n%s", text)
	}
}

func TestFormat_NoWeather(t *testing.T) {
	hs := []news.Headline{{Title: "기사", Link: "https://x/1"}}
	text := Format(nil, hs, nil)
	if text == "" {
		t.Fatal("text should not be empty when news exists")
	}
	if strings.Contains(text, "[날씨]") {
		t.Error("weather section must be omitted when f == nil")
	}
	if !strings.Contains(text, "https://x/1") {
		t.Error("news URL must be in body")
	}
}
