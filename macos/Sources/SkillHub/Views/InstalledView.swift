import SwiftUI

struct InstalledView: View {
    @EnvironmentObject var store: AppStore
    @State private var confirmUninstall: InstalledRow?
    @State private var removeDeployed = false

    var body: some View {
        content
        .navigationTitle("Installed")
        .toolbar {
            ToolbarItem {
                Button {
                    Task { await store.refreshInstalled() }
                } label: { Label("Refresh", systemImage: "arrow.clockwise") }
                .disabled(store.isBusy)
            }
        }
        .confirmationDialog(
            "Uninstall \(confirmUninstall?.skill ?? "")?",
            isPresented: Binding(get: { confirmUninstall != nil },
                                 set: { if !$0 { confirmUninstall = nil } }),
            titleVisibility: .visible
        ) {
            Toggle("Also remove runtime copies", isOn: $removeDeployed)
            Button("Uninstall", role: .destructive) {
                if let row = confirmUninstall {
                    Task { await store.uninstall(row.skill, removeDeployed: removeDeployed) }
                }
                confirmUninstall = nil
            }
            Button("Cancel", role: .cancel) { confirmUninstall = nil }
        }
    }

    @ViewBuilder
    private var content: some View {
        if store.installed.isEmpty {
            ContentUnavailableCompat(
                title: "No installed skills",
                systemImage: "shippingbox",
                description: "Install skills from the Catalog to manage them here."
            )
        } else {
            List {
                if !globals.isEmpty {
                    Section("Managed (global)") {
                        ForEach(globals) { row in
                            InstalledRowView(row: row, held: holdInfo(row), onUninstall: { confirm(row) })
                        }
                    }
                }
                if !projects.isEmpty {
                    Section("Project") {
                        ForEach(projects) { row in
                            InstalledRowView(row: row, held: nil)
                        }
                    }
                }
            }
            .listStyle(.inset)
        }
    }

    private var globals: [InstalledRow] { store.installed.filter { $0.isGlobal } }
    private var projects: [InstalledRow] { store.installed.filter { !$0.isGlobal } }

    private func confirm(_ row: InstalledRow) {
        removeDeployed = false
        confirmUninstall = row
    }

    private func holdInfo(_ row: InstalledRow) -> Hold? {
        store.holds.first { $0.skill == row.skill }
    }
}

struct InstalledRowView: View {
    @EnvironmentObject var store: AppStore
    let row: InstalledRow
    let held: Hold?
    var onUninstall: (() -> Void)? = nil

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: 6) {
                    Text(row.skill).font(.body.weight(.medium))
                    TagChip(text: "v\(row.version)", color: .gray)
                    if held != nil {
                        TagChip(text: "held", color: .orange)
                    }
                }
                if !row.location.isEmpty {
                    Text(row.location).font(.caption).foregroundStyle(.secondary).lineLimit(1)
                }
                if let reason = held?.reason, !reason.isEmpty {
                    Text("Hold: \(reason)").font(.caption2).foregroundStyle(.orange)
                }
            }
            Spacer()
            if onUninstall != nil {
                actions
            }
        }
        .padding(.vertical, 3)
    }

    @ViewBuilder
    private var actions: some View {
        HStack(spacing: 4) {
            if held == nil {
                Button("Hold") { Task { await store.setHold(row.skill, held: true) } }
            } else {
                Button("Unhold") { Task { await store.setHold(row.skill, held: false) } }
            }
            Button("Rollback") { Task { await store.rollback(row.skill) } }
            Button(role: .destructive) { onUninstall?() } label: {
                Image(systemName: "trash")
            }
        }
        .buttonStyle(.bordered)
        .controlSize(.small)
        .disabled(store.isBusy)
    }
}

struct UpdatesView: View {
    @EnvironmentObject var store: AppStore

    var body: some View {
        updatesContent
        .navigationTitle("Updates")
        .toolbar {
            ToolbarItem {
                Button { Task { await store.checkUpdates() } } label: {
                    Label("Check", systemImage: "arrow.clockwise")
                }
                .disabled(store.isBusy)
            }
            ToolbarItem(placement: .primaryAction) {
                Button { Task { await store.update(nil) } } label: {
                    Label("Update All", systemImage: "arrow.down.circle")
                }
                .disabled(store.isBusy || store.updates.allSatisfy { $0.held })
            }
        }
        .task { await store.checkUpdates() }
    }

    @ViewBuilder
    private var updatesContent: some View {
        if store.updates.isEmpty {
            ContentUnavailableCompat(
                title: "Everything is current",
                systemImage: "checkmark.seal",
                description: "No managed skills have updates available."
            )
        } else {
            List(store.updates) { plan in
                HStack {
                    VStack(alignment: .leading, spacing: 2) {
                        HStack(spacing: 6) {
                            Text(plan.identity).font(.body.weight(.medium))
                            if plan.held { TagChip(text: "held", color: .orange) }
                        }
                        Text("\(plan.currentVersion) → \(plan.availableVersion)  ·  \(plan.source)")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button("Update") { Task { await store.update(plan.identity) } }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        .disabled(store.isBusy || plan.held)
                }
                .padding(.vertical, 3)
            }
            .listStyle(.inset)
        }
    }
}
