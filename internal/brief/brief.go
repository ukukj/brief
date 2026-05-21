// Package brief는 전체 흐름을 조립한다(모든 의존은 주입).
package brief

import (
	"context"
	"fmt"
	"log"
	"sync"

	"morning-brief/internal/message"
	"morning-brief/internal/news"
	"morning-brief/internal/weather"
)

// Notifier는 telegram.Client가 만족하는 인터페이스.
type Notifier interface {
	Send(ctx context.Context, text string) error
}

// WeatherClient는 weather.Client가 만족하는 인터페이스.
type WeatherClient interface {
	Fetch(ctx context.Context) (weather.Forecast, error)
}

// NewsClient는 news.Client가 만족하는 인터페이스.
type NewsClient interface {
	Fetch(ctx context.Context) ([]news.Headline, error)
}

// Deps는 Run의 주입 의존성.
type Deps struct {
	Weather  WeatherClient
	News     NewsClient
	Notifier Notifier
}

// Run은 전체 시나리오를 실행한다. 치명 오류 시 error 반환(호출자가 non-zero exit).
func Run(ctx context.Context, d Deps) error {
	var (
		wg          sync.WaitGroup
		fc          weather.Forecast
		fcOK        bool
		hs          []news.Headline
		partialErrs []string
		mu          sync.Mutex
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		f, e := d.Weather.Fetch(ctx)
		mu.Lock()
		defer mu.Unlock()
		if e != nil {
			partialErrs = append(partialErrs, "날씨")
			log.Printf("brief: weather fetch failed: %v", e)
			return
		}
		fc, fcOK = f, true
	}()
	go func() {
		defer wg.Done()
		h, e := d.News.Fetch(ctx)
		mu.Lock()
		defer mu.Unlock()
		if e != nil {
			partialErrs = append(partialErrs, "뉴스")
			log.Printf("brief: news fetch failed: %v", e)
			return
		}
		hs = h
	}()
	wg.Wait()

	if !fcOK && len(hs) == 0 {
		return fmt.Errorf("brief: both weather and news failed; nothing to send")
	}

	var fcPtr *weather.Forecast
	if fcOK {
		fcPtr = &fc
	}
	text := message.Format(fcPtr, hs, partialErrs)

	if err := d.Notifier.Send(ctx, text); err != nil {
		return fmt.Errorf("brief: send: %w", err)
	}
	log.Println("brief: sent successfully")
	return nil
}
