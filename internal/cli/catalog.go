package cli

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/registry"
)

func runCatalog(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skillhub catalog <list|featured|tags|targets|namespaces|trust|export>")
	}
	switch args[0] {
	case "list":
		return runCatalogList(args[1:], stdout, workDir, false)
	case "featured":
		return runCatalogList(args[1:], stdout, workDir, true)
	case "export":
		return runCatalogExport(args[1:], stdout, workDir)
	case "tags":
		return runCatalogAggregate(args[1:], stdout, workDir, "tags")
	case "targets":
		return runCatalogAggregate(args[1:], stdout, workDir, "targets")
	case "namespaces":
		return runCatalogAggregate(args[1:], stdout, workDir, "namespaces")
	case "trust":
		return runCatalogAggregate(args[1:], stdout, workDir, "trust")
	default:
		return fmt.Errorf("usage: skillhub catalog <list|featured|tags|targets|namespaces|trust|export>")
	}
}

func runCatalogList(args []string, stdout io.Writer, workDir string, featuredOnly bool) error {
	filter := registry.CatalogFilter{}
	jsonOutput := false
	if featuredOnly {
		featured := true
		filter.Featured = &featured
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--registry":
			i++
			if i >= len(args) {
				return fmt.Errorf("--registry requires a value")
			}
			filter.Registry = args[i]
		case "--target":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target requires a value")
			}
			filter.Target = args[i]
		case "--tag":
			i++
			if i >= len(args) {
				return fmt.Errorf("--tag requires a value")
			}
			filter.Tag = args[i]
		case "--namespace":
			i++
			if i >= len(args) {
				return fmt.Errorf("--namespace requires a value")
			}
			filter.Namespace = args[i]
		case "--trust":
			i++
			if i >= len(args) {
				return fmt.Errorf("--trust requires a value")
			}
			filter.Trust = args[i]
		case "--featured":
			featured := true
			filter.Featured = &featured
		case "--official":
			filter.Official = true
		default:
			return fmt.Errorf("unknown catalog option %q", args[i])
		}
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.ListCatalog(cfg, filter)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, catalogJSONResults(results))
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no catalog skills found")
		return nil
	}
	_, _ = fmt.Fprint(stdout, formatCatalogResults(results))
	return nil
}

func runCatalogAggregate(args []string, stdout io.Writer, workDir string, kind string) error {
	filter := registry.CatalogFilter{}
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--registry":
			i++
			if i >= len(args) {
				return fmt.Errorf("--registry requires a value")
			}
			filter.Registry = args[i]
		default:
			return fmt.Errorf("unknown catalog option %q", args[i])
		}
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.ListCatalog(cfg, filter)
	if err != nil {
		return err
	}
	counts := aggregateCatalog(results, kind)
	if jsonOutput {
		return writeJSON(stdout, counts)
	}
	if len(counts) == 0 {
		_, _ = fmt.Fprintf(stdout, "no catalog %s found\n", kind)
		return nil
	}
	_, _ = fmt.Fprint(stdout, formatCatalogCounts(kind, counts))
	return nil
}

func runCatalogExport(args []string, stdout io.Writer, workDir string) error {
	filter := registry.CatalogFilter{}
	outputDir := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--registry":
			i++
			if i >= len(args) {
				return fmt.Errorf("--registry requires a value")
			}
			filter.Registry = args[i]
		case "--target":
			i++
			if i >= len(args) {
				return fmt.Errorf("--target requires a value")
			}
			filter.Target = args[i]
		case "--tag":
			i++
			if i >= len(args) {
				return fmt.Errorf("--tag requires a value")
			}
			filter.Tag = args[i]
		case "--namespace":
			i++
			if i >= len(args) {
				return fmt.Errorf("--namespace requires a value")
			}
			filter.Namespace = args[i]
		case "--trust":
			i++
			if i >= len(args) {
				return fmt.Errorf("--trust requires a value")
			}
			filter.Trust = args[i]
		case "--featured":
			featured := true
			filter.Featured = &featured
		case "--official":
			filter.Official = true
		case "--output":
			i++
			if i >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			outputDir = args[i]
		default:
			return fmt.Errorf("unknown catalog export option %q", args[i])
		}
	}
	if outputDir == "" {
		return fmt.Errorf("--output requires a value")
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.ListCatalog(cfg, filter)
	if err != nil {
		return err
	}
	if err := writeCatalogExport(outputDir, results); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "exported %d catalog skills to %s\n", len(results), outputDir)
	return nil
}

type catalogJSONResult struct {
	Registry string              `json:"registry"`
	Skill    registry.IndexSkill `json:"skill"`
}

