package clitree

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestTree_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		wantStdout string
		wantStderr string
		args       []string
		wantExit   int
	}{
		{
			name:       "no args shows help",
			args:       []string{},
			wantExit:   1,
			wantStdout: "Test application",
		},
		{
			name:       "help flag",
			args:       []string{"help"},
			wantExit:   0,
			wantStdout: "Available subcommands:",
		},
		{
			name:       "version command",
			args:       []string{"version"},
			wantExit:   0,
			wantStdout: "",
		},
		{
			name:       "unknown command",
			args:       []string{"unknown"},
			wantExit:   1,
			wantStderr: "unknown subcommand",
		},
		{
			name:       "command help",
			args:       []string{"greet", "--help"},
			wantExit:   0,
			wantStdout: "Greet someone",
		},
		{
			name:       "command with flag",
			args:       []string{"greet", "--name", "Alice"},
			wantExit:   0,
			wantStdout: "",
		},
		{
			name:       "dash h help",
			args:       []string{"-h"},
			wantExit:   0,
			wantStdout: "Available subcommands:",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var capturedArg string
			root := &Command{
				Name:  "test",
				Short: "Test application",
				Subcommands: []*Command{
					{
						Name:  "version",
						Short: "Show version",
						Run: func(ctx context.Context, fs *flag.FlagSet) error {
							return nil
						},
					},
					{
						Name:  "greet",
						Short: "Greet someone",
						Flags: func(fs *flag.FlagSet) {
							fs.StringVar(&capturedArg, "name", "World", "Name to greet")
						},
						Run: func(ctx context.Context, fs *flag.FlagSet) error {
							return nil
						},
					},
				},
			}

			var stdout, stderr bytes.Buffer
			tree := New(root, &stdout, &stderr)

			exitCode := tree.Execute(context.Background(), tt.args)

			if exitCode != tt.wantExit {
				t.Errorf("Execute() exitCode = %v, want %v", exitCode, tt.wantExit)
			}

			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout does not contain %q\ngot: %s", tt.wantStdout, stdout.String())
			}

			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr does not contain %q\ngot: %s", tt.wantStderr, stderr.String())
			}

			if tt.name == "command with flag" && capturedArg != "Alice" {
				t.Errorf("flag not parsed correctly: got %q, want %q", capturedArg, "Alice")
			}
		})
	}
}

func TestTree_ExecuteError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("command failed")

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "fail",
				Run: func(ctx context.Context, fs *flag.FlagSet) error {
					return expectedErr
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	exitCode := tree.Execute(context.Background(), []string{"fail"})

	if exitCode != 1 {
		t.Errorf("Execute() exitCode = %v, want 1", exitCode)
	}

	if !strings.Contains(stderr.String(), "command failed") {
		t.Errorf("stderr should contain error message, got: %s", stderr.String())
	}
}

