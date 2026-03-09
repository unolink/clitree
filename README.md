[Русская версия (README.ru.md)](README.ru.md)

# clitree

Lightweight, table-driven CLI framework for Go with zero dependencies.

## Features

- **Declarative** — define commands as data structures, not method chains
- **Nested subcommands** — unlimited depth with automatic path resolution
- **Auto-generated help** — formatted from command metadata
- **Aliases** — alternative names for commands (e.g., `-v` for `version`)
- **Hidden commands** — deprecated commands invisible in help but still executable
- **Testable** — injectable stdout/stderr writers
- **Zero dependencies** — only Go standard library

## Install

```bash
go get github.com/unolink/clitree
```

## Usage

```go
root := &clitree.Command{
    Name:  "myapp",
    Short: "My CLI application",
    Subcommands: []*clitree.Command{
        {
            Name:    "version",
            Aliases: []string{"-v", "--version"},
            Short:   "Show version",
            Run: func(ctx context.Context, fs *flag.FlagSet) error {
                fmt.Println("v1.0.0")
                return nil
            },
        },
        {
            Name:  "serve",
            Short: "Start server",
            Flags: func(fs *flag.FlagSet) {
                fs.String("addr", ":8080", "Listen address")
            },
            Run: func(ctx context.Context, fs *flag.FlagSet) error {
                addr := fs.Lookup("addr").Value.String()
                fmt.Printf("Listening on %s\n", addr)
                return nil
            },
        },
    },
}

tree := clitree.New(root, os.Stdout, os.Stderr)
os.Exit(tree.Execute(context.Background(), os.Args[1:]))
```

## Comparison with Cobra

| Feature              | clitree | cobra |
|----------------------|---------|-------|
| Subcommands          | yes     | yes   |
| Auto help            | yes     | yes   |
| Aliases              | yes     | yes   |
| Flag parsing         | yes     | yes   |
| Dependencies         | 0       | 4+    |
| Binary size overhead | ~0 KB   | ~2 MB |

## License

MIT — see [LICENSE](LICENSE).
