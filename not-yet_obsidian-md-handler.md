# [Kick sentence] - thread 1
<!-- 목적: 옵시디언 사용자의 공감 유발 → 해결책 제시 → 저장 유도 -->
<!-- 내용 -->
옵시디언 쓰는데 .md 파일 더블클릭하면 파일이 안 열리는 거 나만 불편했나

Obsidian은 Electron 앱이라 파일 경로를 그냥 넘기면 무시하거든. 그래서 매번 옵시디언 열고 파일 직접 찾아야 했어.

근데 이걸 해결하려고 Go로 핸들러 만들었는데

<!-- spoiler feature 적용-->
- exe 하나 (3.3MB)
- Node.js 설치 불필요
- VBS 파일 같은 거 없음
- 설치 스크립트 한 번 실행으로 끝
<!-- spoiler feature 적용 끝 -->

더블클릭하면 vault 안 파일은 옵시디언 새 탭으로 바로 열리고, 이미 열린 파일이면 그 탭으로 포커스됨 (중복 탭 방지까지)
<!-- 내용 끝-->

# [Kick Image]
없음

# explanation

## Kick sentence에 대한 근거, 설명

옵시디언의 .md 파일 더블클릭 문제는 Electron 앱 구조에서 비롯됨.
기존 해결책은 Node.js + VBS + 스크립트 조합이 필요했음.
Go로 단일 exe를 만들면 콘솔 창 없이(-H=windowsgui) 직접 Win32 API 호출 가능.

### thread 2
<!-- 목적: 설치 방법 구체적으로 안내 + 링크 제공 -->
<!-- 내용 -->
설치 방법 간단해

① Windows 설정에서 .md 기본 앱을 Obsidian으로 설정
② obsidian-handler.exe + install.ps1 같은 폴더에 받기
③ 아래 명령어 실행

```
powershell -ExecutionPolicy Bypass -File install.ps1
```

vault 안 파일 → 옵시디언 새 탭
vault 밖 파일 → Typora → VS Code → 메모장 순으로 자동 감지

GitHub에서 받아봐 →
https://github.com/your-username/obsidian-md-handler
<!-- 내용 끝 -->

### thread 3
<!-- 목적: CTA + 팔로우 유도 -->
<!-- 내용 -->
옵시디언 관련 이런 거 만드는 걸 좋아해서 앞으로도 계속 공유할 거야

도움됐으면 스하리 해줘 ㅎㅎ
<!-- 내용 끝 -->

# 쓰레드 포스팅 결과
## 조회수
## 리포스트
## 공유하기
## 좋아요
## 댓글

# 예상 쓰레드 포스팅 결과
## 예상 조회수
800~1,500

## 예상 근거
- 킥 문장 유형: 공감형 (옵시디언 사용자 특정 불편 공략)
- 타겟: 한국 옵시디언 사용자 (니치하지만 충성도 높음)
- spoiler로 스펙 숨겨서 펼쳐보기 유도
- 실제 동작하는 도구라 신뢰도 있음

## 예상 강점
- 구체적인 문제 → 구체적인 해결책 구조
- 수치(3.3MB)로 신뢰감
- 바로 설치해볼 수 있어서 저장 욕구 있음

## 예상 약점
- 타겟이 좁아서 조회수 폭발은 어려움
- GitHub 링크가 아직 미완성 상태
