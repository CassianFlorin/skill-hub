import SwiftUI

struct DeployView: View {
    @EnvironmentObject var store: AppStore
    @State private var runtime = "codex"
    @State private var force = false

    private var globals: [InstalledRow] { store.installed.filter { $0.isGlobal } }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            controls
            Divider()
            HSplitView {
                skillList.frame(minWidth: 280)
                statusList.frame(minWidth: 280, maxWidth: .infinity)
            }
        }
        .navigationTitle("Deploy")
        .task { await store.refreshDeployStatus() }
    }

    private var controls: some View {
        HStack(spacing: 12) {
            Picker("Runtime", selection: $runtime) {
                ForEach(store.runtimeNames, id: \.self) { Text($0.capitalized).tag($0) }
            }
            .pickerStyle(.segmented)
            .fixedSize()

            Toggle("Force overwrite", isOn: $force)
                .help("Replace an existing runtime copy")

            Spacer()

            Button {
                Task { await store.deploy(runtime, identity: nil, force: force) }
            } label: {
                Label("Deploy All", systemImage: "paperplane.fill")
            }
            .buttonStyle(.borderedProminent)
            .disabled(store.isBusy || globals.isEmpty)
        }
        .padding(12)
    }

    private var skillList: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Managed skills").font(.headline).padding(8)
            if globals.isEmpty {
                ContentUnavailableCompat(
                    title: "Nothing to deploy",
                    systemImage: "shippingbox",
                    description: "Install skills first."
                )
            } else {
                List(globals) { row in
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(row.skill).font(.body.weight(.medium))
                            Text("v\(row.version)").font(.caption).foregroundStyle(.secondary)
                        }
                        Spacer()
                        Button("Deploy") {
                            Task { await store.deploy(runtime, identity: row.skill, force: force) }
                        }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        .disabled(store.isBusy)
                    }
                    .padding(.vertical, 2)
                }
                .listStyle(.inset)
            }
        }
    }

    private var statusList: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Deploy status").font(.headline)
                Spacer()
                Button { Task { await store.refreshDeployStatus() } } label: {
                    Image(systemName: "arrow.clockwise")
                }
                .buttonStyle(.borderless)
                .disabled(store.isBusy)
            }
            .padding(8)

            if store.deployStatus.isEmpty {
                ContentUnavailableCompat(
                    title: "No runtime copies",
                    systemImage: "paperplane",
                    description: "Deploy a skill to see its runtime state."
                )
            } else {
                List(store.deployStatus) { status in
                    HStack {
                        Text(status.identity)
                        Spacer()
                        TagChip(text: status.runtime, color: .blue)
                        StateChip(state: status.state)
                    }
                    .padding(.vertical, 2)
                }
                .listStyle(.inset)
            }
        }
    }
}

struct StateChip: View {
    let state: String

    private var color: Color {
        switch state {
        case "deployed": return .green
        case "missing", "not-deployed": return .gray
        case "stale", "would-deploy": return .orange
        case "conflict": return .red
        default: return .secondary
        }
    }

    var body: some View { TagChip(text: state, color: color) }
}
