// Package weather는 기상청 단기예보(getVilageFcst)를 호출·파싱한다.
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// HourRange는 연속된 강수 시간 구간(폐구간, 0–23시 정수).
type HourRange struct {
	Start int
	End   int
}

// Forecast는 아침 발송에 쓸 핵심 예보값.
//
// TempC/Sky는 기준 시각(Now) 슬롯의 값이고, MinTempC/MaxTempC와
// PrecipMaxPOP/PrecipKind/PrecipRanges는 당일(wantDate) 전체를 종합한다.
type Forecast struct {
	TempC        int
	Sky          string
	MinTempC     int
	MaxTempC     int
	PrecipMaxPOP int         // 당일 시간대별 POP 중 최대값(%)
	PrecipKind   string      // 강수 형태 라벨("비"/"눈"/"비/눈"/"소나기"). 무강수면 ""
	PrecipRanges []HourRange // PTY!=0인 연속 시간 구간들(시작·종료시, 폐구간). 무강수면 빈 슬라이스
}

// Client는 기상청 단기예보 클라이언트.
type Client struct {
	baseURL    string
	serviceKey string
	nx, ny     int
	http       *http.Client
	// Now는 "fcstDate:fcstTime" 형식으로 기준 시각을 반환(테스트 주입). 기본은 한국시 다음 정시.
	Now func() string
}

// New는 클라이언트를 만든다. baseURL은 getVilageFcst 엔드포인트(끝 슬래시 없이).
func New(baseURL, serviceKey string, nx, ny int) *Client {
	return &Client{
		baseURL:    baseURL,
		serviceKey: serviceKey,
		nx:         nx,
		ny:         ny,
		http:       &http.Client{Timeout: 10 * time.Second},
		Now:        defaultNow,
	}
}

func defaultNow() string {
	loc, _ := time.LoadLocation("Asia/Seoul")
	t := time.Now().In(loc)
	return t.Format("20060102") + ":" + fmt.Sprintf("%02d00", t.Hour())
}

type apiResponse struct {
	Response struct {
		Header struct {
			ResultCode string `json:"resultCode"`
			ResultMsg  string `json:"resultMsg"`
		} `json:"header"`
		Body struct {
			Items struct {
				Item []item `json:"item"`
			} `json:"items"`
		} `json:"body"`
	} `json:"response"`
}

type item struct {
	Category  string `json:"category"`
	FcstDate  string `json:"fcstDate"`
	FcstTime  string `json:"fcstTime"`
	FcstValue string `json:"fcstValue"`
}

var skyText = map[string]string{"1": "맑음", "3": "구름많음", "4": "흐림"}
var ptyText = map[string]string{"0": "없음", "1": "비", "2": "비/눈", "3": "눈", "4": "소나기"}

