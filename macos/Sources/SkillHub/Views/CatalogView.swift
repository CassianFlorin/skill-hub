import SwiftUI

struct CatalogView: View {
    @EnvironmentObject var store: AppStore
    @State private var selectedID: CatalogResult.ID?

    var body: some View {
        HSplitView {
            list
                .frame(minWidth: 280, idealWidth: 340)
            detail
                .frame(minWidth: 320, maxWidth: .infinity, maxHeight: .infinity)
        }
        .navigationTitle("Catalog")
    }

    private var list: some View {
        VStack(spacing: 0) {
            HStack {
                Image(systemName: "magnifyingglass").foregroundStyle(.secondary)
                TextField("Search skills", text: $store.searchText)
                    .textFieldStyle(.plain)
            }
            .padding(8)
            Divider()

            if store.filteredCatalog.isEmpty {
                ContentUnavailableCompat(
                    title: store.catalog.isEmpty ? "No catalog data" : "No matches",
                    systemImage: "square.grid.2x2",
                    description: store.catalog.isEmpty
                        ? "Sync a registry to populate the catalog."
                        : "Try a different search term."
                )
            } else {
                List(store.filteredCatalog, selection: $selectedID) { result in
                    CatalogRow(result: result, installed: isInstalled(result))
                        .tag(result.id)
                }
                .listStyle(.inset)
            }
        }
    }

    @ViewBuilder
    private var detail: some View {
        if let result = store.filteredCatalog.first(where: { $0.id == selectedID }) {
            SkillDetailView(result: result, installed: isInstalled(result))
        } else {
            ContentUnavailableCompat(
                title: "Select a skill",
                systemImage: "hand.point.up.left",
                description: "Pick a skill from the list to see details and install it."
            )
        }
    }

    private func isInstalled(_ result: CatalogResult) -> Bool {
        store.installed.contains { $0.skill == result.skill.identity && $0.isGlobal }
    }
}

struct CatalogRow: View {
    let result: CatalogResult
    let installed: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            HStack {
                Text(result.skill.identity).font(.body.weight(.medium))
                if result.skill.featured {
                    Image(systemName: "star.fill").font(.caption2).foregroundStyle(.yellow)
                }
                Spacer()
                if installed {
                    Image(systemName: "checkmark.circle.fill").foregroundStyle(.green)
                }
            }
            Text(result.skill.description)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(2)
            HStack(spacing: 6) {
                TagChip(text: result.registry, color: .blue)
                TagChip(text: "v\(result.skill.version)", color: .gray)
                TagChip(text: result.skill.trust.level, color: .purple)
            }
        }
        .padding(.vertical, 2)
    }
}

struct SkillDetailView: View {
    @EnvironmentObject var store: AppStore
    let result: CatalogResult
    let installed: Bool

    private var skill: Skill { result.skill }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(skill.name).font(.title2.bold())
                        Text(skill.identity).font(.callout).foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button {
                        Task { await store.install(result) }
                    } label: {
                        Label(installed ? "Reinstall" : "Install", systemImage: "arrow.down.circle")
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(store.isBusy)
                }

                Text(skill.description).font(.body)

                FlowChips(items: skill.targets.map { ("Target: \($0)", Color.teal) }
                          + (skill.tags ?? []).map { ($0, Color.gray) })

                GroupBox {
                    Grid(alignment: .leading, horizontalSpacing: 16, verticalSpacing: 6) {
                        infoRow("Registry", result.registry)
                        infoRow("Version", skill.version)
                        infoRow("Namespace", skill.namespace)
                        infoRow("Trust", skill.trust.level)
                        if let reviewer = skill.trust.reviewer { infoRow("Reviewer", reviewer) }
                        infoRow("Source", skill.source.type)
                        if let url = skill.source.url, !url.isEmpty { infoRow("URL", url) }
                        if let ref = skill.source.ref, !ref.isEmpty { infoRow("Ref", ref) }
                        if let license = skill.license, !license.isEmpty { infoRow("License", license) }
                        if let maintainers = skill.maintainers, !maintainers.isEmpty {
                            infoRow("Maintainers", maintainers.joined(separator: ", "))
                        }
                        infoRow("Updated", skill.updatedAt)
                    }
                    .padding(4)
                }

                Text("Install command: skillhub install \(result.installSpec)")
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            .padding(20)
        }
    }

    @ViewBuilder
    private func infoRow(_ label: String, _ value: String) -> some View {
        GridRow {
            Text(label).foregroundStyle(.secondary).gridColumnAlignment(.trailing)
            Text(value).textSelection(.enabled)
        }
    }
}

struct TagChip: View {
    let text: String
    let color: Color

    var body: some View {
        Text(text)
            .font(.caption2)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(color.opacity(0.18), in: Capsule())
            .foregroundStyle(color)
    }
}

/// A simple wrapping row of chips.
struct FlowChips: View {
    let items: [(String, Color)]

    var body: some View {
        if items.isEmpty {
            EmptyView()
        } else {
            ViewThatFits(in: .horizontal) {
                HStack { chips }
                VStack(alignment: .leading) { chips }
            }
        }
    }

    private var chips: some View {
        ForEach(Array(items.enumerated()), id: \.offset) { _, item in
            TagChip(text: item.0, color: item.1)
        }
    }
}
