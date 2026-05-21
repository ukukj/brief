// Package message는 날씨/뉴스를 알림 본문 텍스트로 조립한다.
package message

import (
	"fmt"
	"strings"

	"morning-brief/internal/news"
	"morning-brief/internal/weather"
)

// Format은 본문 텍스트를 반환한다. f가 nil이면 날씨 섹션을 생략한다.
// 강수 줄은 당일 종합: 강수 예보 시 "강수 {Kind} HH–HH시[, HH–HH시…] · 최대 P%",
// 무강수 시 "강수 없음 · 최대 P%". 첫 줄(흐림·온도·최저/최고)은 슬롯 기준 그대로.
// 뉴스는 제목 다음 줄에 기사 URL을 포함한다(텔레그램이 자동 링크화).
// partialErrs가 있으면 경고 줄을 덧붙인다.
func Format(f *weather.Forecast, hs []news.Headline, partialErrs []string) string {
	var b strings.Builder
	b.WriteString("🌤 오늘 아침 브리핑\n\n")

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

func formatPrecipLine(f *weather.Forecast) string {
	if f.PrecipKind == "" || len(f.PrecipRanges) == 0 {
		return fmt.Sprintf("강수 없음 · 최대 %d%%\n", f.PrecipMaxPOP)
	}
	parts := make([]string, 0, len(f.PrecipRanges))
	for _, r := range f.PrecipRanges {
		parts = append(parts, fmt.Sprintf("%02d–%02d시", r.Start, r.End))
	}
	return fmt.Sprintf("강수 %s %s · 최대 %d%%\n",
		f.PrecipKind, strings.Join(parts, ", "), f.PrecipMaxPOP)
}
