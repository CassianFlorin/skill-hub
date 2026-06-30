import Foundation

enum CLIError: LocalizedError {
    case binaryNotFound
    case launchFailed(String)
    case nonZeroExit(code: Int32, stdout: String, stderr: String)
    case decodeFailed(String)

    var errorDescription: String? {
        switch self {
        case .binaryNotFound:
            return "Could not locate the bundled skillhub binary. Set SKILLHUB_BIN or install skillhub on PATH."
        case let .launchFailed(message):
            return "Failed to launch skillhub: \(message)"
        case let .nonZeroExit(code, stdout, stderr):
            let detail = stderr.isEmpty ? stdout : stderr
            return "skillhub exited with code \(code): \(detail.trimmingCharacters(in: .whitespacesAndNewlines))"
        case let .decodeFailed(message):
            return "Could not read skillhub output: \(message)"
        }
    }
}

/// Thin wrapper that shells out to the `skillhub` CLI and decodes its JSON.
final class SkillHubCLI {
    let binaryURL: URL
    let workingDirectory: URL

    init() throws {
        guard let binary = SkillHubCLI.locateBinary() else {
            throw CLIError.binaryNotFound
        }
        binaryURL = binary
        workingDirectory = SkillHubCLI.makeWorkingDirectory()
    }

    /// Default working directory: a dedicated Application Support folder so the
    /// GUI keeps its registry config (`skillhub.yaml`) out of the user's home.
    /// The managed store itself still lives at $SKILLHUB_HOME (~/.skillhub).
    private static func makeWorkingDirectory() -> URL {
        let fm = FileManager.default
        let base = fm.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? fm.homeDirectoryForCurrentUser
        let dir = base.appendingPathComponent("SkillHub", isDirectory: true)
        try? fm.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir
    }

    private static func locateBinary() -> URL? {
        let fm = FileManager.default

        // 1. Bundled inside the .app (Contents/Resources/skillhub).
        if let bundled = Bundle.main.url(forResource: "skillhub", withExtension: nil),
           fm.isExecutableFile(atPath: bundled.path) {
            return bundled
        }

        // 2. Explicit override for development (`swift run`).
        if let override = ProcessInfo.processInfo.environment["SKILLHUB_BIN"],
           fm.isExecutableFile(atPath: override) {
            return URL(fileURLWithPath: override)
        }

        // 3. A skillhub binary sitting next to the repo (developer convenience).
        let cwd = URL(fileURLWithPath: fm.currentDirectoryPath)
        for candidate in [cwd.appendingPathComponent("skillhub"),
                          cwd.deletingLastPathComponent().appendingPathComponent("skillhub")] {
            if fm.isExecutableFile(atPath: candidate.path) {
                return candidate
            }
        }

        // 4. Anything on PATH.
        if let path = ProcessInfo.processInfo.environment["PATH"] {
            for dir in path.split(separator: ":") {
                let candidate = URL(fileURLWithPath: String(dir)).appendingPathComponent("skillhub")
                if fm.isExecutableFile(atPath: candidate.path) {
                    return candidate
                }
            }
        }
        return nil
    }

    /// Runs the CLI and returns stdout. Throws on a non-zero exit code.
    @discardableResult
    func run(_ arguments: [String]) async throws -> String {
        let binaryURL = self.binaryURL
        let workingDirectory = self.workingDirectory
        return try await withCheckedThrowingContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                let process = Process()
                process.executableURL = binaryURL
                process.arguments = arguments
                process.currentDirectoryURL = workingDirectory

                let stdoutPipe = Pipe()
                let stderrPipe = Pipe()
                process.standardOutput = stdoutPipe
                process.standardError = stderrPipe

                // Read both pipes concurrently so large output never deadlocks.
                var stdoutData = Data()
                var stderrData = Data()
                let group = DispatchGroup()
                group.enter()
                DispatchQueue.global().async {
                    stdoutData = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
                    group.leave()
                }
                group.enter()
                DispatchQueue.global().async {
                    stderrData = stderrPipe.fileHandleForReading.readDataToEndOfFile()
                    group.leave()
                }

                do {
                    try process.run()
                } catch {
                    continuation.resume(throwing: CLIError.launchFailed(error.localizedDescription))
                    return
                }

                process.waitUntilExit()
                group.wait()

                let stdout = String(data: stdoutData, encoding: .utf8) ?? ""
                let stderr = String(data: stderrData, encoding: .utf8) ?? ""

                if process.terminationStatus != 0 {
                    continuation.resume(throwing: CLIError.nonZeroExit(
                        code: process.terminationStatus, stdout: stdout, stderr: stderr))
                    return
                }
                continuation.resume(returning: stdout)
            }
        }
    }

    /// Runs the CLI and decodes its JSON stdout into `T`.
    func runJSON<T: Decodable>(_ arguments: [String], as type: T.Type) async throws -> T {
        let output = try await run(arguments + ["--json"])
        guard let data = output.data(using: .utf8) else {
            throw CLIError.decodeFailed("output was not valid UTF-8")
        }
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw CLIError.decodeFailed(error.localizedDescription)
        }
    }
}
