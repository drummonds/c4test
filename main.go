package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config types

type Branch struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

type Config struct {
	Title      string   `yaml:"title"`
	Commentary string   `yaml:"commentary"`
	MainPath   string   `yaml:"main_path"`
	Branches   []Branch `yaml:"branches"`
}

type RenderResult struct {
	SVG template.HTML
	Err string
}

type BranchResult struct {
	Branch
	RenderResult
}

type diagram struct {
	Name     string
	Source   string
	Main     RenderResult
	Branches []BranchResult
}

type pageData struct {
	Config   Config
	Diagrams []diagram
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(os.Getenv("HOME"), p[2:])
	}
	return p
}

func loadConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		// Default: single main binary, no branches
		return Config{
			Title:    "C4 Diagram Comparison: mmdg vs mermaid.js",
			MainPath: defaultMmdgPath(),
		}
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("parse %s: %v", path, err)
	}
	cfg.MainPath = expandPath(cfg.MainPath)
	if cfg.MainPath == "" {
		cfg.MainPath = defaultMmdgPath()
	}
	if cfg.Title == "" {
		cfg.Title = "C4 Diagram Comparison: mmdg vs mermaid.js"
	}
	for i := range cfg.Branches {
		cfg.Branches[i].Path = expandPath(cfg.Branches[i].Path)
	}
	return cfg
}

func defaultMmdgPath() string {
	if p := os.Getenv("MMDG_PATH"); p != "" {
		return p
	}
	return filepath.Join(os.Getenv("HOME"), "go", "bin", "mmdg")
}

func renderWith(binPath string, src []byte) (string, error) {
	cmd := exec.Command(binPath)
	cmd.Stdin = bytes.NewReader(src)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w: %s", err, ee.Stderr)
		}
		return "", err
	}
	return string(out), nil
}

func loadDiagrams(dir string, cfg Config) []diagram {
	files, err := filepath.Glob(filepath.Join(dir, "*.mmd"))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)

	var diagrams []diagram
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			log.Printf("skip %s: %v", f, err)
			continue
		}
		name := filepath.Base(f)
		d := diagram{Name: name, Source: string(src)}

		// Render with main binary
		if svg, err := renderWith(cfg.MainPath, src); err != nil {
			d.Main.Err = err.Error()
		} else {
			d.Main.SVG = template.HTML(svg)
		}

		// Render with each branch binary
		for _, b := range cfg.Branches {
			br := BranchResult{Branch: b}
			if svg, err := renderWith(b.Path, src); err != nil {
				br.Err = err.Error()
			} else {
				br.SVG = template.HTML(svg)
			}
			d.Branches = append(d.Branches, br)
		}

		diagrams = append(diagrams, d)
		log.Printf("loaded %s", name)
	}
	return diagrams
}

var tmpl = template.Must(template.New("page").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Config.Title}}</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 2rem; background: #fafafa; }
  h1 { margin-bottom: 0.5rem; }
  .commentary { color: #555; margin-bottom: 1rem; max-width: 80ch; line-height: 1.5; }
  .branch-list { margin-bottom: 1.5rem; }
  .branch-list dt { font-weight: 600; }
  .branch-list dd { margin: 0 0 0.5rem 1rem; color: #555; }
  .diagram { margin-bottom: 3rem; }
  .diagram h2 { margin-bottom: 0.5rem; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-bottom: 1rem; }
  .branches-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 1rem; }
  .col { background: #fff; border: 1px solid #ddd; border-radius: 4px; padding: 1rem; overflow: auto; }
  .col h3 { margin: 0 0 0.5rem; font-size: 0.9rem; color: #666; }
  .col svg { max-width: 100%; height: auto; border: 1px solid #e0e0e0; background: #fafafa; }
  .source { background: #f5f5f5; border: 1px solid #ddd; border-radius: 4px; padding: 1rem;
            overflow: auto; font-family: monospace; font-size: 0.82rem; line-height: 1.4; margin-bottom: 1rem; }
  .error { background: #fee; border-color: #c00; color: #900; padding: 1rem; border-radius: 4px;
           white-space: pre-wrap; font-family: monospace; font-size: 0.85rem; }
</style>
</head>
<body>
<h1>{{.Config.Title}}</h1>
<p><a href="index.html">&larr; Home</a> | <a href="ecosystem.html">Ecosystem</a></p>
{{- if .Config.Commentary}}
<div class="commentary">{{.Config.Commentary}}</div>
{{- end}}
{{- if .Config.Branches}}
<dl class="branch-list">
{{- range .Config.Branches}}
  <dt>{{.Name}}</dt>
  <dd>{{.Description}} <code>{{.Path}}</code></dd>
{{- end}}
</dl>
{{- end}}
{{range .Diagrams}}
<div class="diagram">
  <h2>{{.Name}}</h2>
  <pre class="source">{{.Source}}</pre>
  <div class="grid">
    <div class="col">
      <h3>mmdg (Go)</h3>
      {{if .Main.Err}}<div class="error">{{.Main.Err}}</div>{{else}}{{.Main.SVG}}{{end}}
    </div>
    <div class="col">
      <h3>mermaid.js (browser)</h3>
      <pre class="mermaid">{{.Source}}</pre>
    </div>
  </div>
  {{- if .Branches}}
  <div class="branches-grid">
    {{- range .Branches}}
    <div class="col">
      <h3>{{.Name}}</h3>
      {{if .Err}}<div class="error">{{.Err}}</div>{{else}}{{.SVG}}{{end}}
    </div>
    {{- end}}
  </div>
  {{- end}}
</div>
{{end}}
<script type="module">
  import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
  mermaid.initialize({ startOnLoad: true, securityLevel: 'loose' });
</script>
</body>
</html>`))

func main() {
	dir := "diagrams"
	outDir := "docs"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		outDir = os.Args[2]
	}

	cfg := loadConfig("c4test.yml")
	diagrams := loadDiagrams(dir, cfg)
	log.Printf("rendered %d diagrams", len(diagrams))

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	data := pageData{Config: cfg, Diagrams: diagrams}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatal(err)
	}

	outPath := filepath.Join(outDir, "comparison.html")
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", outPath)
}
