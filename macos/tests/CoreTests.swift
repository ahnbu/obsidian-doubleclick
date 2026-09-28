// Core.swift 자체 테스트. XCTest 없이 Command Line Tools만으로 돈다.
//   swiftc -parse-as-library ../Core.swift CoreTests.swift -o /tmp/core-tests && /tmp/core-tests
// 실패가 하나라도 있으면 exit 1.

import Foundation

var failures = 0
var passes = 0

func check(_ name: String, _ cond: Bool, _ detail: @autoclosure () -> String = "") {
    if cond {
        passes += 1
        print("PASS \(name)")
    } else {
        failures += 1
        print("FAIL \(name) \(detail())")
    }
}

func eq<T: Equatable>(_ name: String, _ got: T, _ want: T) {
    check(name, got == want, "got=\(got) want=\(want)")
}

// 파일시스템 없이 findVault를 돌리기 위한 canonicalize: 심볼릭 링크 해석만 뺀 것.
func pureCanon(_ p: String) -> String {
    return (p as NSString).standardizingPath.precomposedStringWithCanonicalMapping
}

func tempDir(_ name: String) -> String {
    let d = (NSTemporaryDirectory() as NSString).appendingPathComponent("odc-tests-\(getpid())/\(name)")
    try? FileManager.default.createDirectory(atPath: d + "/.obsidian", withIntermediateDirectories: true)
    return d
}

