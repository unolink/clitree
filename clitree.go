package clitree

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

// RunFunc is the function signature for command execution.
// It receives a context (for cancellation), a parsed FlagSet, and returns an error.
type RunFunc func(ctx context.Context, fs *flag.FlagSet) error

// FlagsFunc is the function signature for setting up command flags.
// It receives a FlagSet to configure.
type FlagsFunc func(fs *flag.FlagSet)

// Command defines a CLI command declaratively.
type Command struct {
	// Name is the command name (e.g., "run", "capture")
	Name string

	// Aliases are alternative names for this command (e.g., ["-v", "--version"])
	Aliases []string

	// Short is a one-line description shown in command lists
	Short string

	// Long is a detailed description shown in command help
	Long string

	// Examples are usage examples shown in help
	Examples []string

	// Flags is a function that configures this command's flags
	Flags FlagsFunc

	// Run is the function executed when this command is invoked
	Run RunFunc

	// Subcommands are nested commands under this command
	Subcommands []*Command

	// Hidden prevents the command from appearing in help (useful for deprecated commands)
	Hidden bool
}

// Tree represents a command tree with fast lookup capabilities.
type Tree struct {
	root   *Command
	index  map[string]*Command // Fast lookup: "capture run" -> Command
	stdout io.Writer
	stderr io.Writer
}

// New creates a new command tree from a root Command.
//
// The stdout and stderr writers are used for help output and error messages.
// Typically these are os.Stdout and os.Stderr, but can be replaced for testing.
func New(root *Command, stdout, stderr io.Writer) *Tree {
	tree := &Tree{
		root:   root,
		index:  make(map[string]*Command),
		stdout: stdout,
		stderr: stderr,
	}
	tree.buildIndex(root, "")
	return tree
}

// buildIndex recursively builds an index of all commands for fast lookup.
// The index maps command paths like "capture run" to their Command definitions.
func (t *Tree) buildIndex(cmd *Command, prefix string) {
	// Build full path
	var fullPath string
	var relativePath string

	if prefix == "" {
		// Root command
		fullPath = cmd.Name
		relativePath = ""
	} else {
		// Subcommand
		fullPath = prefix + " " + cmd.Name

		// Extract relative path (without root name)
		// Example: "app config get" -> "config get"
		parts := strings.Split(fullPath, " ")
		if len(parts) > 1 {
			relativePath = strings.Join(parts[1:], " ")
		}
	}

	// Index by full path (e.g., "app config")
	// Full paths should always be unique, so we can safely overwrite
	t.index[fullPath] = cmd

	// Index by relative path (e.g., "config") for Find() method
	// Don't overwrite if key already exists to avoid collisions
	if relativePath != "" {
		if _, exists := t.index[relativePath]; !exists {
			t.index[relativePath] = cmd
		}
	}

	// Index by aliases (only direct aliases, not with path)
	// Don't overwrite if alias already exists to avoid collisions
	for _, alias := range cmd.Aliases {
		if _, exists := t.index[alias]; !exists {
			t.index[alias] = cmd
		}
	}

	// Recursively index subcommands
	for _, sub := range cmd.Subcommands {
		t.buildIndex(sub, fullPath)
	}
}

