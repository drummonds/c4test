# c4test

Side-by-side comparison of C4 diagrams rendered by mmdg (pure Go) vs mmdc (JS reference).

## Prerequisites

- Go 1.25+
- mmdg: `go install github.com/nicholasgasior/mmdg@latest`
- mmdc: `task install-mmdc` or `npm install -g @mermaid-js/mermaid-cli`

## Usage

Place `.mmd` files in `diagrams/`, then:

```
task run
```

Open http://localhost:8080
