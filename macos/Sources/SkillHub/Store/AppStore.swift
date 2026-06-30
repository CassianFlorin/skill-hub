import Foundation
import SwiftUI

/// Sidebar destinations.
enum SidebarSection: String, CaseIterable, Identifiable {
    case catalog = "Catalog"
    case installed = "Installed"
    case updates = "Updates"
    case deploy = "Deploy"
    case doctor = "Doctor"

    var id: String { rawValue }
    var symbol: String {
        switch self {
        case .catalog: return "square.grid.2x2"
        case .installed: return "shippingbox"
        case .updates: return "arrow.triangle.2.circlepath"
        case .deploy: return "paperplane"
        case .doctor: return "stethoscope"
        }
    }
}

@MainActor
final class AppStore: ObservableObject {
    // Data
    @Published var catalog: [CatalogResult] = []
    @Published var installed: [InstalledRow] = []
    @Published var updates: [UpdatePlan] = []
    @Published var deployStatus: [DeployStatus] = []
    @Published var registries: [RegistryStatus] = []
    @Published var holds: [Hold] = []
    @Published var doctor: DoctorReport?

    // UI state
    @Published var searchText: String = ""
    @Published var isBusy: Bool = false
    @Published var statusMessage: String = ""
    @Published var errorMessage: String?
    @Published var cliAvailable: Bool = true

    /// The runtimes skillhub can deploy to.
    let runtimeNames = ["codex", "claude", "gemini", "hermes"]

    private var cli: SkillHubCLI?

    init() {
        do {
            cli = try SkillHubCLI()
        } catch {
            cliAvailable = false
            errorMessage = error.localizedDescription
        }
    }

    var filteredCatalog: [CatalogResult] {
        guard !searchText.trimmingCharacters(in: .whitespaces).isEmpty else { return catalog }
        let needle = searchText.lowercased()
        return catalog.filter {
            $0.skill.identity.lowercased().contains(needle)
                || $0.skill.description.lowercased().contains(needle)
                || ($0.skill.tags?.contains { $0.lowercased().contains(needle) } ?? false)
        }
    }

    private func requireCLI() throws -> SkillHubCLI {
        guard let cli else { throw CLIError.binaryNotFound }
        return cli
    }

    /// Runs `work`, toggling the busy flag and surfacing errors uniformly.
    private func perform(_ message: String, _ work: @escaping (SkillHubCLI) async throws -> Void) async {
        guard let cli else { return }
        isBusy = true
        statusMessage = message
        errorMessage = nil
        do {
            try await work(cli)
        } catch {
            errorMessage = error.localizedDescription
        }
        isBusy = false
    }

    // MARK: - Loads

    func refreshAll() async {
        await refreshDoctor()
        await refreshRegistries()
        await refreshInstalled()
        await refreshDeployStatus()
        await loadCatalog()
    }

    func loadCatalog() async {
        await perform("Loading catalog…") { cli in
            // Aggregate every configured registry's catalog.
            var combined: [CatalogResult] = []
            let registries = try await cli.runJSON(["registry", "list"], as: [RegistryStatus].self)
            for registry in registries {
                if let rows = try? await cli.runJSON(
                    ["catalog", "list", "--registry", registry.name], as: [CatalogResult].self) {
                    combined.append(contentsOf: rows)
                }
            }
            self.catalog = combined.sorted { $0.skill.identity < $1.skill.identity }
            self.statusMessage = "Catalog: \(combined.count) skills"
        }
    }

    func syncAllRegistries() async {
        await perform("Syncing registries…") { cli in
            let registries = try await cli.runJSON(["registry", "list"], as: [RegistryStatus].self)
            for registry in registries {
                _ = try? await cli.run(["registry", "sync", registry.name])
            }
        }
        await loadCatalog()
        await refreshRegistries()
    }

    func refreshRegistries() async {
        await perform("Loading registries…") { cli in
            self.registries = try await cli.runJSON(["registry", "list"], as: [RegistryStatus].self)
        }
    }

    func refreshInstalled() async {
        await perform("Loading installed skills…") { cli in
            self.installed = try await cli.runJSON(["list"], as: [InstalledRow].self)
            self.holds = (try? await cli.runJSON(["holds"], as: [Hold].self)) ?? []
        }
    }

    func checkUpdates() async {
        await perform("Checking for updates…") { cli in
            self.updates = try await cli.runJSON(["check"], as: [UpdatePlan].self)
            self.statusMessage = self.updates.isEmpty ? "All skills are current" : "\(self.updates.count) update(s) available"
        }
    }

    func refreshDeployStatus() async {
        await perform("Loading deploy status…") { cli in
            self.deployStatus = try await cli.runJSON(["deploy", "status"], as: [DeployStatus].self)
        }
    }

    func refreshDoctor() async {
        await perform("Running doctor…") { cli in
            self.doctor = try await cli.runJSON(["doctor"], as: DoctorReport.self)
        }
    }

    // MARK: - Mutations

    func install(_ result: CatalogResult) async {
        await perform("Installing \(result.skill.identity)…") { cli in
            let output = try await cli.run(["install", result.installSpec])
            self.statusMessage = output.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        await refreshInstalled()
        await checkUpdates()
    }

    func uninstall(_ identity: String, removeDeployed: Bool) async {
        await perform("Uninstalling \(identity)…") { cli in
            var args = ["uninstall", identity]
            if removeDeployed { args.append("--deployed") }
            let output = try await cli.run(args)
            self.statusMessage = output.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        await refreshInstalled()
        await refreshDeployStatus()
    }

    func update(_ identity: String?) async {
        let label = identity ?? "all skills"
        await perform("Updating \(label)…") { cli in
            var args = ["update"]
            if let identity { args.append(identity) }
            let output = try await cli.run(args)
            self.statusMessage = output.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        await refreshInstalled()
        await checkUpdates()
    }

    func setHold(_ identity: String, held: Bool, reason: String = "") async {
        await perform(held ? "Holding \(identity)…" : "Unholding \(identity)…") { cli in
            if held {
                var args = ["hold", identity]
                if !reason.isEmpty { args.append(contentsOf: ["--reason", reason]) }
                _ = try await cli.run(args)
            } else {
                _ = try await cli.run(["unhold", identity])
            }
        }
        await refreshInstalled()
        await checkUpdates()
    }

    func rollback(_ identity: String) async {
        await perform("Rolling back \(identity)…") { cli in
            let output = try await cli.run(["rollback", identity])
            self.statusMessage = output.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        await refreshInstalled()
    }

    func deploy(_ runtime: String, identity: String?, force: Bool) async {
        await perform("Deploying to \(runtime)…") { cli in
            var args = ["deploy", runtime]
            if let identity { args.append(identity) }
            if force { args.append("--force") }
            let output = try await cli.run(args)
            self.statusMessage = output.trimmingCharacters(in: .whitespacesAndNewlines)
        }
        await refreshDeployStatus()
    }

    func addRegistry(type: String, name: String, location: String) async {
        await perform("Adding registry \(name)…") { cli in
            _ = try await cli.run(["registry", "add", type, name, location])
            _ = try? await cli.run(["registry", "sync", name])
        }
        await refreshRegistries()
        await loadCatalog()
    }
}
