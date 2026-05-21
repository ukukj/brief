// Command brief는 스케줄 진입점: 설정을 읽어 아침 브리핑을 1회 발송한다.
package main

import (
	"context"
	"log"
	"os"
	"time"
	_ "time/tzdata" // distroless/static에 zoneinfo 없어 weather.go time.LoadLocation 패닉 방지

	"morning-brief/internal/brief"
	"morning-brief/internal/config"
	"morning-brief/internal/news"
	"morning-brief/internal/telegram"
	"morning-brief/internal/weather"
)

const (
	kmaBase     = "http://apis.data.go.kr/1360000/VilageFcstInfoService_2.0/getVilageFcst"
	telegramAPI = "https://api.telegram.org"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	deps := brief.Deps{
		Weather:  weather.New(kmaBase, cfg.WeatherServiceKey, cfg.WeatherNx, cfg.WeatherNy),
		News:     news.New(cfg.NewsFeeds, cfg.NewsTopN),
		Notifier: telegram.New(telegramAPI, cfg.TelegramBotToken, cfg.TelegramChatID),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := brief.Run(ctx, deps); err != nil {
		log.Fatalf("brief failed: %v", err)
	}
}