type catalogExport struct {
	GeneratedBy string               `json:"generated_by"`
	Skills      []catalogExportSkill `json:"skills"`
	Tags        []catalogCount       `json:"tags"`
	Targets     []catalogCount       `json:"targets"`
	Namespaces  []catalogCount       `json:"namespaces"`
	Trust       []catalogCount       `json:"trust"`
}

type catalogExportSkill struct {
	Registry       string              `json:"registry"`
	Skill          registry.IndexSkill `json:"skill"`
	InstallCommand string              `json:"install_command"`
}

type infoJSONResult struct {
	Registry       string              `json:"registry"`
	Skill          registry.IndexSkill `json:"skill"`
	InstallCommand string              `json:"install_command"`
}

type catalogCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func catalogJSONResults(results []registry.SearchResult) []catalogJSONResult {
	jsonResults := make([]catalogJSONResult, 0, len(results))
	for _, result := range results {
		jsonResults = append(jsonResults, catalogJSONResult{Registry: result.Registry, Skill: result.Skill})
	}
	return jsonResults
}

func catalogExportResults(results []registry.SearchResult) catalogExport {
	skills := make([]catalogExportSkill, 0, len(results))
	for _, result := range results {
		skills = append(skills, catalogExportSkill{
			Registry:       result.Registry,
			Skill:          result.Skill,
			InstallCommand: fmt.Sprintf("skillhub install %s/%s", result.Registry, result.Skill.Identity),
		})
	}
	return catalogExport{
		GeneratedBy: "skillhub",
		Skills:      skills,
		Tags:        aggregateCatalog(results, "tags"),
		Targets:     aggregateCatalog(results, "targets"),
		Namespaces:  aggregateCatalog(results, "namespaces"),
		Trust:       aggregateCatalog(results, "trust"),
	}
}

func aggregateCatalog(results []registry.SearchResult, kind string) []catalogCount {
	countMap := map[string]int{}
	for _, result := range results {
		values := result.Skill.Tags
		switch kind {
		case "targets":
			values = result.Skill.Targets
		case "namespaces":
			values = []string{result.Skill.Namespace}
		case "trust":
			values = []string{result.Skill.Trust.Level}
		}
		for _, value := range values {
			countMap[value]++
		}
	}
	counts := make([]catalogCount, 0, len(countMap))
	for name, count := range countMap {
		counts = append(counts, catalogCount{Name: name, Count: count})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Name < counts[j].Name
	})
	return counts
}

func writeCatalogExport(outputDir string, results []registry.SearchResult) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	data := catalogExportResults(results)
	jsonFile, err := os.Create(filepath.Join(outputDir, "catalog.json"))
	if err != nil {
		return err
	}
	if err := writeJSON(jsonFile, data); err != nil {
		_ = jsonFile.Close()
		return err
	}
	if err := jsonFile.Close(); err != nil {
		return err
	}
	htmlFile, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		return err
	}
	if err := catalogHTMLTemplate.Execute(htmlFile, data); err != nil {
		_ = htmlFile.Close()
		return err
	}
	return htmlFile.Close()
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

var catalogHTMLTemplate = template.Must(template.New("catalog").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>skill-hub catalog</title>
  <style>
    body { font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 2rem; color: #1f2937; }
    header { border-bottom: 1px solid #d1d5db; margin-bottom: 1.5rem; padding-bottom: 1rem; }
    h1 { margin: 0 0 .25rem; font-size: 1.75rem; }
    section { margin: 1.5rem 0; }
    table { border-collapse: collapse; width: 100%; }
    th, td { border-bottom: 1px solid #e5e7eb; padding: .7rem; text-align: left; vertical-align: top; }
    th { background: #f9fafb; font-weight: 600; }
    code { background: #f3f4f6; border-radius: 4px; padding: .15rem .3rem; }
    .meta { color: #4b5563; }
  </style>
</head>
<body>
  <header>
    <h1>skill-hub catalog</h1>
    <p class="meta">{{len .Skills}} skills exported by {{.GeneratedBy}}</p>
  </header>
  <section>
    <h2>Skills</h2>
    <table>
      <thead>
        <tr><th>Identity</th><th>Targets</th><th>Trust</th><th>Tags</th><th>Install</th></tr>
      </thead>
      <tbody>
      {{range .Skills}}
        <tr>
          <td><strong>{{.Registry}}/{{.Skill.Identity}}</strong><br>{{.Skill.Description}}</td>
          <td>{{range $i, $target := .Skill.Targets}}{{if $i}}, {{end}}{{$target}}{{end}}</td>
          <td>{{.Skill.Trust.Level}}</td>
          <td>{{range $i, $tag := .Skill.Tags}}{{if $i}}, {{end}}{{$tag}}{{end}}</td>
          <td><code>{{.InstallCommand}}</code></td>
        </tr>
      {{end}}
      </tbody>
    </table>
  </section>
</body>
</html>
`))
