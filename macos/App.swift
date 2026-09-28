// obsidian-doubleclick (macOS): Finder 더블클릭 핸들러.
// 볼트 안 .md는 obsidian:// URI로 옵시디언에서 열고, 볼트 밖 파일은 폴백 편집기
// (Typora → VS Code → TextEdit)로 연다. 로직은 Core.swift, 여기는 AppKit 접점만 둔다.
//
// 더블클릭은 argv가 아니라 Apple Event(odoc)로 온다 → application(_:open:)에서 받는다.
//
// CLI (Contents/MacOS/obsidian-doubleclick):
//   --debug <파일경로>   볼트·모드·URI를 JSON으로 출력만 하고 열지 않는다
//   --set-default        .md 기본 앱을 이 앱으로 지정한다
//   --version            버전 출력

import AppKit
import CoreServices
import UniformTypeIdentifiers

// ---------------------------------------------------------------------------
// 경로·로그
// ---------------------------------------------------------------------------

let home = FileManager.default.homeDirectoryForCurrentUser.path
let obsidianJSONPath = home + "/Library/Application Support/obsidian/obsidian.json"
let configPath = home + "/Library/Application Support/obsidian-doubleclick/config.json"
let logPath = home + "/Library/Logs/obsidian-doubleclick.log"

func writeLog(_ msg: String) {
    let fmt = ISO8601DateFormatter()
    fmt.timeZone = TimeZone(identifier: "UTC")
    let line = fmt.string(from: Date()) + " " + msg + "\n"
    let data = Data(line.utf8)
    if let h = FileHandle(forWritingAtPath: logPath) {
        h.seekToEndOfFile()
        h.write(data)
        h.closeFile()
    } else {
        try? FileManager.default.createDirectory(atPath: (logPath as NSString).deletingLastPathComponent,
                                                 withIntermediateDirectories: true)
        FileManager.default.createFile(atPath: logPath, contents: data)
    }
}

func loadVaults() -> [String] {
    guard let data = FileManager.default.contents(atPath: obsidianJSONPath) else {
        writeLog("WARN obsidian.json 읽기 실패: " + obsidianJSONPath)
        return []
    }
    guard let vaults = parseVaults(data) else {
        writeLog("WARN obsidian.json 파싱 실패")
        return []
    }
    return vaults
}

func loadHandlerConfig() -> HandlerConfig {
    return parseHandlerConfig(FileManager.default.contents(atPath: configPath))
}

// 번들 ID 또는 앱 경로를 앱 URL로 바꾼다.
func appURL(for app: String) -> URL? {
    if app.hasPrefix("/") || app.hasPrefix("~") {
        let p = (app as NSString).expandingTildeInPath
        return FileManager.default.fileExists(atPath: p) ? URL(fileURLWithPath: p) : nil
    }
    return NSWorkspace.shared.urlForApplication(withBundleIdentifier: app)
}

// ---------------------------------------------------------------------------
// 파일 처리
// ---------------------------------------------------------------------------

// handle: 파일 하나를 처리한다. 비동기로 여는 경우 끝나면 done을 부른다.
func handle(filePath: String, done: @escaping () -> Void) {
    let cfg = loadHandlerConfig()
    let vaults = loadVaults()

    if let m = findVault(filePath: filePath, vaults: vaults) {
        let mode = resolveURIMode(config: cfg, vaultPath: m.vaultPath)
        let uri = buildURI(vaultName: m.vaultName, relPath: m.relPath, mode: mode)
        writeLog("LAUNCH obsidian-uri uri_mode=\(mode.rawValue) \(uri)")
        guard let url = URL(string: uri) else {
            writeLog("WARN URI 생성 실패: " + uri)
            done()
            return
        }
        if !NSWorkspace.shared.open(url) {
            writeLog("WARN NSWorkspace.open 실패: " + uri)
        }
        done()
        return
    }

    let choice = pickFallback(configured: cfg.fallbackApp, isInstalled: { appURL(for: $0) != nil })
    let app: String
    switch choice {
    case .configured(let a):
        writeLog("LAUNCH fallback-config " + a)
        app = a
    case .detected(let a):
        if let c = cfg.fallbackApp, !c.isEmpty {
            writeLog("WARN fallbackApp 사용 불가(미설치 또는 자기 자신·옵시디언): " + c)
        }
        writeLog("LAUNCH fallback-detected " + a)
        app = a
    case .none:
        writeLog("WARN no fallback found — 파일을 열 수 없음: " + filePath)
        done()
        return
    }
    guard let appURL = appURL(for: app) else {
        writeLog("WARN 폴백 앱 경로 없음: " + app)
        done()
        return
    }
    let conf = NSWorkspace.OpenConfiguration()
    NSWorkspace.shared.open([URL(fileURLWithPath: filePath)], withApplicationAt: appURL,
                            configuration: conf) { _, err in
        if let err = err {
            writeLog("WARN 폴백 열기 실패 app=\(app) error=\(err.localizedDescription)")
        } else {
            writeLog("LAUNCH fallback-opened \(app)")
        }
        done()
    }
}

