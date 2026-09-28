// obsidian-doubleclick (macOS): 순수 로직. AppKit에 의존하지 않는다 — tests/CoreTests.swift가 이 파일만 붙여 컴파일한다.
// Windows main.go의 buildURI·findVault·resolveURIMode·detectAdvancedURI와 1:1 대응한다.

import Foundation

let appVersion = "0.2.0"
let selfBundleID = "io.github.ahnbu.obsidian-doubleclick"
let obsidianBundleID = "md.obsidian"
let markdownUTI = "net.daringfireball.markdown"

enum URIMode: String {
    case advanced = "adv-uri"
    case official = "official"
}

let advancedURIID = "obsidian-advanced-uri"

// 볼트 밖 파일을 열 편집기. 순서가 우선순위다(Windows: Typora → VS Code → 메모장).
let defaultFallbackBundleIDs = [
    "abnerworks.Typora",
    "com.microsoft.VSCode",
    "com.apple.TextEdit",
]

// ~/Library/Application Support/obsidian-doubleclick/config.json (선택)
struct HandlerConfig: Codable, Equatable {
    // "auto"(기본) | "adv-uri"(강제) | "official"(강제)
    var uriMode: String?
    // 번들 ID(com.microsoft.VSCode) 또는 앱 경로(/Applications/Typora.app)
    var fallbackApp: String?
}

struct VaultMatch: Equatable {
    let vaultPath: String   // obsidian.json에 적힌 원래 경로
    let vaultName: String   // NFC
    let relPath: String     // 볼트 루트 기준, "/" 구분, NFC
}

// ---------------------------------------------------------------------------
// URI 빌드
// ---------------------------------------------------------------------------

// safeEscape: Go의 url.QueryEscape + "+"→"%20"과 같은 결과를 낸다.
// ASCII unreserved(A-Za-z0-9-_.~)만 남기고 UTF-8 바이트 단위로 전부 퍼센트 인코딩한다.
// CharacterSet.alphanumerics는 한글을 문자로 취급해 인코딩하지 않으므로 쓰지 않는다.
func safeEscape(_ s: String) -> String {
    var out = ""
    for b in s.utf8 {
        switch b {
        case UInt8(ascii: "A")...UInt8(ascii: "Z"),
             UInt8(ascii: "a")...UInt8(ascii: "z"),
             UInt8(ascii: "0")...UInt8(ascii: "9"),
             UInt8(ascii: "-"), UInt8(ascii: "_"), UInt8(ascii: "."), UInt8(ascii: "~"):
            out.unicodeScalars.append(Unicode.Scalar(b))
        default:
            out += String(format: "%%%02X", b)
        }
    }
    return out
}

func buildURI(vaultName: String, relPath: String, mode: URIMode) -> String {
    let vault = safeEscape(vaultName.precomposedStringWithCanonicalMapping)
    let rel = safeEscape(relPath.precomposedStringWithCanonicalMapping)
    switch mode {
    case .official:
        return "obsidian://open?vault=" + vault + "&file=" + rel + "&paneType=tab"
    case .advanced:
        return "obsidian://adv-uri?vault=" + vault + "&filepath=" + rel + "&openmode=true"
    }
}

// ---------------------------------------------------------------------------
// 볼트 탐색
// ---------------------------------------------------------------------------

// canonicalPath: 심볼릭 링크 해석(/tmp → /private/tmp 포함) 후 NFC.
// URL.resolvingSymlinksInPath는 /private 접두를 떼어내므로 realpath(3)를 쓴다.
// 존재하지 않는 경로는 realpath가 실패하므로 표준화만 한다.
func canonicalPath(_ path: String) -> String {
    let standardized = (path as NSString).standardizingPath
    var resolved = standardized
    if let p = realpath(standardized, nil) {
        resolved = String(cString: p)
        free(p)
    }
    return resolved.precomposedStringWithCanonicalMapping
}

func pathComponents(_ path: String) -> [String] {
    return path.split(separator: "/", omittingEmptySubsequences: true).map(String.init)
}

// findVault: 가장 긴 접두 경로의 볼트를 고른다. 비교는 경로 구성요소 단위로
// NFC + 소문자화해서 한다 — 구성요소 단위라 /a/vault2가 /a/vault에 걸리지 않는다.
// canonicalize는 테스트에서 파일시스템 없이 돌리기 위한 주입점이다.
func findVault(filePath: String, vaults: [String],
               canonicalize: (String) -> String = canonicalPath) -> VaultMatch? {
    let fileComps = pathComponents(canonicalize(filePath))
    let fileLower = fileComps.map { $0.lowercased() }
    var best: VaultMatch?
    var bestLen = -1

    for v in vaults where !v.isEmpty {
        let vComps = pathComponents(canonicalize(v))
        guard vComps.count <= fileComps.count else { continue }
        let vLower = vComps.map { $0.lowercased() }
        guard Array(fileLower.prefix(vComps.count)) == vLower else { continue }
        if vComps.count > bestLen {
            bestLen = vComps.count
            let name = (vComps.last ?? v).precomposedStringWithCanonicalMapping
            let rel = fileComps.dropFirst(vComps.count).joined(separator: "/")
            best = VaultMatch(vaultPath: v, vaultName: name, relPath: rel.precomposedStringWithCanonicalMapping)
        }
    }
    return best
}

