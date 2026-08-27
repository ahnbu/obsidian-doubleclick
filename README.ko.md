# obsidian-md-handler

### `.md` 더블클릭해도 옵시디언에서 그 파일이 안 열리는 문제, 이걸로 해결됨

**Windows 전용** · exe 하나 3.4MiB · .NET·Node.js·플러그인 **전부 불필요**

[![Release](https://img.shields.io/github/v/release/ahnbu/obsidian-md-handler?color=7c3aed)](https://github.com/ahnbu/obsidian-md-handler/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/ahnbu/obsidian-md-handler/total?color=7c3aed)](https://github.com/ahnbu/obsidian-md-handler/releases)
[![License](https://img.shields.io/github/license/ahnbu/obsidian-md-handler?color=7c3aed)](LICENSE)
![Platform](https://img.shields.io/badge/platform-Windows%2010%20%7C%2011-0078d4)

[English README](README.md)

<!-- DEMO GIF: 여기에 `![demo](docs/demo.gif)` 한 줄. 영어판과 같은 파일을 쓴다. -->

---

## 이 문제 겪어봤다면

`.md` 기본 앱을 옵시디언으로 잡아두고 노트를 더블클릭했는데, **클릭한 파일은 안 열리고 마지막 워크스페이스만 뜬 적** 있을 거다.

설정이 잘못된 게 아니다. 옵시디언은 Electron 앱이라 윈도우가 넘겨주는 파일 경로(`Obsidian.exe "%1"`)를 무시한다.

이건 [2020년 5월부터 열려 있는 최다 요청 이슈](https://forum.obsidian.md/t/have-obsidian-be-the-handler-of-md-files-add-ability-to-use-obsidian-as-a-markdown-editor-on-files-outside-vault-file-association/314)다 — 좋아요 100개 이상, 댓글 170개 이상, 지금도 미해결.

그래서 해결은 옵시디언 바깥에 있어야 한다. 탐색기와 옵시디언 사이에 끼어서 파일 경로를 `obsidian://` URI로 번역해주는 작은 프로그램 — 이게 그거다.

## 기존 우회안과 뭐가 다른가

| | 이 도구 | [ObsidianShell](https://github.com/Chaoses-Ib/ObsidianShell) | BAT / AHK 스크립트 |
|---|---|---|---|
| 의존성 | 없음 (exe 1개) | .NET 런타임 | 제각각 |
| 옵시디언 플러그인 필요 | 불필요 (선택) | 불필요 | 대개 필요 |
| 이미 열린 탭 포커스 (중복 방지) | ✅ (플러그인 있을 때) | ❌ | ❌ |
| 볼트 여러 개일 때 올바른 창 선택 | ✅ | ❌ | ❌ |
| 볼트 외부 파일 | 다른 에디터로 폴백 | 설정 가능 | 대개 미처리 |
| 유지보수 | 진행 중 | 2023년 이후 없음 | — |

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

> **exe에 코드 서명이 없음.** 처음 실행하면 Windows SmartScreen 경고가 뜬다. 남의 바이너리를 그냥 받기 꺼려지면 [직접 빌드](#직접-빌드)하면 된다 — 외부 의존성 없는 Go 900줄이고 명령 한 줄이면 끝난다.

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

**Lazy Plugin Loader 쓰는데 Advanced URI가 안 먹어요** → [Lazy Plugin Loader](https://github.com/alangrainger/obsidian-lazy-plugins)는 플러그인 로딩을 지연시키는데, 이 핸들러는 `community-plugins.json`만 읽는다. 그 파일은 **"켜져 있음"은 알려주지만 "이미 로딩됐음"은 알려주지 않는다.** URI가 도착한 시점에 Advanced URI가 아직 안 떴으면 요청이 그냥 씹힌다. **Lazy Loader 설정에서 Advanced URI를 `instant`로 두면 된다** — 나머지 전부의 입구라서 지연 대상이 아니다. 아니면 config에 `"uriMode": "official"`을 넣어 플러그인을 우회해도 된다.

**옵시디언이 꺼져 있을 때 몇 초씩 걸려요** → 핸들러가 아니라 옵시디언 콜드 스타트다. 실측해보면 **창 자체는 약 1초 만에 뜬다.** 다만 빈 창으로 떠서 내용이 채워지는 데 한참 걸린다 — 볼트 인덱싱 + 플러그인 로딩. 즉 체감하는 대기는 창이 만들어지는 시간이 아니라 **채워지는 시간**이다.

이건 이 핸들러가 어떻게 해도 못 줄인다. [Lazy Plugin Loader](https://github.com/alangrainger/obsidian-lazy-plugins)는 그중 플러그인 로딩 쪽만 도와주고, 볼트가 크면 인덱싱이 지배적이다. 핸들러는 콜드 스타트일 때 창을 더 오래(30초, 이미 떠 있으면 3초) 기다려 먼저 포기하지 않도록만 한다. 로그의 `mode=`·`elapsed=`로 어느 경로였는지 확인할 수 있다

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
