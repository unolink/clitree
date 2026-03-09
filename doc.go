// Package clitree provides a lightweight, table-driven approach to building CLI applications.
//
// clitree allows you to define your entire command structure declaratively using nested
// Command definitions. It automatically generates help text, supports subcommands of any depth,
// and requires only the Go standard library.
//
// # Philosophy
//
// - Declarative: Define commands as data structures
// - Zero dependencies: Only Go stdlib (flag, context, io)
// - Automatic help: Generated from command metadata
// - Testable: Easy to mock stdout/stderr
// - Extensible: Add commands without changing existing code
//
// # Quick Start
//
//	root := &clitree.Command{
//	    Name:  "myapp",
//	    Short: "My CLI application",
//	    Subcommands: []*clitree.Command{
//	        {
//	            Name:  "version",
//	            Short: "Show version",
//	            Run: func(ctx context.Context, fs *flag.FlagSet) error {
//	                fmt.Println("v1.0.0")
//	                return nil
//	            },
//	        },
//	    },
//	}
//
//	tree := clitree.New(root, os.Stdout, os.Stderr)
//	os.Exit(tree.Execute(context.Background(), os.Args[1:]))
//
// # Features
//
// - Nested subcommands (unlimited depth)
// - Command aliases
// - Automatic help generation
// - Per-command flag sets
// - Context support for cancellation
// - Examples and long descriptions
// - Table-formatted output
//
// # Comparison with Cobra
//
// clitree provides similar functionality to spf13/cobra but with zero dependencies:
//
//	| Feature              | clitree | cobra |
//	|----------------------|---------|-------|
//	| Subcommands          | ✅      | ✅    |
//	| Auto help            | ✅      | ✅    |
//	| Aliases              | ✅      | ✅    |
//	| Flag parsing         | ✅      | ✅    |
//	| Dependencies         | 0       | 4+    |
//	| Binary size overhead | ~0 KB   | ~2 MB |
//
// # Design
//
// Commands are defined as a tree structure using the Command type. Each command can have:
//   - Metadata (name, description, examples)
//   - Flags (via flag.FlagSet)
//   - A Run function
//   - Subcommands (nested Commands)
//
// The Tree type manages command lookup, help generation, and execution.
package clitree
