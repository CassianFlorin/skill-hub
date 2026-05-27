package cli

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cassian/skill-hub/internal/config"
	"github.com/cassian/skill-hub/internal/deploy"
	"github.com/cassian/skill-hub/internal/install"
	"github.com/cassian/skill-hub/internal/registry"
)

var Version = "dev"

func Run(args []string, stdout io.Writer, stderr io.Writer, workDir string) error {
	if len(args) == 0 {
		return usage(stderr)
	}
	switch args[0] {
	case "version":
		return runVersion(stdout)
	case "doctor":
		return runDoctor(stdout, workDir)
	case "init":
		return runInit(stdout, workDir)
	case "registry":
		return runRegistry(args[1:], stdout, workDir)
	case "catalog":
		return runCatalog(args[1:], stdout, workDir)
	case "search":
		return runSearch(args[1:], stdout, workDir)
	case "info":
		return runInfo(args[1:], stdout, workDir)
	case "install":
		if len(args) != 2 {
			return fmt.Errorf("usage: skillhub install <path|registry/skill>")
		}
		locked, err := install.Install(workDir, args[1])
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "installed %s@%s\n", locked.DisplayIdentity(), locked.Version)
		return nil
	case "rollback":
		return runRollback(args[1:], stdout)
	case "uninstall":
		return runUninstall(args[1:], stdout)
	case "list":
		return runList(stdout)
	case "update":
		return runUpdate(stdout)
	case "deploy":
		return runDeploy(args[1:], stdout)
	default:
		return usage(stderr)
	}
}

func runVersion(stdout io.Writer) error {
	_, _ = fmt.Fprintf(stdout, "skillhub %s\n", Version)
	return nil
}

func runDoctor(stdout io.Writer, workDir string) error {
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	home, err := config.DefaultHome()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, "config: ok")
	_, _ = fmt.Fprintf(stdout, "config.path: %s\n", config.Path(workDir))
	_, _ = fmt.Fprintf(stdout, "home: %s\n", home)
	_, _ = fmt.Fprintf(stdout, "install_dir: %s\n", cfg.InstallDir)
	for _, runtime := range deploy.SupportedRuntimes() {
		dir, err := deploy.RuntimeDir(runtime.Name)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "runtime %s: %s\n", runtime.Name, dir)
	}
	_, _ = fmt.Fprintf(stdout, "registries: %d\n", len(cfg.Registries))
	for _, status := range registry.ListRegistries(cfg) {
		_, _ = fmt.Fprintf(stdout, "registry %s: %s %s skills=%d generated_at=%s\n", status.Name, status.Type, status.Location, status.SkillCount, status.GeneratedAt)
	}
	lock, err := install.LoadLock()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "installed: %d\n", len(lock.Skills))
	return nil
}

func runUninstall(args []string, stdout io.Writer) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub uninstall <identity> [--deployed]")
	}
	removeDeployed := false
	if len(args) == 2 {
		if args[1] != "--deployed" {
			return fmt.Errorf("usage: skillhub uninstall <identity> [--deployed]")
		}
		removeDeployed = true
	}
	locked, err := install.Uninstall(args[0])
	if err != nil {
		return err
	}
	if removeDeployed {
		for _, runtime := range []string{"codex", "claude"} {
			target, err := deploy.RuntimeTarget(runtime, locked.DisplayIdentity())
			if err != nil {
				return err
			}
			if err := os.RemoveAll(target); err != nil {
				return err
			}
		}
	}
	_, _ = fmt.Fprintf(stdout, "uninstalled %s\n", locked.DisplayIdentity())
	return nil
}

func runRollback(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub rollback <identity>")
	}
	locked, err := install.Rollback(args[0])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "rolled back %s to %s\n", locked.DisplayIdentity(), locked.Version)
	return nil
}

func runInit(stdout io.Writer, workDir string) error {
	cfg, err := config.NewDefault()
	if err != nil {
		return err
	}
	if err := config.Save(workDir, cfg); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "initialized %s\n", config.Path(workDir))
	return nil
}

func runRegistry(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skillhub registry <add|list|sync|index>")
	}
	switch args[0] {
	case "add":
		return runRegistryAdd(args[1:], stdout, workDir)
	case "list":
		return runRegistryList(stdout, workDir)
	case "sync":
		return runRegistrySync(args[1:], stdout, workDir)
	case "index":
		return runRegistryIndex(args[1:], stdout, workDir)
	default:
		return fmt.Errorf("usage: skillhub registry <add|list|sync|index>")
	}
}

