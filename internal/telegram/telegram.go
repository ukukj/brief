// Package telegram은 Telegram Bot API sendMessage로 알림을 보낸다.
package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client는 Telegram 봇 클라이언트.
type Client struct {
	baseURL string // 예: https://api.telegram.org (테스트는 httptest URL 주입)
	token   string
	chatID  string
	http    *http.Client
}

// New는 클라이언트를 만든다.
func New(baseURL, token, chatID string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		chatID:  chatID,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type sendResp struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// Send는 chat_id로 plain text 메시지를 보낸다(웹 미리보기 끔).
func (c *Client) Send(ctx context.Context, text string) error {
	form := url.Values{}
	form.Set("chat_id", c.chatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")

	endpoint := c.baseURL + "/bot" + c.token + "/sendMessage"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			return fmt.Errorf("telegram: build request: %v", ue.Err)
		}
		return fmt.Errorf("telegram: build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		// url.Error는 요청 URL(봇 토큰 포함)을 담으므로 토큰 노출 방지를 위해 벗긴다.
		var ue *url.Error
		if errors.As(err, &ue) {
			return fmt.Errorf("telegram: http: %v", ue.Err)
		}
		return fmt.Errorf("telegram: http: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var sr sendResp
	if err := json.Unmarshal(body, &sr); err != nil {
		return fmt.Errorf("telegram: decode resp: %w (status=%d)", err, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || !sr.OK {
		return fmt.Errorf("telegram: send failed (status=%d ok=%v desc=%s)",
			resp.StatusCode, sr.OK, sr.Description)
	}
	return nil
}
