import Foundation

// Codable mirrors of the JSON shapes emitted by `skillhub ... --json`.
// The decoder uses `.convertFromSnakeCase`, so snake_case JSON keys map onto
// camelCase Swift properties automatically (e.g. skill_count -> skillCount).

struct Skill: Codable, Hashable, Identifiable {
    var identity: String
    var name: String
    var namespace: String
    var version: String
    var description: String
    var targets: [String]
    var tags: [String]?
    var source: Source
    var maintainers: [String]?
    var license: String?
    var trust: Trust
    var featured: Bool
    var updatedAt: String
    var checksum: String?

    var id: String { identity }

    struct Source: Codable, Hashable {
        var type: String
        var url: String?
        var path: String
        var ref: String?
    }

    struct Trust: Codable, Hashable {
        var level: String
        var reviewedAt: String?
        var reviewer: String?
    }
}

/// One row from `catalog list --json` / `search --json`.
struct CatalogResult: Codable, Hashable, Identifiable {
    var registry: String
    var skill: Skill

    var id: String { "\(registry)/\(skill.identity)" }
    var installSpec: String { "\(registry)/\(skill.identity)" }
}

/// One row from `list --json`.
struct InstalledRow: Codable, Hashable, Identifiable {
    var scope: String
    var skill: String
    var version: String
    var location: String

    var id: String { "\(scope):\(skill)" }
    var isGlobal: Bool { scope == "global" }
}

/// One row from `check --json`.
struct UpdatePlan: Codable, Hashable, Identifiable {
    var identity: String
    var currentVersion: String
    var availableVersion: String
    var currentCommit: String?
    var availableCommit: String?
    var source: String
    var targets: [String]?
    var deployedTo: [String]?
    var held: Bool
    var holdReason: String?

    var id: String { identity }
}

/// One row from `deploy status --json`.
struct DeployStatus: Codable, Hashable, Identifiable {
    var identity: String
    var runtime: String
    var state: String

    var id: String { "\(identity)@\(runtime)" }
}

/// One row from `registry list --json` and the doctor report registries array.
struct RegistryStatus: Codable, Hashable, Identifiable {
    var name: String
    var type: String
    var location: String
    var skillCount: Int
    var generatedAt: String

    var id: String { name }
}

/// One row from `holds --json`.
struct Hold: Codable, Hashable, Identifiable {
    var skill: String
    var version: String
    var reason: String?
    var heldAt: String

    var id: String { skill }
}

/// The object from `doctor --json`.
struct DoctorReport: Codable {
    var config: String
    var configPath: String
    var home: String
    var installDir: String
    var runtimes: [Runtime]
    var registries: [RegistryStatus]
    var installed: Int

    struct Runtime: Codable, Hashable, Identifiable {
        var name: String
        var dir: String
        var id: String { name }
    }
}