func runRegistryAdd(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: skillhub registry add <local|git> <name> <path-or-url>")
	}
	registryType, name, location := args[0], args[1], args[2]
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	switch registryType {
	case "local":
		if !filepath.IsAbs(location) {
			location = filepath.Join(workDir, location)
		}
		abs, err := filepath.Abs(location)
		if err != nil {
			return err
		}
		cfg.Registries[name] = config.Registry{Type: "local", Path: abs}
	case "git":
		cfg.Registries[name] = config.Registry{Type: "git", URL: location}
	default:
		return fmt.Errorf("unsupported registry type %q", registryType)
	}
	if err := config.Save(workDir, cfg); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "registered %s\n", name)
	return nil
}

func runRegistryIndex(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: skillhub registry index <generate|validate> <registry>")
	}
	action, name := args[0], args[1]
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	reg, ok := cfg.Registries[name]
	if !ok {
		return fmt.Errorf("unknown registry %q", name)
	}
	switch action {
	case "generate":
		index, _, err := registry.GenerateIndex(name, reg)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "indexed %s with %d skills\n", name, len(index.Skills))
	case "validate":
		count, err := registry.ValidateIndex(name, reg)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "validated %s with %d skills\n", name, count)
	default:
		return fmt.Errorf("usage: skillhub registry index <generate|validate> <registry>")
	}
	return nil
}

func runRegistryList(stdout io.Writer, workDir string) error {
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	for _, status := range registry.ListRegistries(cfg) {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\t%d\t%s\n", status.Name, status.Type, status.Location, status.SkillCount, status.GeneratedAt)
	}
	return nil
}

func runRegistrySync(args []string, stdout io.Writer, workDir string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: skillhub registry sync <registry|--all>")
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	if args[0] == "--all" {
		for _, status := range registry.ListRegistries(cfg) {
			reg := cfg.Registries[status.Name]
			count, err := registry.SyncRegistry(status.Name, reg)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(stdout, "synced %s with %d skills\n", status.Name, count)
		}
		return nil
	}
	reg, ok := cfg.Registries[args[0]]
	if !ok {
		return fmt.Errorf("unknown registry %q", args[0])
	}
	count, err := registry.SyncRegistry(args[0], reg)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "synced %s with %d skills\n", args[0], count)
	return nil
}

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
	for _, result := range results {
		_, _ = fmt.Fprintln(stdout, formatCatalogResult(result))
	}
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
	for _, count := range counts {
		_, _ = fmt.Fprintf(stdout, "%s\t%d\n", count.Name, count.Count)
	}
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

func runSearch(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub search <query> [--json]")
	}
	query := args[0]
	jsonOutput := false
	if len(args) == 2 {
		if args[1] != "--json" {
			return fmt.Errorf("usage: skillhub search <query> [--json]")
		}
		jsonOutput = true
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	results, err := registry.SearchIndexes(cfg, query)
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, catalogJSONResults(results))
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills found")
		return nil
	}
	for _, result := range results {
		_, _ = fmt.Fprintln(stdout, formatCatalogResult(result))
	}
	return nil
}

func runInfo(args []string, stdout io.Writer, workDir string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: skillhub info <registry/identity|identity> [--json]")
	}
	spec := args[0]
	jsonOutput := false
	if len(args) == 2 {
		if args[1] != "--json" {
			return fmt.Errorf("usage: skillhub info <registry/identity|identity> [--json]")
		}
		jsonOutput = true
	}
	cfg, err := config.Load(workDir)
	if err != nil {
		return err
	}
	result, ok, err := registry.FindIndexedSkill(cfg, spec)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("skill %q not found", spec)
	}
	indexed := result.Skill
	installCommand := fmt.Sprintf("skillhub install %s/%s", result.Registry, indexed.Identity)
	if jsonOutput {
		return writeJSON(stdout, infoJSONResult{
			Registry:       result.Registry,
			Skill:          indexed,
			InstallCommand: installCommand,
		})
	}
	_, _ = fmt.Fprintf(stdout, "identity: %s\n", indexed.Identity)
	_, _ = fmt.Fprintf(stdout, "registry: %s\n", result.Registry)
	_, _ = fmt.Fprintf(stdout, "name: %s\n", indexed.Name)
	_, _ = fmt.Fprintf(stdout, "namespace: %s\n", indexed.Namespace)
	_, _ = fmt.Fprintf(stdout, "version: %s\n", indexed.Version)
	_, _ = fmt.Fprintf(stdout, "description: %s\n", indexed.Description)
	_, _ = fmt.Fprintf(stdout, "targets: %s\n", strings.Join(indexed.Targets, ", "))
	_, _ = fmt.Fprintf(stdout, "tags: %s\n", strings.Join(indexed.Tags, ", "))
	_, _ = fmt.Fprintf(stdout, "source.type: %s\n", indexed.Source.Type)
	_, _ = fmt.Fprintf(stdout, "source.url: %s\n", indexed.Source.URL)
	_, _ = fmt.Fprintf(stdout, "source.path: %s\n", indexed.Source.Path)
	_, _ = fmt.Fprintf(stdout, "source.ref: %s\n", indexed.Source.Ref)
	_, _ = fmt.Fprintf(stdout, "maintainers: %s\n", strings.Join(indexed.Maintainers, ", "))
	_, _ = fmt.Fprintf(stdout, "license: %s\n", indexed.License)
	_, _ = fmt.Fprintf(stdout, "trust: %s\n", indexed.Trust.Level)
	_, _ = fmt.Fprintf(stdout, "trust.reviewed_at: %s\n", indexed.Trust.ReviewedAt)
	_, _ = fmt.Fprintf(stdout, "trust.reviewer: %s\n", indexed.Trust.Reviewer)
	_, _ = fmt.Fprintf(stdout, "featured: %t\n", indexed.Featured)
	_, _ = fmt.Fprintf(stdout, "updated_at: %s\n", indexed.UpdatedAt)
	_, _ = fmt.Fprintf(stdout, "checksum: %s\n", indexed.Checksum)
	_, _ = fmt.Fprintf(stdout, "install: %s\n", installCommand)
	return nil
}

