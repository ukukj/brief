#!/bin/zsh
# morning-brief 자동 실행 래퍼 (launchd가 호출).
# .env(시크릿, gitignore됨)를 export 한 뒤 빌드된 바이너리를 실행한다.
cd /Users/hjchoi/Desktop/Projects/morning-brief || exit 1
set -a
. ./.env
set +a
echo "----- $(date '+%Y-%m-%d %H:%M:%S %Z') run -----"
exec /Users/hjchoi/Desktop/Projects/morning-brief/bin/brief
