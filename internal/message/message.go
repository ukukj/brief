// Package message는 날씨/뉴스를 알림 본문 텍스트로 조립한다.
package message

import (
	"fmt"
	"strings"

	"morning-brief/internal/news"
	"morning-brief/internal/weather"
)

// Format은 본문 텍스트를 반환한다. f가 nil이면 날씨 섹션을 생략한다.
// 첫 줄은 오늘 날씨를 한 단어로 요약한다("☀️ 맑음", "🌧 비" 등).
// f == nil일 땐 기본 헤더("🌤 아침 브리핑")로 대체.
// 강수 줄은 활동시간대(08–22시) 종합: 강수 예보 시
// "강수 {Kind} HH–HH시[, HH–HH시…] · 최대 P%", 무강수 시 "강수 없음".
// 뉴스는 제목 다음 줄에 기사 URL을 포함한다(텔레그램이 자동 링크화).
// partialErrs가 있으면 경고 줄을 덧붙인다.
func Format(f *weather.Forecast, hs []news.Headline, partialErrs []string) string {
	var b strings.Builder
	b.WriteString(headline(f) + "\n\n")

	if f != nil {
		b.WriteString(fmt.Sprintf("[날씨] %s · %d℃ (최저 %d / 최고 %d)\n",
			f.Sky, f.TempC, f.MinTempC, f.MaxTempC))
		b.WriteString(formatPrecipLine(f))
		b.WriteString("\n")
	}

	b.WriteString("[뉴스]\n")
	for i, h := range hs {
		b.WriteString(fmt.Sprintf("%d. %s\n%s\n", i+1, h.Title, h.Link))
	}

	if len(partialErrs) > 0 {
		b.WriteString("\n⚠️ 일부 정보 누락: " + strings.Join(partialErrs, ", ") + "\n")
	}

	return b.String()
}

// 강수가 있으면 강수 종류, 없으면 SKY를 한 단어로 채택한다.
// 강수가 정보값이 더 커서 "흐림 + 비"인 날엔 "비"가 이긴다.
var precipIcon = map[string]string{
	"비":   "🌧",
	"소나기": "🌦",
	"눈":   "❄️",
	"비/눈": "🌨",
}

var skyIcon = map[string]string{
	"맑음":   "☀️",
	"구름많음": "⛅",
	"흐림":   "☁️",
}

func headline(f *weather.Forecast) string {
	if f == nil {
		return "🌤 아침 브리핑"
	}
	if f.PrecipKind != "" && len(f.PrecipRanges) > 0 {
		icon := precipIcon[f.PrecipKind]
		if icon == "" {
			icon = "🌧"
		}
		return icon + " " + f.PrecipKind
	}
	if f.Sky != "" {
		icon := skyIcon[f.Sky]
		if icon == "" {
			icon = "🌤"
		}
		return icon + " " + f.Sky
	}
	return "🌤 아침 브리핑"
}

func formatPrecipLine(f *weather.Forecast) string {
	if f.PrecipKind == "" || len(f.PrecipRanges) == 0 {
		return "강수 없음\n"
	}
	parts := make([]string, 0, len(f.PrecipRanges))
	for _, r := range f.PrecipRanges {
		parts = append(parts, fmt.Sprintf("%02d–%02d시", r.Start, r.End))
	}
	return fmt.Sprintf("강수 %s %s · 최대 %d%%\n",
		f.PrecipKind, strings.Join(parts, ", "), f.PrecipMaxPOP)
}
