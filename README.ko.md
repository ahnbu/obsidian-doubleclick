# obsidian-md-handler

**.md 파일 더블클릭 → 옵시디언에서 바로 열기** (Windows)

Node.js 불필요 · 플러그인 불필요 · exe 3.4MiB · 설치 스크립트 한 번 실행으로 완성

[English README](README.md)

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
       ├─ vault 내부 파일 → 옵시디언에서 열기
       │    ├─ Advanced URI 플러그인 켜짐 → 이미 열린 노트면 그 탭으로 포커스,
       │    │                               아니면 새 탭 (중복 탭 방지)
       │    └─ 플러그인 없음            → 공식 URI로 새 탭에 열기
       └─ vault 외부 파일 → Typora → VS Code → 메모장 순 자동 감지
```

- vault 목록을 `%APPDATA%\Obsidian\obsidian.json`에서 자동 읽음 (하드코딩 불필요)
- **vault를 여러 개 켜두면 해당 vault의 창을 골라서** 앞으로 가져옴
- Obsidian 업데이트 후 `.md` 연결 command, 아이콘, 앱 이름이 틀어지면 더블클릭 시 안전 항목만 자동 복구
- 콘솔 창 깜빡임 없음 (`-H=windowsgui` 빌드)
- 실행 로그: `%TEMP%\obsidian-md-handler.log`

---

## 설치

### 필요한 것

- Windows 10 / 11
- 옵시디언
- *(선택)* [Advanced URI 플러그인](https://obsidian.md/plugins?id=obsidian-advanced-uri) — **없어도 동작함.** 이미 열려 있는 노트를 새로 여는 대신 그 탭으로 포커스하고 싶을 때만 필요

> **exe에 코드 서명이 없음.** 처음 실행하면 Windows SmartScreen 경고가 뜬다. 남의 바이너리를 그냥 받기 꺼려지면 [직접 빌드](#직접-빌드)하면 된다 — 외부 의존성 없는 Go 840줄이고 명령 한 줄이면 끝난다.

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
  "uriMode": "auto",
  "fallbackCommand": "C:\\Program Files\\Typora\\Typora.exe",
  "obsidianExePath": "C:\\Program Files\\Obsidian\\Obsidian.exe"
}
```

| 항목 | 기본값 | 설명 |
|------|------|------|
| `uriMode` | `auto` | `auto`는 해당 vault에서 Advanced URI가 켜져 있는지 보고 알아서 고른다. `adv-uri` 또는 `official`로 강제 지정 가능 |
| `fallbackCommand` | 자동 감지 | vault 외부 파일을 열 앱 경로. 미설정 시 Typora → VS Code → 메모장 순 |
| `obsidianExePath` | 자동 감지 | 옵시디언이 흔치 않은 경로에 설치된 경우에만 필요 |

### uriMode가 하는 일

`auto`는 `<vault>/.obsidian/community-plugins.json`을 읽어 Advanced URI가 **활성화**돼 있는지(설치만 된 게 아니라) 확인하고 방식을 고른다.

| | `adv-uri` (플러그인 켜짐) | `official` (플러그인 없음) |
|---|---|---|
| 파일 열기 | ✅ | ✅ |
| 새 탭으로 열기 | ✅ | ✅ |
| 이미 열린 노트면 그 탭으로 포커스 | ✅ | ❌ 탭이 하나 더 생김 |

파일을 못 읽으면 항상 동작하는 공식 URI로 폴백한다.

---

## 문제 해결

**아무 반응이 없어요** → 로그 파일 확인: `%TEMP%\obsidian-md-handler.log`

**"advanced-uri" 오류가 떠요** → config에 `uriMode`를 `adv-uri`로 강제해뒀는데 플러그인이 꺼져 있는 경우다. `auto`로 바꾸거나 지우면 자동으로 공식 URI를 쓴다

**파일 아이콘이 이상해졌어요** → vault 안의 `.md`를 한 번 더블클릭하거나 `.\obsidian-handler.exe --repair` 실행. 그래도 안 되면 `install.ps1` 재실행

**vault 외부 파일이 안 열려요** → `obsidian-handler.config.json`에 `fallbackCommand` 직접 지정

**연결 상태를 직접 점검하고 싶어요**:

```powershell
.\obsidian-handler.exe --doctor
.\obsidian-handler.exe --repair
```

`--repair`는 `.md` 기본 앱 자체를 강제로 바꾸지 않고, handler command, Obsidian 아이콘, 앱 이름만 복구함.

---

## 직접 빌드

```bash
git clone https://github.com/ahnbu/obsidian-md-handler
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

# 연결 상태 확인
obsidian-handler-debug.exe --doctor
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

MIT — [LICENSE](LICENSE) 참조