// ---------------------------------------------------------------------------
// URI 모드
// ---------------------------------------------------------------------------

// detectAdvancedURI: community-plugins.json은 "활성화된" 목록이다.
// 없거나 깨졌으면 false — 공식 URI로 가는 쪽이 안전하다.
func detectAdvancedURI(vaultPath: String) -> Bool {
    let p = (vaultPath as NSString).appendingPathComponent(".obsidian/community-plugins.json")
    guard let data = FileManager.default.contents(atPath: p),
          let ids = try? JSONDecoder().decode([String].self, from: data) else {
        return false
    }
    return ids.contains(advancedURIID)
}

func resolveURIMode(config: HandlerConfig, vaultPath: String) -> URIMode {
    if let m = config.uriMode, let forced = URIMode(rawValue: m) {
        return forced
    }
    return detectAdvancedURI(vaultPath: vaultPath) ? .advanced : .official
}

// ---------------------------------------------------------------------------
// 설정 로드
// ---------------------------------------------------------------------------

private struct ObsidianConfig: Decodable {
    struct Entry: Decodable { let path: String? }
    let vaults: [String: Entry]?
}

func parseVaults(_ data: Data) -> [String]? {
    guard let cfg = try? JSONDecoder().decode(ObsidianConfig.self, from: data) else { return nil }
    return (cfg.vaults ?? [:]).values.compactMap { $0.path }.filter { !$0.isEmpty }.sorted()
}

func parseHandlerConfig(_ data: Data?) -> HandlerConfig {
    guard let data = data, let cfg = try? JSONDecoder().decode(HandlerConfig.self, from: data) else {
        return HandlerConfig()
    }
    return cfg
}

// ---------------------------------------------------------------------------
// 폴백 앱 선택
// ---------------------------------------------------------------------------

enum FallbackChoice: Equatable {
    case configured(String)   // 설정 fallbackApp
    case detected(String)     // 자동 탐색으로 찾은 번들 ID
    case none
}

// pickFallback: 설정값이 있으면 그것을, 없으면 후보 순서대로 설치된 첫 앱을 고른다.
// 자기 자신과 옵시디언은 제외한다 — .md를 다시 자기에게 넘기면 무한 루프가 된다.
// isInstalled는 번들 ID 또는 앱 경로를 받아 설치 여부를 답한다(테스트 주입점).
func pickFallback(configured: String?,
                  candidates: [String] = defaultFallbackBundleIDs,
                  excluded: Set<String> = [selfBundleID, obsidianBundleID],
                  isInstalled: (String) -> Bool) -> FallbackChoice {
    let excludedLower = Set(excluded.map { $0.lowercased() })
    func isExcluded(_ app: String) -> Bool {
        let key = app.lowercased()
        if excludedLower.contains(key) { return true }
        let base = ((app as NSString).lastPathComponent as NSString).deletingPathExtension.lowercased()
        return base == "obsidian" || base == "obsidiandoubleclick"
    }
    if let c = configured, !c.isEmpty, !isExcluded(c), isInstalled(c) {
        return .configured(c)
    }
    for id in candidates where !isExcluded(id) && isInstalled(id) {
        return .detected(id)
    }
    return .none
}

// ---------------------------------------------------------------------------
// 디버그 출력
// ---------------------------------------------------------------------------

struct DebugInfo: Encodable {
    let filePath: String
    let vault: String?
    let relPath: String?
    let config: HandlerConfig
    let allVaults: [String]
    let uriMode: String?
    let uri: String?
}

func makeDebugInfo(filePath: String, vaults: [String], config: HandlerConfig) -> DebugInfo {
    let match = findVault(filePath: filePath, vaults: vaults)
    var mode: URIMode?
    var uri: String?
    if let m = match {
        mode = resolveURIMode(config: config, vaultPath: m.vaultPath)
        uri = buildURI(vaultName: m.vaultName, relPath: m.relPath, mode: mode!)
    }
    return DebugInfo(filePath: filePath, vault: match?.vaultPath, relPath: match?.relPath,
                     config: config, allVaults: vaults, uriMode: mode?.rawValue, uri: uri)
}
