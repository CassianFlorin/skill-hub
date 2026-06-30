import SwiftUI

struct RootView: View {
    @EnvironmentObject var store: AppStore
    @State private var selection: SidebarSection = .catalog

    var body: some View {
        NavigationSplitView {
            List(SidebarSection.allCases, selection: $selection) { section in
                Label(section.rawValue, systemImage: section.symbol)
                    .tag(section)
                    .badge(badge(for: section))
            }
            .navigationSplitViewColumnWidth(min: 180, ideal: 200)
            .listStyle(.sidebar)
        } detail: {
            detail
                .toolbar { toolbar }
                .safeAreaInset(edge: .bottom) { statusBar }
        }
        .alert("Something went wrong",
               isPresented: Binding(
                get: { store.errorMessage != nil },
                set: { if !$0 { store.errorMessage = nil } })) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(store.errorMessage ?? "")
        }
    }

    @ViewBuilder
    private var detail: some View {
        if !store.cliAvailable {
            CLIMissingView()
        } else {
            switch selection {
            case .catalog: CatalogView()
            case .installed: InstalledView()
            case .updates: UpdatesView()
            case .deploy: DeployView()
            case .doctor: DoctorView()
            }
        }
    }

    private func badge(for section: SidebarSection) -> Int {
        switch section {
        case .installed: return store.installed.count
        case .updates: return store.updates.count
        default: return 0
        }
    }

    @ToolbarContentBuilder
    private var toolbar: some ToolbarContent {
        ToolbarItem(placement: .primaryAction) {
            Button {
                Task { await store.syncAllRegistries() }
            } label: {
                Label("Sync", systemImage: "arrow.clockwise")
            }
            .disabled(store.isBusy)
            .help("Sync all registries and reload the catalog")
        }
    }

    private var statusBar: some View {
        HStack(spacing: 8) {
            if store.isBusy {
                ProgressView().controlSize(.small)
            }
            Text(store.isBusy ? store.statusMessage : (store.statusMessage.isEmpty ? "Ready" : store.statusMessage))
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(1)
            Spacer()
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
        .background(.bar)
    }
}

struct CLIMissingView: View {
    @EnvironmentObject var store: AppStore

    var body: some View {
        ContentUnavailableCompat(
            title: "skillhub CLI not found",
            systemImage: "exclamationmark.triangle",
            description: store.errorMessage
                ?? "Bundle the skillhub binary into the app, set SKILLHUB_BIN, or install it on your PATH."
        )
    }
}

/// `ContentUnavailableView` is macOS 14+, so provide a small fallback.
struct ContentUnavailableCompat: View {
    let title: String
    let systemImage: String
    let description: String

    var body: some View {
        VStack(spacing: 12) {
            Image(systemName: systemImage)
                .font(.system(size: 40))
                .foregroundStyle(.secondary)
            Text(title).font(.title3.bold())
            Text(description)
                .font(.callout)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding()
    }
}