// ---------------------------------------------------------------------------
// 기본 앱 지정
// ---------------------------------------------------------------------------

func currentDefaultHandler() -> String? {
    return LSCopyDefaultRoleHandlerForContentType(markdownUTI as CFString, .all)?.takeRetainedValue() as String?
}

// setDefault: NSWorkspace API로 지정하고 LaunchServices로 재확인한다.
// 결과가 다르면 LSSetDefaultRoleHandlerForContentType로 한 번 더 시도한다.
func setDefault() -> Int32 {
    let before = currentDefaultHandler() ?? "(none)"
    let selfURL = Bundle.main.bundleURL
    guard let type = UTType(markdownUTI) else {
        print("ERROR UTType \(markdownUTI) 없음")
        return 1
    }
    // macOS는 기본 앱 변경 전에 확인 창을 띄우고, 사용자가 답해야 완료 핸들러가 불린다
    // (2026-09-28 macOS 26.6 실측). 메인 런루프를 돌리며 기다린다.
    var finished = false
    var apiError: Error?
    NSWorkspace.shared.setDefaultApplication(at: selfURL, toOpen: type) { err in
        apiError = err
        finished = true
    }
    print("macOS may show a confirmation dialog — choose to use Obsidian Doubleclick. Waiting up to 60 s…")
    let t0 = Date()
    while !finished && Date().timeIntervalSince(t0) < 60 {
        RunLoop.main.run(until: Date().addingTimeInterval(0.2))
    }
    if !finished {
        writeLog("WARN set-default: 확인 창 응답 없음(60초)")
    }
    if let e = apiError {
        writeLog("WARN set-default: setDefaultApplication 실패 " + e.localizedDescription)
    }
    var after = currentDefaultHandler() ?? "(none)"
    var method = "NSWorkspace"
    if after.lowercased() != selfBundleID {
        let st = LSSetDefaultRoleHandlerForContentType(markdownUTI as CFString, .all, selfBundleID as CFString)
        method = "LSSetDefaultRoleHandler(status=\(st))"
        after = currentDefaultHandler() ?? "(none)"
    }
    let ok = after.lowercased() == selfBundleID
    let confirmed = finished && apiError == nil
    let msg = "SET-DEFAULT \(ok ? "ok" : "fail") method=\(method) confirmed=\(confirmed) before=\(before) after=\(after) app=\(selfURL.path)"
    writeLog(msg)
    print(msg)
    return ok ? 0 : 1
}

// ---------------------------------------------------------------------------
// AppDelegate
// ---------------------------------------------------------------------------

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var pending = 0
    private var quitTimer: Timer?

    func applicationDidFinishLaunching(_ notification: Notification) {
        // 파일 없이 앱만 실행된 경우에도 곧 종료한다.
        scheduleQuit(after: 2.0)
    }

    func application(_ application: NSApplication, open urls: [URL]) {
        quitTimer?.invalidate()
        for url in urls where url.isFileURL {
            pending += 1
            handle(filePath: url.path) { [weak self] in
                DispatchQueue.main.async { self?.finishOne() }
            }
        }
        if pending == 0 { scheduleQuit(after: 1.0) }
    }

    private func finishOne() {
        pending -= 1
        if pending <= 0 { scheduleQuit(after: 1.0) }
    }

    // 마지막 이벤트 1초 뒤 종료한다. 그 사이에 새 odoc가 오면 타이머를 다시 건다.
    private func scheduleQuit(after seconds: TimeInterval) {
        quitTimer?.invalidate()
        quitTimer = Timer.scheduledTimer(withTimeInterval: seconds, repeats: false) { _ in
            NSApp.terminate(nil)
        }
    }
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

@main
struct Main {
    static func main() {
        let args = Array(CommandLine.arguments.dropFirst())

        if args.contains("--version") {
            print(appVersion)
            return
        }
        if args.contains("--set-default") {
            exit(setDefault())
        }
        if let i = args.firstIndex(of: "--debug") {
            guard i + 1 < args.count else {
                print("usage: obsidian-doubleclick --debug <파일경로>")
                exit(2)
            }
            let info = makeDebugInfo(filePath: args[i + 1], vaults: loadVaults(), config: loadHandlerConfig())
            let enc = JSONEncoder()
            enc.outputFormatting = [.prettyPrinted, .sortedKeys, .withoutEscapingSlashes]
            if let data = try? enc.encode(info), let s = String(data: data, encoding: .utf8) {
                writeLog("DEBUG " + s)
                print(s)
            }
            return
        }

        let app = NSApplication.shared
        let delegate = AppDelegate()
        app.delegate = delegate
        app.run()   // Dock·메뉴 막대 숨김은 Info.plist의 LSUIElement가 맡는다
    }
}
