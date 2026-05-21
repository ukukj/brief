package brief

import (
	"context"
	"errors"
	"testing"

	"morning-brief/internal/news"
	"morning-brief/internal/weather"
)

type fakeNotifier struct {
	sent    string
	sendErr error
}

func (f *fakeNotifier) Send(_ context.Context, text string) error {
	f.sent = text
	return f.sendErr
}

type fakeWeather struct {
	f   weather.Forecast
	err error
}

func (f *fakeWeather) Fetch(context.Context) (weather.Forecast, error) { return f.f, f.err }

type fakeNews struct {
	hs  []news.Headline
	err error
}

func (f *fakeNews) Fetch(context.Context) ([]news.Headline, error) { return f.hs, f.err }

func newDeps() (*fakeWeather, *fakeNews, *fakeNotifier) {
	return &fakeWeather{f: weather.Forecast{TempC: 14, Sky: "맑음"}},
		&fakeNews{hs: []news.Headline{{Title: "기사", Link: "https://x/1"}}},
		&fakeNotifier{}
}

func TestRun_Success(t *testing.T) {
	wx, nw, nt := newDeps()
	if err := Run(context.Background(), Deps{Weather: wx, News: nw, Notifier: nt}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nt.sent == "" {
		t.Error("notification not sent")
	}
}

func TestRun_WeatherFailsButStillSends(t *testing.T) {
	_, nw, nt := newDeps()
	wx := &fakeWeather{err: errors.New("kma down")}
	if err := Run(context.Background(), Deps{Weather: wx, News: nw, Notifier: nt}); err != nil {
		t.Fatalf("partial failure must not abort: %v", err)
	}
	if nt.sent == "" {
		t.Error("should still send with news only")
	}
}

func TestRun_NewsFailsButStillSends(t *testing.T) {
	wx, _, nt := newDeps()
	nw := &fakeNews{err: errors.New("rss down")}
	if err := Run(context.Background(), Deps{Weather: wx, News: nw, Notifier: nt}); err != nil {
		t.Fatalf("partial failure must not abort: %v", err)
	}
	if nt.sent == "" {
		t.Error("should still send with weather only")
	}
}

func TestRun_AllContentFailsIsFatal(t *testing.T) {
	_, _, nt := newDeps()
	wx := &fakeWeather{err: errors.New("kma down")}
	nw := &fakeNews{err: errors.New("rss down")}
	if err := Run(context.Background(), Deps{Weather: wx, News: nw, Notifier: nt}); err == nil {
		t.Fatal("should be fatal when both weather and news fail")
	}
}

func TestRun_SendFailsIsFatal(t *testing.T) {
	wx, nw, _ := newDeps()
	nt := &fakeNotifier{sendErr: errors.New("telegram 401")}
	if err := Run(context.Background(), Deps{Weather: wx, News: nw, Notifier: nt}); err == nil {
		t.Fatal("send failure must be fatal")
	}
}
