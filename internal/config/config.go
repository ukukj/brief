// Package config는 환경변수에서 앱 설정을 로드하고 검증한다.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Config는 앱 실행에 필요한 모든 설정값을 담는다.
type Config struct {
	WeatherServiceKey string
	WeatherNx         int
	WeatherNy         int
	NewsFeeds         []string
	NewsTopN          int
	TelegramBotToken  string
	TelegramChatID    string
}

// Getenv는 os.Getenv 시그니처. 테스트에서 주입 가능.
type Getenv func(string) string

// Load는 getenv로 설정을 읽고 검증한다.
func Load(getenv Getenv) (*Config, error) {
	c := &Config{
		WeatherServiceKey: getenv("WEATHER_SERVICE_KEY"),
		TelegramBotToken:  getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:    getenv("TELEGRAM_CHAT_ID"),
	}

	var err error
	if c.WeatherNx, err = mustInt(getenv, "WEATHER_NX"); err != nil {
		return nil, err
	}
	if c.WeatherNy, err = mustInt(getenv, "WEATHER_NY"); err != nil {
		return nil, err
	}
	if c.NewsTopN, err = mustInt(getenv, "NEWS_TOP_N"); err != nil {
		return nil, err
	}

	for _, f := range strings.Split(getenv("NEWS_FEEDS"), ",") {
		if s := strings.TrimSpace(f); s != "" {
			c.NewsFeeds = append(c.NewsFeeds, s)
		}
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
	missing := func(name, val string) error {
		if val == "" {
			return fmt.Errorf("config: required env %s is empty", name)
		}
		return nil
	}
	for _, e := range []error{
		missing("WEATHER_SERVICE_KEY", c.WeatherServiceKey),
		missing("TELEGRAM_BOT_TOKEN", c.TelegramBotToken),
		missing("TELEGRAM_CHAT_ID", c.TelegramChatID),
	} {
		if e != nil {
			return e
		}
	}
	if len(c.NewsFeeds) == 0 {
		return fmt.Errorf("config: NEWS_FEEDS is empty")
	}
	if c.NewsTopN <= 0 {
		return fmt.Errorf("config: NEWS_TOP_N must be > 0")
	}
	return nil
}

func mustInt(getenv Getenv, name string) (int, error) {
	raw := getenv(name)
	if raw == "" {
		return 0, fmt.Errorf("config: required env %s is empty", name)
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("config: %s must be integer: %w", name, err)
	}
	return n, nil
}
