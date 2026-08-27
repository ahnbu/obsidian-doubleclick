# CHANGELOG

모든 Git 커밋 이력을 최신순으로 기록합니다. 새 커밋은 표 최상단에 추가합니다.

| 일시 | 유형 | 범위 | 변경내용 (목적 포함) |
|---|---|---|---|
| 2026-08-27 15:31 | fix | handler | 콜드 스타트 적응형 창 대기 — 옵시디언이 꺼져 있으면 대기 예산을 3초에서 30초로 늘리고, 예산 소진 전까지는 exact 매칭만 받아 다른 볼트 창을 잘못 잡지 않도록 함. 실측 근거: 실행 2350건 중 실패 23건(1.0%), 그 65%가 직전 실행 1시간 초과(중앙값 357분). README 배지·상단 재구성, Lazy Plugin Loader 상호작용과 콜드 스타트 문제해결 항목 추가, 한국어판 톤 동기화 |
| 2026-08-27 15:03 | docs | verify | 릴리스 직전 추가 검증 2건 기록 — 탐색기 연결 경로(ShellExecute) 실행 확인, 플러그인 없는 실제 볼트에서 공식 URI 자동 전환 E2E 통과. .agents 볼트의 community-plugins.json만 임시 이동하는 격리 방식으로 글로벌 obsidian.json 미접촉, 해시 대조로 원복 확인. 계획서의 8번 생략 기록을 실행 완료로 갱신 |
| 2026-08-27 14:55 | docs | readme | 공개 배포 준비 — 영어 README를 정본으로 신설(한국어는 README.ko.md로 분리), MIT LICENSE 파일 추가, your-username 플레이스홀더를 ahnbu로 수정, uriMode 설정과 다중 vault 창 선택 동작 문서화, 미서명 exe 안내 추가 |
| 2026-08-27 13:57 | fix | handler | 다중 vault 포커스 버그 수정 + Advanced URI 플러그인 의존 제거 — 창 제목의 vault명으로 대상 창을 특정(2단계 폴백), community-plugins.json 감지로 플러그인 없으면 공식 URI(paneType=tab) 자동 전환, uriMode config 옵션 추가, 판정용 로깅 강화. 공개 배포 선행 조건 |
| 2026-07-08 11:27 | chore | gitignore | gitignore에 .codegraph/를 추가해 CodeGraph 로컬 캐시가 작업트리에 노출되지 않도록 보완 |
| 2026-05-09 | feat | self-heal | 더블클릭 실행 시 handler command, DefaultIcon, FriendlyAppName drift를 안전하게 자동 복구하고 `--doctor`/`--repair` 진단 명령 추가 |
| 2026-03-29 | fix | handler | Typora fallback 공백 파일명 인자 quoting 수정 — syscall.EscapeArg 적용 및 회귀 테스트 추가 |
| 2026-03-25 | chore | .gitignore | _handoff/ 항목 제거 — handoff git 추적 복원 |
| 2026-03-20 | feat | icon | Obsidian 아이콘 EXE 임베딩 — 연결 프로그램 메뉴에서 Go 기본 아이콘 대신 Obsidian 로고 표시 |
| 2026-03-19 | chore | gitignore | .backup/ 디렉토리 gitignore 추가 — 백업 파일 추적 방지 |
| 2026-03-18 | fix | handler | advanced-uri → adv-uri 프로토콜 전환 — 이중 디코딩 버그 수정판 사용, % 포함 파일명 분기 로직 제거, 모든 파일 단일 경로로 통합 |
| 2026-03-17 | fix | handler | url.QueryEscape 공백→"+" 버그 수정 — safeEscape() 헬퍼로 "%20" 인코딩하여 공백 파일명 더블클릭 시 0바이트 복제 파일 생성 방지 |
| 2026-03-15 | docs | readme | 한국어 README.md 작성 + 쓰레드 초안(not-yet_obsidian-md-handler.md) 추가 |
| 2026-03-15 | feat | install | install.ps1 추가 — 레지스트리 자동 등록, Obsidian 아이콘·앱이름 유지, 기존 command 백업, Explorer 더블클릭 실검증 완료 |
| 2026-03-15 | feat | handler | 최초 구현 — Go 단일 exe, VBS 불필요, 콘솔 창 없음(-H=windowsgui), obsidian.json 볼트 자동 감지, ShellExecuteW URI 열기, EnumWindows 창 활성화(3회 재시도), 한글 경로 URL 인코딩 정상 동작 확인 |
