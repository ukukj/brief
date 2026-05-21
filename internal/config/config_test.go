package config

import "testing"

func TestLoad_Valid(t *testing.T) {
	env := map[string]string{
		"WEATHER_SERVICE_KEY": "wkey",
		"WEATHER_NX":          "60",
		"WEATHER_NY":          "124",
		"NEWS_FEEDS":          "https://a.com/rss,https://b.com/rss",
		"NEWS_TOP_N":          "3",
		"TELEGRAM_BOT_TOKEN":  "123:ABC",
		"TELEGRAM_CHAT_ID":    "7951915138",
	}
	get := func(k string) string { return env[k] }
	cfg, err := Load(get)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.WeatherNx != 60 || cfg.WeatherNy != 124 {
		t.Errorf("nx/ny = %d/%d, want 60/124", cfg.WeatherNx, cfg.WeatherNy)
	}
	if len(cfg.NewsFeeds) != 2 {
		t.Errorf("NewsFeeds len = %d, want 2", len(cfg.NewsFeeds))
	}
	if cfg.NewsTopN != 3 {
		t.Errorf("NewsTopN = %d, want 3", cfg.NewsTopN)
	}
	if cfg.TelegramBotToken != "123:ABC" || cfg.TelegramChatID != "7951915138" {
		t.Errorf("telegram = %q/%q", cfg.TelegramBotToken, cfg.TelegramChatID)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	get := func(k string) string { return "" }
	if _, err := Load(get); err == nil {
		t.Fatal("expected error for missing required env, got nil")
	}
}

func TestLoad_MissingTelegram(t *testing.T) {
	env := map[string]string{
		"WEATHER_SERVICE_KEY": "wkey",
		"WEATHER_NX":          "60",
		"WEATHER_NY":          "124",
		"NEWS_FEEDS":          "https://a.com/rss",
		"NEWS_TOP_N":          "3",
	}
	get := func(k string) string { return env[k] }
	if _, err := Load(get); err == nil {
		t.Fatal("expected error: telegram token/chat id required")
	}
}
