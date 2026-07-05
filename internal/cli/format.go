package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CassianFlorin/skill-hub/internal/registry"
)

func featuredLabel(featured bool) string {
	if featured {
		return "featured"
	}
	return "-"
}

func formatCatalogResults(results []registry.SearchResult) string {
	rows := make([][]string, 0, len(results))
	for _, result := range results {
		rows = append(rows, []string{
			fmt.Sprintf("%s/%s", result.Registry, result.Skill.Identity),
			result.Skill.Version,
			strings.Join(result.Skill.Targets, ","),
			result.Skill.Trust.Level,
			featuredLabel(result.Skill.Featured),
			result.Skill.Description,
		})
	}
	return formatTable([]string{"Skill", "Version", "Targets", "Trust", "Featured", "Description"}, rows)
}

func formatCatalogCounts(kind string, counts []catalogCount) string {
	nameHeader := "Name"
	switch kind {
	case "tags":
		nameHeader = "Tag"
	case "targets":
		nameHeader = "Target"
	case "namespaces":
		nameHeader = "Namespace"
	case "trust":
		nameHeader = "Trust"
	}
	rows := make([][]string, 0, len(counts))
	for _, count := range counts {
		rows = append(rows, []string{count.Name, strconv.Itoa(count.Count)})
	}
	return formatTable([]string{nameHeader, "Count"}, rows)
}

func formatRegistryList(statuses []registry.RegistryStatus) string {
	rows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		rows = append(rows, []string{
			status.Name,
			status.Type,
			status.Location,
			strconv.Itoa(status.SkillCount),
			status.GeneratedAt,
		})
	}
	return formatTable([]string{"Name", "Type", "Location", "Skills", "Generated At"}, rows)
}

func formatRequires(requires map[string]string) string {
	if len(requires) == 0 {
		return ""
	}
	components := make([]string, 0, len(requires))
	for component := range requires {
		components = append(components, component)
	}
	sort.Strings(components)
	pairs := make([]string, 0, len(components))
	for _, component := range components {
		pairs = append(pairs, component+" "+requires[component])
	}
	return strings.Join(pairs, ", ")
}

func formatInfoResult(result registry.SearchResult, installCommand string) string {
	indexed := result.Skill
	rows := [][]string{
		{"identity", indexed.Identity},
		{"registry", result.Registry},
		{"name", indexed.Name},
		{"namespace", indexed.Namespace},
		{"version", indexed.Version},
		{"description", indexed.Description},
		{"targets", strings.Join(indexed.Targets, ", ")},
		{"tags", strings.Join(indexed.Tags, ", ")},
		{"requires", formatRequires(indexed.Requires)},
		{"source.type", indexed.Source.Type},
		{"source.url", indexed.Source.URL},
		{"source.path", indexed.Source.Path},
		{"source.ref", indexed.Source.Ref},
		{"maintainers", strings.Join(indexed.Maintainers, ", ")},
		{"license", indexed.License},
		{"trust", indexed.Trust.Level},
		{"trust.reviewed_at", indexed.Trust.ReviewedAt},
		{"trust.reviewer", indexed.Trust.Reviewer},
		{"featured", fmt.Sprintf("%t", indexed.Featured)},
		{"updated_at", indexed.UpdatedAt},
		{"checksum", indexed.Checksum},
		{"install", installCommand},
	}
	return formatTable([]string{"Field", "Value"}, rows)
}
