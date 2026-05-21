// Package news는 RSS 피드에서 분야 균형(라운드로빈) + 노이즈 필터로 헤드라인을 모은다.
package news

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"
)

// Headline은 뉴스 한 건.
type Headline struct {
	Title string
	Link  string
}

// Client는 설정된 RSS 피드들에서 헤드라인을 모은다.
type Client struct {
	feeds  []string
	topN   int
	parser *gofeed.Parser
}

// New는 피드 URL 목록과 상위 개수로 Client를 만든다.
func New(feeds []string, topN int) *Client {
	return &Client{feeds: feeds, topN: topN, parser: gofeed.NewParser()}
}

// leadingTag는 제목 맨 앞의 [..] 또는 (..) 말머리 1개와 뒤 공백을 매칭한다.
var leadingTag = regexp.MustCompile(`^\s*[\[(][^\])]*[\])]\s*`)

// boilerplatePrefix: 원제목이 이 말머리로 시작하면 비기사로 제외(보수적 최소 집합).
var boilerplatePrefix = []string{
	"[표]", "[게시판]", "[부고]", "[인사]", "[동정]", "[알림]", "[고침]",
}

// marketClose: 코스피/코스닥으로 시작하며 마감/종가 포함하는 증시 시세표.
var marketClose = regexp.MustCompile(`^(코스피|코스닥).*(마감|종가)`)

// normalizeTitle은 앞쪽 말머리를 모두 떼고 공백/대소문자 정규화한 중복 판정 키.
func normalizeTitle(t string) string {
	s := strings.TrimSpace(t)
	for {
		n := leadingTag.ReplaceAllString(s, "")
		if n == s {
			break
		}
		s = n
	}
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// isBoilerplate는 표/게시판/부고/시세표 등 비기사면 true.
func isBoilerplate(title string) bool {
	t := strings.TrimSpace(title)
	for _, p := range boilerplatePrefix {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return marketClose.MatchString(t)
}

// Fetch는 피드별로 수집·필터한 뒤 피드 순서대로 라운드로빈하여 topN개를 반환한다.
// 일부 피드 실패는 허용하고(나머지로 진행), 모든 피드 실패 시에만 에러.
func (c *Client) Fetch(ctx context.Context) ([]Headline, error) {
	perFeed := make([][]Headline, len(c.feeds))
	seen := make(map[string]bool)
	var firstErr error
	okCount := 0

	for i, url := range c.feeds {
		feed, err := c.parser.ParseURLWithContext(url, ctx)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		okCount++
		for _, it := range feed.Items {
			if isBoilerplate(it.Title) {
				continue
			}
			key := normalizeTitle(it.Title)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			perFeed[i] = append(perFeed[i], Headline{Title: it.Title, Link: it.Link})
		}
	}

	if okCount == 0 {
		return nil, fmt.Errorf("news: all %d feeds failed: %w", len(c.feeds), firstErr)
	}

	var out []Headline
	for idx := 0; len(out) < c.topN; idx++ {
		progressed := false
		for f := range perFeed {
			if idx < len(perFeed[f]) {
				out = append(out, perFeed[f][idx])
				progressed = true
				if len(out) == c.topN {
					break
				}
			}
		}
		if !progressed {
			break
		}
	}
	return out, nil
}
