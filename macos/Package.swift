// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "SkillHub",
    platforms: [.macOS(.v13)],
    targets: [
        .executableTarget(
            name: "SkillHub",
            path: "Sources/SkillHub"
        )
    ]
)