// Execute executes a command based on the provided arguments.
//
// It finds the appropriate command, parses its flags, and calls its Run function.
// Returns an exit code suitable for os.Exit().
//
// Example:
//
//	tree := clitree.New(root, os.Stdout, os.Stderr)
//	exitCode := tree.Execute(context.Background(), os.Args[1:])
//	os.Exit(exitCode)
func (t *Tree) Execute(ctx context.Context, args []string) int {
	// No arguments - show root help
	if len(args) == 0 {
		t.printHelp(t.root, "")
		return 1
	}

	// Find the deepest matching command
	cmd, remainingArgs := t.findCommand(args)

	// Handle help flag
	if len(remainingArgs) > 0 {
		firstArg := remainingArgs[0]
		if firstArg == "help" || firstArg == "-h" || firstArg == "--help" {
			path := strings.Join(args[:len(args)-len(remainingArgs)], " ")
			t.printHelp(cmd, path)
			return 0
		}
	}

	// If command has subcommands but no args, show help
	if len(cmd.Subcommands) > 0 && len(remainingArgs) == 0 {
		path := strings.Join(args[:len(args)-len(remainingArgs)], " ")
		t.printHelp(cmd, path)
		return 0
	}

	// If command has no Run function but has subcommands
	if cmd.Run == nil {
		if len(remainingArgs) > 0 {
			_, _ = fmt.Fprintf(t.stderr, "Error: unknown subcommand: %s\n\n", remainingArgs[0])
		} else {
			_, _ = fmt.Fprintf(t.stderr, "Error: command requires a subcommand\n\n")
		}
		path := strings.Join(args[:len(args)-len(remainingArgs)], " ")
		t.printHelp(cmd, path)
		return 1
	}

	// Setup and parse flags
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(t.stderr)

	if cmd.Flags != nil {
		cmd.Flags(fs)
	}

	if err := fs.Parse(remainingArgs); err != nil {
		return 1
	}

	// Execute command
	if err := cmd.Run(ctx, fs); err != nil {
		_, _ = fmt.Fprintf(t.stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}

// findCommand finds the deepest matching command in the tree.
// Returns the command and any remaining arguments that weren't part of the command path.
//
// Example: args=["capture", "run", "--flag", "value"]
// Returns: (capture_run_command, ["--flag", "value"])
func (t *Tree) findCommand(args []string) (cmd *Command, remaining []string) {
	if len(args) == 0 {
		return t.root, args
	}

	// Start from root
	current := t.root
	matchedArgs := 0

	// Try to descend as deep as possible
	for i, arg := range args {
		// Try to find subcommand or alias
		found := false

		// First, check direct subcommands
		for _, sub := range current.Subcommands {
			if sub.Name == arg {
				current = sub
				matchedArgs = i + 1
				found = true
				break
			}
		}

		// If not found, check aliases
		if !found {
			for _, sub := range current.Subcommands {
				for _, alias := range sub.Aliases {
					if alias == arg {
						current = sub
						matchedArgs = i + 1
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}

		// If no matching subcommand found, stop here
		if !found {
			break
		}
	}

	return current, args[matchedArgs:]
}

// printHelp generates and prints help text for a command.
func (t *Tree) printHelp(cmd *Command, path string) {
	if path == "" {
		path = cmd.Name
	}

	// Title
	if cmd.Short != "" {
		_, _ = fmt.Fprintf(t.stdout, "%s - %s\n\n", path, cmd.Short)
	} else {
		_, _ = fmt.Fprintf(t.stdout, "%s\n\n", path)
	}

	// Long description
	if cmd.Long != "" {
		_, _ = fmt.Fprintln(t.stdout, cmd.Long)
		_, _ = fmt.Fprintln(t.stdout)
	}

	// Usage
	_, _ = fmt.Fprintln(t.stdout, "Usage:")
	if len(cmd.Subcommands) > 0 {
		_, _ = fmt.Fprintf(t.stdout, "  %s <subcommand> [options]\n\n", path)
	} else {
		_, _ = fmt.Fprintf(t.stdout, "  %s [options]\n\n", path)
	}

	// Subcommands
	if len(cmd.Subcommands) > 0 {
		t.printSubcommands(cmd)
	}

	// Flags
	if cmd.Flags != nil {
		t.printFlags(cmd)
	}

	// Examples
	if len(cmd.Examples) > 0 {
		t.printExamples(cmd.Examples)
	}

	// Footer for subcommands
	if len(cmd.Subcommands) > 0 {
		_, _ = fmt.Fprintf(t.stdout, "Use '%s <subcommand> --help' for more information.\n", path)
	}
}

// printSubcommands prints the list of available subcommands.
func (t *Tree) printSubcommands(cmd *Command) {
	_, _ = fmt.Fprintln(t.stdout, "Available subcommands:")

	// Filter out hidden commands and sort
	visible := make([]*Command, 0, len(cmd.Subcommands))
	for _, sub := range cmd.Subcommands {
		if !sub.Hidden {
			visible = append(visible, sub)
		}
	}

	sort.Slice(visible, func(i, j int) bool {
		return visible[i].Name < visible[j].Name
	})

	// Calculate max name length for alignment
	maxLen := 0
	for _, sub := range visible {
		if len(sub.Name) > maxLen {
			maxLen = len(sub.Name)
		}
	}

	// Print each subcommand
	for _, sub := range visible {
		padding := strings.Repeat(" ", maxLen-len(sub.Name)+2)
		aliases := ""
		if len(sub.Aliases) > 0 {
			aliases = fmt.Sprintf(" (aliases: %s)", strings.Join(sub.Aliases, ", "))
		}
		_, _ = fmt.Fprintf(t.stdout, "  %s%s%s%s\n", sub.Name, padding, sub.Short, aliases)
	}
	_, _ = fmt.Fprintln(t.stdout)
}

// printFlags prints the available flags for a command.
func (t *Tree) printFlags(cmd *Command) {
	fs := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	fs.SetOutput(t.stdout)
	cmd.Flags(fs)

	// Check if there are any flags
	hasFlags := false
	fs.VisitAll(func(f *flag.Flag) {
		hasFlags = true
	})

	if hasFlags {
		_, _ = fmt.Fprintln(t.stdout, "Options:")
		fs.PrintDefaults()
		_, _ = fmt.Fprintln(t.stdout)
	}
}

// printExamples prints usage examples.
func (t *Tree) printExamples(examples []string) {
	_, _ = fmt.Fprintln(t.stdout, "Examples:")
	for _, ex := range examples {
		_, _ = fmt.Fprintf(t.stdout, "  %s\n", ex)
	}
	_, _ = fmt.Fprintln(t.stdout)
}

// Root returns the root command of the tree.
func (t *Tree) Root() *Command {
	return t.root
}

// Find finds a command by its path (e.g., "capture run").
// Returns nil if not found.
func (t *Tree) Find(path string) *Command {
	return t.index[path]
}

// CommandCount returns the total number of commands in the tree (including subcommands).
// Aliases are not counted separately.
func (t *Tree) CommandCount() int {
	// Count unique commands (not aliases)
	unique := make(map[*Command]bool)
	for _, cmd := range t.index {
		unique[cmd] = true
	}
	return len(unique)
}
