# CHANGELOG

모든 Git 커밋 이력을 최신순으로 기록합니다. 새 커밋은 표 최상단에 추가합니다.

| 일시 | 유형 | 범위 | 변경내용 (목적 포함) |
|---|---|---|---|
| 2026-03-29 | fix | handler | Typora fallback 공백 파일명 인자 quoting 수정 — syscall.EscapeArg 적용 및 회귀 테스트 추가 |
| 2026-03-25 | chore | .gitignore | _handoff/ 항목 제거 — handoff git 추적 복원 |
| 2026-03-20 | feat | icon | Obsidian 아이콘 EXE 임베딩 — 연결 프로그램 메뉴에서 Go 기본 아이콘 대신 Obsidian 로고 표시 |
| 2026-03-19 | chore | gitignore | .backup/ 디렉토리 gitignore 추가 — 백업 파일 추적 방지 |
| 2026-03-18 | fix | handler | advanced-uri → adv-uri 프로토콜 전환 — 이중 디코딩 버그 수정판 사용, % 포함 파일명 분기 로직 제거, 모든 파일 단일 경로로 통합 |
| 2026-03-17 | fix | handler | url.QueryEscape 공백→"+" 버그 수정 — safeEscape() 헬퍼로 "%20" 인코딩하여 공백 파일명 더블클릭 시 0바이트 복제 파일 생성 방지 |
| 2026-03-15 | docs | readme | 한국어 README.md 작성 + 쓰레드 초안(not-yet_obsidian-md-handler.md) 추가 |
| 2026-03-15 | feat | install | install.ps1 추가 — 레지스트리 자동 등록, Obsidian 아이콘·앱이름 유지, 기존 command 백업, Explorer 더블클릭 실검증 완료 |
| 2026-03-15 | feat | handler | 최초 구현 — Go 단일 exe, VBS 불필요, 콘솔 창 없음(-H=windowsgui), obsidian.json 볼트 자동 감지, ShellExecuteW URI 열기, EnumWindows 창 활성화(3회 재시도), 한글 경로 URL 인코딩 정상 동작 확인 |