@main
struct CoreTests {
    static func main() {
        // T1 한글 + 공백. Go: strings.ReplaceAll(url.QueryEscape("노트 1.md"), "+", "%20")
        eq("T1 safeEscape 한글+공백", safeEscape("노트 1.md"), "%EB%85%B8%ED%8A%B8%201.md")
        eq("T1 buildURI adv-uri",
           buildURI(vaultName: "cowork", relPath: "폴더/노트 1.md", mode: .advanced),
           "obsidian://adv-uri?vault=cowork&filepath=%ED%8F%B4%EB%8D%94%2F%EB%85%B8%ED%8A%B8%201.md&openmode=true")
        eq("T1 buildURI official",
           buildURI(vaultName: "my vault", relPath: "a b.md", mode: .official),
           "obsidian://open?vault=my%20vault&file=a%20b.md&paneType=tab")

        // T2 특수문자. Go QueryEscape: % → %25, # → %23, & → %26, + → %2B
        eq("T2 safeEscape %#&+", safeEscape("50% #1 & a+b.md"), "50%25%20%231%20%26%20a%2Bb.md")
        eq("T2 unreserved 유지", safeEscape("A-z_0.~"), "A-z_0.~")

        // T3 NFD 파일 경로 vs NFC 볼트 경로
        let nfcVault = "/Users/u/코워크"
        let nfdFile = "/Users/u/코워크/한글 노트.md".decomposedStringWithCanonicalMapping
        // Swift String의 ==는 정규화 동치로 비교하므로 스칼라 수로 확인한다.
        check("T3 입력이 실제로 NFD",
              nfdFile.unicodeScalars.count > nfdFile.precomposedStringWithCanonicalMapping.unicodeScalars.count)
        check("T3 URI 상대경로가 NFC 바이트",
              buildURI(vaultName: "v", relPath: "한".decomposedStringWithCanonicalMapping, mode: .advanced)
                == "obsidian://adv-uri?vault=v&filepath=%ED%95%9C&openmode=true")
        let m3 = findVault(filePath: nfdFile, vaults: [nfcVault], canonicalize: pureCanon)
        check("T3 상대경로 스칼라가 NFC",
              m3.map { Array($0.relPath.unicodeScalars) == Array("한글 노트.md".precomposedStringWithCanonicalMapping.unicodeScalars) } ?? false)
        eq("T3 NFD 매칭", m3?.vaultPath, nfcVault)
        eq("T3 상대경로 NFC", m3?.relPath, "한글 노트.md")
        eq("T3 볼트 이름 NFC", m3?.vaultName, "코워크")
        // 볼트 경로가 NFD여도 매칭
        let m3b = findVault(filePath: "/Users/u/코워크/x.md",
                            vaults: [nfcVault.decomposedStringWithCanonicalMapping], canonicalize: pureCanon)
        check("T3 NFD 볼트 매칭", m3b != nil)
        eq("T3 NFD 볼트 이름 NFC", m3b?.vaultName, "코워크")

        // T4 대소문자
        let m4 = findVault(filePath: "/users/U/Vault/Notes/A.md", vaults: ["/Users/u/vault"], canonicalize: pureCanon)
        eq("T4 대소문자 매칭", m4?.vaultPath, "/Users/u/vault")
        eq("T4 상대경로는 파일 쪽 대소문자 유지", m4?.relPath, "Notes/A.md")

        // T5 경계
        let m5 = findVault(filePath: "/a/vault2/x.md", vaults: ["/a/vault"], canonicalize: pureCanon)
        check("T5 /a/vault2 는 /a/vault 에 걸리지 않음", m5 == nil, "got=\(String(describing: m5))")
        let m5b = findVault(filePath: "/a/vault/x.md", vaults: ["/a/vault/"], canonicalize: pureCanon)
        eq("T5 끝 슬래시 볼트", m5b?.relPath, "x.md")

        // T6 중첩 볼트
        let m6 = findVault(filePath: "/a/v/sub/n.md", vaults: ["/a/v", "/a/v/sub"], canonicalize: pureCanon)
        eq("T6 더 긴 볼트", m6?.vaultPath, "/a/v/sub")
        eq("T6 상대경로", m6?.relPath, "n.md")
        let m6b = findVault(filePath: "/a/v/sub/n.md", vaults: ["/a/v/sub", "/a/v"], canonicalize: pureCanon)
        eq("T6 순서 무관", m6b?.vaultPath, "/a/v/sub")

        // T3/T4 실파일: 심볼릭 링크(/tmp → /private/tmp)와 기본 canonicalPath
        let base = "/tmp/odc-tests-\(getpid())-link"
        try? FileManager.default.createDirectory(atPath: base + "/vault", withIntermediateDirectories: true)
        FileManager.default.createFile(atPath: base + "/vault/n.md", contents: Data())
        let mReal = findVault(filePath: "/private" + base + "/vault/n.md", vaults: [base + "/vault"])
        eq("T3 /private 심볼릭 링크 해석", mReal?.relPath, "n.md")

        // T7 community-plugins.json
        let vHas = tempDir("has")
        FileManager.default.createFile(atPath: vHas + "/.obsidian/community-plugins.json",
                                       contents: Data("[\"dataview\",\"obsidian-advanced-uri\"]".utf8))
        let vNone = tempDir("none")
        let vBroken = tempDir("broken")
        FileManager.default.createFile(atPath: vBroken + "/.obsidian/community-plugins.json",
                                       contents: Data("[not json".utf8))
        eq("T7 있음 → adv-uri", resolveURIMode(config: HandlerConfig(), vaultPath: vHas), .advanced)
        eq("T7 없음 → official", resolveURIMode(config: HandlerConfig(), vaultPath: vNone), .official)
        eq("T7 깨짐 → official", resolveURIMode(config: HandlerConfig(), vaultPath: vBroken), .official)

        // T8 uriMode 강제
        eq("T8 official 강제", resolveURIMode(config: HandlerConfig(uriMode: "official"), vaultPath: vHas), .official)
        eq("T8 adv-uri 강제", resolveURIMode(config: HandlerConfig(uriMode: "adv-uri"), vaultPath: vNone), .advanced)
        eq("T8 auto 는 감지", resolveURIMode(config: HandlerConfig(uriMode: "auto"), vaultPath: vHas), .advanced)
        eq("T8 설정 파싱", parseHandlerConfig(Data("{\"uriMode\":\"official\",\"fallbackApp\":\"com.x\"}".utf8)),
           HandlerConfig(uriMode: "official", fallbackApp: "com.x"))
        eq("T8 깨진 설정 → 기본", parseHandlerConfig(Data("{".utf8)), HandlerConfig())

        // T9 폴백 탐색
        let all: Set<String> = ["abnerworks.Typora", "com.microsoft.VSCode", "com.apple.TextEdit"]
        eq("T9 Typora 우선", pickFallback(configured: nil, isInstalled: { all.contains($0) }), .detected("abnerworks.Typora"))
        eq("T9 VS Code 다음", pickFallback(configured: nil, isInstalled: { $0 != "abnerworks.Typora" && all.contains($0) }),
           .detected("com.microsoft.VSCode"))
        eq("T9 TextEdit 마지막", pickFallback(configured: nil, isInstalled: { $0 == "com.apple.TextEdit" }),
           .detected("com.apple.TextEdit"))
        eq("T9 설정값 우선", pickFallback(configured: "com.microsoft.VSCode", isInstalled: { all.contains($0) }),
           .configured("com.microsoft.VSCode"))
        eq("T9 설정이 옵시디언이면 무시", pickFallback(configured: "md.obsidian", isInstalled: { _ in true }),
           .detected("abnerworks.Typora"))
        eq("T9 설정이 자기 자신 경로면 무시",
           pickFallback(configured: "/Users/u/Applications/ObsidianDoubleclick.app", isInstalled: { _ in true }),
           .detected("abnerworks.Typora"))
        eq("T9 후보에 섞인 자기 자신·옵시디언 제외",
           pickFallback(configured: nil, candidates: ["md.obsidian", selfBundleID, "com.apple.TextEdit"],
                        isInstalled: { _ in true }),
           .detected("com.apple.TextEdit"))
        eq("T9 아무것도 없음", pickFallback(configured: nil, isInstalled: { _ in false }), .none)

        // 볼트 목록 파싱
        let vj = Data("{\"vaults\":{\"a\":{\"path\":\"/b\",\"ts\":1},\"c\":{\"path\":\"/a\"},\"d\":{}}}".utf8)
        eq("vaults 파싱", parseVaults(vj), ["/a", "/b"])
        check("vaults 깨짐 → nil", parseVaults(Data("x".utf8)) == nil)

        print("\n\(passes) passed, \(failures) failed")
        exit(failures == 0 ? 0 : 1)
    }
}