func TestTree_ExecuteInvalidFlag(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "cmd",
				Flags: func(fs *flag.FlagSet) {
					fs.String("valid", "", "Valid flag")
				},
				Run: func(ctx context.Context, fs *flag.FlagSet) error {
					return nil
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	exitCode := tree.Execute(context.Background(), []string{"cmd", "--invalid"})

	if exitCode != 1 {
		t.Errorf("Execute() exitCode = %v, want 1 for invalid flag", exitCode)
	}
}

func TestTree_FindCommand(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "config",
				Subcommands: []*Command{
					{
						Name: "get",
						Run:  func(ctx context.Context, fs *flag.FlagSet) error { return nil },
					},
					{
						Name: "set",
						Run:  func(ctx context.Context, fs *flag.FlagSet) error { return nil },
					},
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tests := []struct {
		args          []string
		wantCmdName   string
		wantRemaining []string
	}{
		{
			args:          []string{"config"},
			wantCmdName:   "config",
			wantRemaining: []string{},
		},
		{
			args:          []string{"config", "get"},
			wantCmdName:   "get",
			wantRemaining: []string{},
		},
		{
			args:          []string{"config", "get", "--flag"},
			wantCmdName:   "get",
			wantRemaining: []string{"--flag"},
		},
		{
			args:          []string{"config", "get", "--flag", "value"},
			wantCmdName:   "get",
			wantRemaining: []string{"--flag", "value"},
		},
		{
			args:          []string{"unknown"},
			wantCmdName:   "app",
			wantRemaining: []string{"unknown"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			t.Parallel()

			cmd, remaining := tree.findCommand(tt.args)

			if cmd.Name != tt.wantCmdName {
				t.Errorf("findCommand() cmd.Name = %v, want %v", cmd.Name, tt.wantCmdName)
			}

			if len(remaining) != len(tt.wantRemaining) {
				t.Errorf("findCommand() remaining length = %v, want %v", len(remaining), len(tt.wantRemaining))
			}

			for i, arg := range remaining {
				if i < len(tt.wantRemaining) && arg != tt.wantRemaining[i] {
					t.Errorf("findCommand() remaining[%d] = %v, want %v", i, arg, tt.wantRemaining[i])
				}
			}
		})
	}
}

func TestTree_Aliases(t *testing.T) {
	t.Parallel()

	for _, alias := range []string{"version", "-v", "--version", "ver"} {
		alias := alias
		t.Run(alias, func(t *testing.T) {
			t.Parallel()

			runCount := 0
			root := &Command{
				Name: "app",
				Subcommands: []*Command{
					{
						Name:    "version",
						Aliases: []string{"-v", "--version", "ver"},
						Run: func(ctx context.Context, fs *flag.FlagSet) error {
							runCount++
							return nil
						},
					},
				},
			}

			var stdout, stderr bytes.Buffer
			tree := New(root, &stdout, &stderr)

			exitCode := tree.Execute(context.Background(), []string{alias})

			if exitCode != 0 {
				t.Errorf("Execute(%s) = %v, want 0", alias, exitCode)
			}

			if runCount != 1 {
				t.Errorf("Execute(%s) runCount = %v, want 1", alias, runCount)
			}
		})
	}
}

func TestTree_Find(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "config",
				Subcommands: []*Command{
					{Name: "get"},
					{Name: "set"},
				},
			},
			{
				Name:    "version",
				Aliases: []string{"-v"},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tests := []struct {
		path      string
		wantName  string
		wantFound bool
	}{
		{path: "app", wantName: "app", wantFound: true},
		{path: "config", wantName: "config", wantFound: true},
		{path: "config get", wantName: "get", wantFound: true},
		{path: "config set", wantName: "set", wantFound: true},
		{path: "version", wantName: "version", wantFound: true},
		{path: "-v", wantName: "version", wantFound: true},
		{path: "unknown", wantName: "", wantFound: false},
		{path: "config unknown", wantName: "", wantFound: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()

			cmd := tree.Find(tt.path)

			if tt.wantFound && cmd == nil {
				t.Errorf("Find(%q) = nil, want non-nil", tt.path)
			}

			if !tt.wantFound && cmd != nil {
				t.Errorf("Find(%q) = %v, want nil", tt.path, cmd.Name)
			}

			if tt.wantFound && cmd != nil && cmd.Name != tt.wantName {
				t.Errorf("Find(%q).Name = %v, want %v", tt.path, cmd.Name, tt.wantName)
			}
		})
	}
}

func TestTree_CommandCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		root      *Command
		name      string
		wantCount int
	}{
		{
			name: "single command",
			root: &Command{
				Name: "app",
			},
			wantCount: 1,
		},
		{
			name: "with subcommands",
			root: &Command{
				Name: "app",
				Subcommands: []*Command{
					{Name: "cmd1"},
					{Name: "cmd2"},
				},
			},
			wantCount: 3,
		},
		{
			name: "nested subcommands",
			root: &Command{
				Name: "app",
				Subcommands: []*Command{
					{Name: "cmd1"},
					{
						Name: "cmd2",
						Subcommands: []*Command{
							{Name: "sub1"},
							{Name: "sub2"},
						},
					},
				},
			},
			wantCount: 5,
		},
		{
			name: "with aliases",
			root: &Command{
				Name: "app",
				Subcommands: []*Command{
					{
						Name:    "version",
						Aliases: []string{"-v", "--version"},
					},
				},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			tree := New(tt.root, &stdout, &stderr)

			count := tree.CommandCount()

			if count != tt.wantCount {
				t.Errorf("CommandCount() = %v, want %v", count, tt.wantCount)
			}
		})
	}
}

func TestTree_HiddenCommands(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name:  "visible",
				Short: "Visible command",
			},
			{
				Name:   "hidden",
				Short:  "Hidden command",
				Hidden: true,
			},
			{
				Name:   "deprecated",
				Short:  "Deprecated command",
				Hidden: true,
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tree.Execute(context.Background(), []string{"help"})

	output := stdout.String()

	if !strings.Contains(output, "visible") {
		t.Error("help should show visible command")
	}

	if strings.Contains(output, "hidden") {
		t.Error("help should not show hidden command")
	}

	if strings.Contains(output, "deprecated") {
		t.Error("help should not show deprecated command")
	}
}