// Fetch는 예보를 받아 기준 시각(Now)에 가장 가까운 슬롯의 값(TempC/Sky)과
// 당일 전체의 종합값(MinTempC/MaxTempC/PrecipMaxPOP/PrecipKind/PrecipRanges)을 파싱한다.
func (c *Client) Fetch(ctx context.Context) (Forecast, error) {
	q := url.Values{}
	q.Set("serviceKey", c.serviceKey)
	q.Set("dataType", "JSON")
	q.Set("numOfRows", "1000")
	q.Set("pageNo", "1")
	q.Set("nx", strconv.Itoa(c.nx))
	q.Set("ny", strconv.Itoa(c.ny))

	now := c.Now()
	// 전날 23시 발표를 사용한다: 기상청은 시간이 지나면 당일 TMN(일최저)을
	// 발표에서 제외하므로(05시 발표엔 당일 TMN 없음), 실행 시각과 무관하게
	// 당일 TMN+TMX+24h가 온전한 "전날 23:00 발표"를 받는다.
	baseDate := ""
	if len(now) >= 8 {
		if d, err := time.Parse("20060102", now[:8]); err == nil {
			baseDate = d.AddDate(0, 0, -1).Format("20060102")
		}
	}
	q.Set("base_date", baseDate)
	q.Set("base_time", "2300")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"?"+q.Encode(), nil)
	if err != nil {
		return Forecast{}, fmt.Errorf("weather: build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Forecast{}, fmt.Errorf("weather: http: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Forecast{}, fmt.Errorf("weather: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Forecast{}, fmt.Errorf("weather: http status %d (body=%.200s)", resp.StatusCode, string(body))
	}

	var ar apiResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return Forecast{}, fmt.Errorf("weather: decode: %w (body=%.200s)", err, string(body))
	}
	if ar.Response.Header.ResultCode != "00" {
		return Forecast{}, fmt.Errorf("weather: api error %s: %s",
			ar.Response.Header.ResultCode, ar.Response.Header.ResultMsg)
	}

	want := ""
	if len(now) >= 13 {
		want = now[:8] + now[9:13] // "YYYYMMDD"+"HHMM"
	}
	// TMN/TMX와 일일 강수 종합은 날짜 단위 — 대상일만 사용(다일치 응답 마지막날 덮어쓰기 방지).
	wantDate := ""
	if len(want) >= 8 {
		wantDate = want[:8]
	}

	// 당일 시간대별 POP/PTY를 모은다(시:00 단위, 0–23).
	type hourly struct {
		pop, pty   string
		havePop    bool
		havePtyRaw bool
	}
	dayHours := make(map[int]hourly)

	var f Forecast
	for _, it := range ar.Response.Body.Items.Item {
		atSlot := it.FcstDate+it.FcstTime == want
		switch it.Category {
		case "TMP":
			if atSlot {
				f.TempC = atoiSafe(it.FcstValue)
			}
		case "SKY":
			if atSlot {
				f.Sky = skyText[it.FcstValue]
			}
		case "TMN":
			if it.FcstDate == wantDate {
				f.MinTempC = atoiSafe(it.FcstValue)
			}
		case "TMX":
			if it.FcstDate == wantDate {
				f.MaxTempC = atoiSafe(it.FcstValue)
			}
		case "POP":
			if it.FcstDate == wantDate {
				h := parseHour(it.FcstTime)
				if h >= 0 {
					e := dayHours[h]
					e.pop = it.FcstValue
					e.havePop = true
					dayHours[h] = e
				}
			}
		case "PTY":
			if it.FcstDate == wantDate {
				h := parseHour(it.FcstTime)
				if h >= 0 {
					e := dayHours[h]
					e.pty = it.FcstValue
					e.havePtyRaw = true
					dayHours[h] = e
				}
			}
		}
	}

	// 일일 강수 종합: 최대 POP / 첫 등장 PTY 라벨 / 연속 PTY!=0 구간들.
	maxPOP := 0
	var ranges []HourRange
	firstPTY := ""
	inRange := false
	start := 0
	for h := 0; h < 24; h++ {
		e := dayHours[h]
		if e.havePop {
			if v := atoiSafe(e.pop); v > maxPOP {
				maxPOP = v
			}
		}
		isPrecip := e.havePtyRaw && e.pty != "" && e.pty != "0"
		if isPrecip && firstPTY == "" {
			firstPTY = e.pty
		}
		switch {
		case isPrecip && !inRange:
			start = h
			inRange = true
		case !isPrecip && inRange:
			ranges = append(ranges, HourRange{Start: start, End: h - 1})
			inRange = false
		}
	}
	if inRange {
		ranges = append(ranges, HourRange{Start: start, End: 23})
	}

	f.PrecipMaxPOP = maxPOP
	if firstPTY != "" {
		f.PrecipKind = ptyText[firstPTY]
	}
	f.PrecipRanges = ranges
	return f, nil
}

func parseHour(fcstTime string) int {
	if len(fcstTime) < 2 {
		return -1
	}
	h, err := strconv.Atoi(fcstTime[:2])
	if err != nil || h < 0 || h > 23 {
		return -1
	}
	return h
}

func atoiSafe(s string) int {
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return int(v)
	}
	return 0
}
