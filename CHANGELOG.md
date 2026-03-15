# CHANGELOG

모든 Git 커밋 이력을 최신순으로 기록합니다. 새 커밋은 표 최상단에 추가합니다.

| 일시 | 유형 | 범위 | 변경내용 (목적 포함) |
|---|---|---|---|
| 2026-03-15 | docs | readme | 한국어 README.md 작성 + 쓰레드 초안(not-yet_obsidian-md-handler.md) 추가 |
| 2026-03-15 | feat | install | install.ps1 추가 — 레지스트리 자동 등록, Obsidian 아이콘·앱이름 유지, 기존 command 백업, Explorer 더블클릭 실검증 완료 |
| 2026-03-15 | feat | handler | 최초 구현 — Go 단일 exe, VBS 불필요, 콘솔 창 없음(-H=windowsgui), obsidian.json 볼트 자동 감지, ShellExecuteW URI 열기, EnumWindows 창 활성화(3회 재시도), 한글 경로 URL 인코딩 정상 동작 확인 |
