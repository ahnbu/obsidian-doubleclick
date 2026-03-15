# obsidian-md-handler

**.md 파일 더블클릭 → 옵시디언에서 바로 열기** (Windows)

Node.js 불필요 · exe 3.3MB · 설치 스크립트 한 번 실행으로 완성

---

## 이런 문제 없었어?

탐색기에서 `.md` 파일 더블클릭 → 옵시디언이 열리긴 하는데 **클릭한 파일은 안 열리고 마지막 워크스페이스만 나옴.**

옵시디언은 Electron 앱이라 `Obsidian.exe 파일경로` 방식으로는 특정 파일을 열 수 없어. 이걸 해결하려면 `obsidian://` URI 프로토콜을 경유하는 래퍼가 필요한데, 기존 방법들은 Node.js 설치 + VBS 파일 여러 개 + 스크립트 구성이 복잡했음.

이 도구는 **Go로 만든 단일 exe 하나**로 이 모든 걸 대체함.

---

## 동작 방식

```
.md 더블클릭
  └─ obsidian-handler.exe "%1"
       ├─ vault 내부 파일 → obsidian://advanced-uri?...&openmode=true
       │    ├─ 이미 열린 파일 → 해당 탭으로 포커스 (중복 탭 방지)
       │    └─ 새 파일 → 새 탭으로 열기
       └─ vault 외부 파일 → Typora → VS Code → 메모장 순 자동 감지
```

- vault 목록을 `%APPDATA%\Obsidian\obsidian.json`에서 자동 읽음 (하드코딩 불필요)
- 콘솔 창 깜빡임 없음 (`-H=windowsgui` 빌드)
- 실행 로그: `%TEMP%\obsidian-md-handler.log`

---

## 설치

### 필요한 것

- Windows 10 / 11
- 옵시디언
- [Advanced URI 플러그인](https://obsidian.md/plugins?id=obsidian-advanced-uri) (이미 열린 탭 포커스 기능에 필요)

### 설치 순서

> **순서 중요**: Windows 기본 앱 설정을 먼저 하고 설치 스크립트를 실행해야 함.
> 반대로 하면 Windows가 레지스트리 값을 덮어씀.

**① .md 기본 앱을 Obsidian으로 설정**

Windows 설정 → 앱 → 기본 앱 → `.md` 검색 → **Obsidian** 선택

**② Releases에서 파일 다운로드**

[GitHub Releases](../../releases/latest) 에서 `obsidian-handler.exe`와 `install.ps1`을 **같은 폴더**에 받기

**③ 설치 스크립트 실행**

```powershell
powershell -ExecutionPolicy Bypass -File install.ps1
```

이렇게 뜨면 완료:

```
✅ obsidian-md-handler (Go) 설치 완료
  Command : "C:\...\obsidian-handler.exe" "%1"
✅ .md 기본 앱: Applications\Obsidian.exe — 준비 완료
```

**④ 확인**

탐색기에서 vault 안의 `.md` 파일 더블클릭 → 옵시디언 새 탭으로 열리면 완료.

---

## 설정 (선택)

`obsidian-handler.exe`와 같은 폴더에 `obsidian-handler.config.json` 생성:

```json
{
  "fallbackCommand": "C:\\Program Files\\Typora\\Typora.exe"
}
```

| 항목 | 설명 |
|------|------|
| `fallbackCommand` | vault 외부 파일을 열 앱 경로. 미설정 시 Typora → VS Code → 메모장 순으로 자동 감지 |

---

## 문제 해결

**아무 반응이 없어요** → 로그 파일 확인: `%TEMP%\obsidian-md-handler.log`

**"advanced-uri" 오류가 떠요** → 옵시디언 설정 → 커뮤니티 플러그인 → Advanced URI 설치·활성화 확인

**파일 아이콘이 이상해졌어요** → `install.ps1` 다시 실행하면 Obsidian 아이콘으로 복원됨

**vault 외부 파일이 안 열려요** → `obsidian-handler.config.json`에 `fallbackCommand` 직접 지정

---

## 직접 빌드

```bash
git clone https://github.com/your-username/obsidian-md-handler
cd obsidian-md-handler

# 배포용 (콘솔 창 없음)
go build -ldflags "-H=windowsgui" -o obsidian-handler.exe .

# 디버그용 (콘솔 창 있음, --debug 플래그 사용 가능)
go build -o obsidian-handler-debug.exe .
```

Go 1.20+ 필요. 외부 의존성 없음.

```bash
# 볼트 감지 확인
obsidian-handler-debug.exe --debug "C:\path\to\file.md"
```

---

## 제거

`install.ps1`이 `.backup/` 폴더에 이전 설정을 백업해둬. 복구하려면:

```powershell
$backup = Get-ChildItem ".backup\backup_*.json" | Sort-Object Name | Select-Object -Last 1 | Get-Content | ConvertFrom-Json
Set-ItemProperty "HKCU:\Software\Classes\Applications\Obsidian.exe\shell\open\command" -Name "(default)" -Value $backup.previousCommand
```

또는 Windows 설정에서 `.md` 기본 앱을 다른 앱으로 바꾸면 됨.

---

## 라이선스

MIT
