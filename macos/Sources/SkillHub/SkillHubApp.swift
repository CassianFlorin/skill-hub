import SwiftUI

@main
struct SkillHubApp: App {
    @StateObject private var store = AppStore()

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(store)
                .frame(minWidth: 900, minHeight: 560)
                .task { await store.refreshAll() }
        }
        .windowStyle(.titleBar)
        .commands {
            CommandGroup(after: .newItem) {
                Button("Sync Registries") {
                    Task { await store.syncAllRegistries() }
                }
                .keyboardShortcut("r", modifiers: [.command])
            }
        }
    }
}