func TestTree_HiddenCommandsStillExecutable(t *testing.T) {
	t.Parallel()

	runCount := 0

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name:   "hidden",
				Hidden: true,
				Run: func(ctx context.Context, fs *flag.FlagSet) error {
					runCount++
					return nil
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	exitCode := tree.Execute(context.Background(), []string{"hidden"})

	if exitCode != 0 {
		t.Errorf("Hidden command should still be executable, exit code = %v", exitCode)
	}

	if runCount != 1 {
		t.Errorf("Hidden command should execute, runCount = %v", runCount)
	}
}

func TestTree_Root(t *testing.T) {
	t.Parallel()

	root := &Command{Name: "myapp"}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	if tree.Root() != root {
		t.Error("Root() should return the root command")
	}

	if tree.Root().Name != "myapp" {
		t.Errorf("Root().Name = %v, want myapp", tree.Root().Name)
	}
}

func TestTree_HelpWithLongDescription(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name:  "app",
		Short: "Short description",
		Long:  "This is a much longer description\nthat spans multiple lines\nand provides detailed information.",
		Subcommands: []*Command{
			{Name: "cmd1"},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tree.Execute(context.Background(), []string{"help"})

	output := stdout.String()

	if !strings.Contains(output, "Short description") {
		t.Error("help should contain short description")
	}

	if !strings.Contains(output, "much longer description") {
		t.Error("help should contain long description")
	}
}

func TestTree_HelpWithExamples(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name:  "deploy",
		Short: "Deploy app",
		Examples: []string{
			"deploy --env production",
			"deploy --env staging --dry-run",
		},
		Run: func(ctx context.Context, fs *flag.FlagSet) error { return nil },
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tree.Execute(context.Background(), []string{"--help"})

	output := stdout.String()

	if !strings.Contains(output, "Examples:") {
		t.Error("help should contain Examples section")
	}

	if !strings.Contains(output, "deploy --env production") {
		t.Error("help should contain example 1")
	}

	if !strings.Contains(output, "deploy --env staging --dry-run") {
		t.Error("help should contain example 2")
	}
}

func TestTree_HelpWithFlags(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name:  "build",
		Short: "Build project",
		Flags: func(fs *flag.FlagSet) {
			fs.String("output", "dist/", "Output directory")
			fs.Bool("verbose", false, "Verbose output")
		},
		Run: func(ctx context.Context, fs *flag.FlagSet) error { return nil },
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tree.Execute(context.Background(), []string{"--help"})

	output := stdout.String()

	if !strings.Contains(output, "Options:") {
		t.Error("help should contain Options section")
	}

	if !strings.Contains(output, "output") {
		t.Error("help should list output flag")
	}

	if !strings.Contains(output, "verbose") {
		t.Error("help should list verbose flag")
	}
}

func TestTree_SubcommandHelp(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name:  "config",
				Short: "Manage config",
				Subcommands: []*Command{
					{
						Name:  "get",
						Short: "Get config value",
					},
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tree.Execute(context.Background(), []string{"config", "help"})

	output := stdout.String()

	if !strings.Contains(output, "Manage config") {
		t.Error("subcommand help should show description")
	}

	if !strings.Contains(output, "get") {
		t.Error("subcommand help should show nested commands")
	}
}

func TestTree_ContextCancellation(t *testing.T) {
	t.Parallel()

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "task",
				Run: func(ctx context.Context, fs *flag.FlagSet) error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						return nil
					}
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	exitCode := tree.Execute(canceledCtx, []string{"task"})

	if exitCode != 1 {
		t.Errorf("Canceled context should result in error, exit code = %v", exitCode)
	}

	if !strings.Contains(stderr.String(), "context canceled") {
		t.Error("Error should mention context cancellation")
	}
}

func TestTree_CommandWithoutRun(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name:  "group",
				Short: "Command group",
				Subcommands: []*Command{
					{
						Name: "action",
						Run:  func(ctx context.Context, fs *flag.FlagSet) error { return nil },
					},
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	tree.Execute(context.Background(), []string{"group"})

	if !strings.Contains(stdout.String(), "Command group") {
		t.Error("Should show help for command without Run function")
	}
}

func TestTree_EmptyCommand(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	exitCode := tree.Execute(context.Background(), []string{})

	if exitCode != 1 {
		t.Errorf("Empty command should show help and return 1, got %v", exitCode)
	}
}

// Benchmark tests
func BenchmarkTree_Execute(b *testing.B) {
	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "version",
				Run:  func(ctx context.Context, fs *flag.FlagSet) error { return nil },
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Execute(ctx, []string{"version"})
	}
}

func BenchmarkTree_FindCommand(b *testing.B) {
	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "config",
				Subcommands: []*Command{
					{Name: "get"},
					{Name: "set"},
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)
	args := []string{"config", "get"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.findCommand(args)
	}
}

func BenchmarkTree_BuildIndex(b *testing.B) {
	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{Name: "cmd1"},
			{Name: "cmd2", Subcommands: []*Command{
				{Name: "sub1"},
				{Name: "sub2"},
			}},
			{Name: "cmd3"},
		},
	}

	var stdout, stderr bytes.Buffer

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(root, &stdout, &stderr)
	}
}