func featuredLabel(featured bool) string {
	if featured {
		return "featured"
	}
	return "-"
}

func formatCatalogResult(result registry.SearchResult) string {
	return fmt.Sprintf("%s/%s\t%s\t%s\t%s\t%s\t%s",
		result.Registry,
		result.Skill.Identity,
		result.Skill.Version,
		strings.Join(result.Skill.Targets, ","),
		result.Skill.Trust.Level,
		featuredLabel(result.Skill.Featured),
		result.Skill.Description,
	)
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

func runList(stdout io.Writer) error {
	lock, err := install.LoadLock()
	if err != nil {
		return err
	}
	if len(lock.Skills) == 0 {
		_, _ = fmt.Fprintln(stdout, "no skills installed")
		return nil
	}
	for _, locked := range lock.Skills {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", locked.DisplayIdentity(), locked.Version, locked.Description)
	}
	return nil
}

func runUpdate(stdout io.Writer) error {
	changes, err := install.UpdateAll()
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(stdout, "all skills are current")
		return nil
	}
	for _, change := range changes {
		_, _ = fmt.Fprintf(stdout, "updated %s %s -> %s\n", change[0], change[1], change[2])
	}
	return nil
}

func runDeploy(args []string, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: skillhub deploy <codex|claude> [identity] [--dry-run] [--force]")
	}
	runtime := args[0]
	if runtime == "status" {
		return runDeployStatus(args[1:], stdout)
	}
	options := deploy.Options{}
	for _, arg := range args[1:] {
		switch arg {
		case "--dry-run":
			options.DryRun = true
		case "--force":
			options.Force = true
		default:
			if options.Identity != "" {
				return fmt.Errorf("usage: skillhub deploy codex [identity] [--dry-run] [--force]")
			}
			options.Identity = arg
		}
	}
	var deployed []deploy.Result
	var err error
	switch runtime {
	case "codex":
		deployed, err = deploy.DeployCodex(options)
	case "claude":
		deployed, err = deploy.DeployClaude(options)
	default:
		return fmt.Errorf("usage: skillhub deploy <codex|claude> [identity] [--dry-run] [--force]")
	}
	if err != nil {
		return err
	}
	for _, result := range deployed {
		_, _ = fmt.Fprintln(stdout, deployResultLine(result, runtime))
	}
	return nil
}

func deployResultLine(result deploy.Result, runtime string) string {
	if result.Runtime != "" {
		runtime = result.Runtime
	}
	switch result.State {
	case deploy.StateSkipped:
		return fmt.Sprintf("skipped %s to %s: %s", result.Identity, runtime, result.Reason)
	case deploy.StateConflict:
		return fmt.Sprintf("conflict %s to %s: %s", result.Identity, runtime, result.Reason)
	case deploy.StateWouldDeploy:
		return fmt.Sprintf("would deploy %s to %s", result.Identity, runtime)
	default:
		if result.DryRun {
			return fmt.Sprintf("would deploy %s to %s", result.Identity, runtime)
		}
		return fmt.Sprintf("deployed %s to %s", result.Identity, runtime)
	}
}

func runDeployStatus(args []string, stdout io.Writer) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: skillhub deploy status [codex|claude]")
	}
	runtime := ""
	if len(args) == 1 {
		runtime = args[0]
	}
	statuses, err := deploy.Statuses(runtime)
	if err != nil {
		return err
	}
	for _, status := range statuses {
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", status.Identity, status.Runtime, status.State)
	}
	return nil
}

func usage(stderr io.Writer) error {
	_, _ = fmt.Fprintln(stderr, "usage: skillhub <version|doctor|init|registry|catalog|search|info|install|rollback|uninstall|list|update|deploy>")
	return fmt.Errorf("invalid command")
}
