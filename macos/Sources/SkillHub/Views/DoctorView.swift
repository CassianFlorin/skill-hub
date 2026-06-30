import SwiftUI

struct DoctorView: View {
    @EnvironmentObject var store: AppStore

    var body: some View {
        ScrollView {
            if let doctor = store.doctor {
                VStack(alignment: .leading, spacing: 16) {
                    summary(doctor)
                    paths(doctor)
                    runtimes(doctor)
                    registries(doctor)
                }
                .padding(20)
                .frame(maxWidth: .infinity, alignment: .leading)
            } else {
                ContentUnavailableCompat(
                    title: "No diagnostics yet",
                    systemImage: "stethoscope",
                    description: "Run doctor to inspect local config and runtime readiness."
                )
                .frame(minHeight: 400)
            }
        }
        .navigationTitle("Doctor")
        .toolbar {
            ToolbarItem {
                Button { Task { await store.refreshDoctor() } } label: {
                    Label("Run", systemImage: "arrow.clockwise")
                }
                .disabled(store.isBusy)
            }
        }
    }

    private func summary(_ d: DoctorReport) -> some View {
        HStack(spacing: 16) {
            StatCard(title: "Config", value: d.config, symbol: "checkmark.seal",
                     color: d.config == "ok" ? .green : .orange)
            StatCard(title: "Installed", value: "\(d.installed)", symbol: "shippingbox", color: .blue)
            StatCard(title: "Registries", value: "\(d.registries.count)", symbol: "tray.full", color: .purple)
            StatCard(title: "Runtimes", value: "\(d.runtimes.count)", symbol: "cpu", color: .teal)
        }
    }

    private func paths(_ d: DoctorReport) -> some View {
        GroupBox("Paths") {
            Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 6) {
                pathRow("Config", d.configPath)
                pathRow("Home", d.home)
                pathRow("Install dir", d.installDir)
            }
            .padding(4)
        }
    }

    private func runtimes(_ d: DoctorReport) -> some View {
        GroupBox("Runtime directories") {
            Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 6) {
                ForEach(d.runtimes) { rt in
                    pathRow(rt.name.capitalized, rt.dir)
                }
            }
            .padding(4)
        }
    }

    @ViewBuilder
    private func registries(_ d: DoctorReport) -> some View {
        GroupBox("Registries") {
            if d.registries.isEmpty {
                Text("No registries configured").foregroundStyle(.secondary).padding(4)
            } else {
                VStack(spacing: 0) {
                    ForEach(d.registries) { reg in
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                HStack(spacing: 6) {
                                    Text(reg.name).font(.body.weight(.medium))
                                    TagChip(text: reg.type, color: .blue)
                                }
                                Text(reg.location).font(.caption).foregroundStyle(.secondary).lineLimit(1)
                            }
                            Spacer()
                            VStack(alignment: .trailing, spacing: 2) {
                                Text("\(reg.skillCount) skills").font(.caption)
                                if !reg.generatedAt.isEmpty {
                                    Text(reg.generatedAt).font(.caption2).foregroundStyle(.secondary)
                                }
                            }
                        }
                        .padding(.vertical, 4)
                        if reg.id != d.registries.last?.id { Divider() }
                    }
                }
                .padding(4)
            }
        }
    }

    @ViewBuilder
    private func pathRow(_ label: String, _ value: String) -> some View {
        GridRow {
            Text(label).foregroundStyle(.secondary).gridColumnAlignment(.trailing)
            Text(value).font(.callout.monospaced()).textSelection(.enabled)
        }
    }
}

struct StatCard: View {
    let title: String
    let value: String
    let symbol: String
    let color: Color

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            Image(systemName: symbol).foregroundStyle(color)
            Text(value).font(.title2.bold())
            Text(title).font(.caption).foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .background(color.opacity(0.10), in: RoundedRectangle(cornerRadius: 10))
    }
}
