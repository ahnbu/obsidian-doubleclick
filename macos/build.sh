#!/bin/bash
# macOS 빌드: 테스트 → 유니버설 바이너리 → .app 조립 → ad-hoc 서명 → zip
#   ./macos/build.sh
# 결과: macos/build/ObsidianDoubleclick.app, macos/build/obsidian-doubleclick-macos.zip
# 필요: Xcode Command Line Tools (swiftc, lipo, codesign, iconutil, sips, ditto)

set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/.." && pwd)"
OUT="$HERE/build"
APP="$OUT/ObsidianDoubleclick.app"
ZIP="$OUT/obsidian-doubleclick-macos.zip"
EXE="obsidian-doubleclick"
WORK="$OUT/.work"

mkdir -p "$WORK"

# 1. 테스트 — 하나라도 실패하면 중단
echo "== test"
swiftc -parse-as-library "$HERE/Core.swift" "$HERE/tests/CoreTests.swift" -o "$WORK/core-tests"
"$WORK/core-tests"

# 2. 유니버설 바이너리. x86_64 빌드가 안 되면 arm64 전용으로 진행한다.
echo "== build"
SRC=("$HERE/Core.swift" "$HERE/App.swift")
swiftc -O -parse-as-library -target arm64-apple-macos12 "${SRC[@]}" -o "$WORK/$EXE-arm64"
ARCHS="arm64"
if swiftc -O -parse-as-library -target x86_64-apple-macos12 "${SRC[@]}" -o "$WORK/$EXE-x86_64" 2>"$WORK/x86_64.err"; then
    lipo -create "$WORK/$EXE-arm64" "$WORK/$EXE-x86_64" -output "$WORK/$EXE"
    ARCHS="arm64 x86_64"
else
    echo "WARN x86_64 빌드 실패 — arm64 전용으로 진행 ($(head -1 "$WORK/x86_64.err"))"
    cp "$WORK/$EXE-arm64" "$WORK/$EXE"
fi

# 3. .app 조립
echo "== bundle"
# 기존 번들은 지우지 않고 제자리에 덮어쓴다(번들 구성 파일은 아래 셋과 서명뿐이다).
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$WORK/$EXE" "$APP/Contents/MacOS/$EXE"
cp "$HERE/Info.plist" "$APP/Contents/Info.plist"

# 아이콘: Windows 버전과 같은 obsidian.ico → AppIcon.icns. 실패하면 아이콘 없이 진행한다.
ICONSET="$WORK/AppIcon.iconset"
mkdir -p "$ICONSET"
if sips -s format png "$ROOT/obsidian.ico" --out "$WORK/icon.png" >/dev/null 2>&1; then
    for s in 16 32 128 256 512; do
        sips -z $s $s "$WORK/icon.png" --out "$ICONSET/icon_${s}x${s}.png" >/dev/null
        d=$((s * 2))
        sips -z $d $d "$WORK/icon.png" --out "$ICONSET/icon_${s}x${s}@2x.png" >/dev/null
    done
    if iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/AppIcon.icns"; then
        echo "icon: AppIcon.icns"
    else
        echo "WARN iconutil 실패 — 아이콘 없이 진행"
    fi
else
    echo "WARN obsidian.ico 변환 실패 — 아이콘 없이 진행"
fi

# 4. ad-hoc 서명 (공증 없음)
echo "== sign"
codesign --force --deep -s - "$APP"
codesign --verify --verbose=1 "$APP"

# 5. zip
echo "== zip"
# 확장 속성은 뺀다 — 넣으면 ._ 파일이 섞여, Apple 외 압축 도구로 풀었을 때 서명이 깨진다.
ditto -c -k --keepParent --norsrc --noextattr --noacl "$APP" "$ZIP"

echo "== done"
echo "archs: $ARCHS"
echo "app:   $APP ($(du -sh "$APP" | cut -f1))"
echo "zip:   $ZIP ($(du -h "$ZIP" | cut -f1))"
