# morning-brief

매일 아침 텔레그램으로 받는 1회성 브리핑 (날씨 + 경제·IT 헤드라인 3건).

- 언어: Go 1.22
- 배포 단위: 단일 정적 바이너리 + distroless 컨테이너
- 트리거: 외부 스케줄러(맥 `launchd` 또는 K8s `CronJob`)에서 1회 실행

## 환경변수 (config.go)

| Key | 필수 | 예시 |
| --- | --- | --- |
| `WEATHER_SERVICE_KEY` | ✅ | 기상청 단기예보 API 키 |
| `WEATHER_NX`, `WEATHER_NY` | ✅ | 격자 좌표(예: 60, 124) |
| `NEWS_FEEDS` | ✅ | 콤마 구분 RSS URL 목록 |
| `NEWS_TOP_N` | ✅ | 라운드로빈 후 노출 개수(예: 3) |
| `TELEGRAM_BOT_TOKEN` | ✅ | BotFather에서 발급 |
| `TELEGRAM_CHAT_ID` | ✅ | 수신자 chat id |

