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

## 로컬 실행

```bash
go build -o bin/brief ./cmd/brief
WEATHER_SERVICE_KEY=... WEATHER_NX=60 WEATHER_NY=124 \
NEWS_FEEDS=https://www.hankyung.com/feed/economy,https://www.hankyung.com/feed/it NEWS_TOP_N=3 \
TELEGRAM_BOT_TOKEN=... TELEGRAM_CHAT_ID=... \
./bin/brief
```

## 운영

- **회사 PC (macOS launchd)**: `~/Library/LaunchAgents/com.hjchoi.morningbrief.plist`, 매일 07:00 KST.
- **집 k3s**: K8s 매니페스트는 `~/Desktop/Projects/homelab-k8s/gitops/morning-brief/` (Namespace + ConfigMap + Secret(example) + CronJob + Kustomization). 빌드·배포 절차는 그쪽 `README.md` 참조.

> 동시에 두 환경을 켜두면 메시지가 두 번 옴 — 하나만 활성화할 것.

## 테스트

```bash
go test ./... -race -count=1
```

## 코드 구조

| 패키지 | 역할 |
| --- | --- |
| `cmd/brief` | 엔트리포인트, 의존성 와이어링 |
| `internal/brief` | 전체 시나리오 조립 (날씨·뉴스 동시 fetch + 부분 실패 허용 + send) |
| `internal/config` | env 로드·검증 |
| `internal/weather` | 기상청 단기예보 호출·일일 강수 종합 |
| `internal/news` | 라운드로빈 + 보일러플레이트 필터 |
| `internal/message` | 텔레그램 본문 포맷 |
| `internal/telegram` | sendMessage 호출 (토큰 마스킹) |
