package clitree_test

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/unolink/clitree"
)

// Example shows basic usage of clitree.
func Example() {
	// Define your command structure
	root := &clitree.Command{
		Name:  "myapp",
		Short: "My CLI application",
		Long:  "A longer description of what my application does.",

		Subcommands: []*clitree.Command{
			{
				Name:    "version",
				Aliases: []string{"-v", "--version"},
				Short:   "Show version",
				Run: func(ctx context.Context, fs *flag.FlagSet) error {
					fmt.Println("myapp version 1.0.0")
					return nil
				},
			},
		},
	}

	// Create tree and execute
	tree := clitree.New(root, os.Stdout, os.Stderr)
	exitCode := tree.Execute(context.Background(), []string{"version"})

	fmt.Printf("Exit code: %d\n", exitCode)
	// Output:
	// myapp version 1.0.0
	// Exit code: 0
}

// ExampleCommand_flags shows how to use flags with commands.
// IMPORTANT: For concurrent/long-lived applications, extract flag values
// inside Run() using fs.Lookup() instead of closure variables to avoid race conditions.
func ExampleCommand_flags() {
	root := &clitree.Command{
		Name:  "greet",
		Short: "Greet someone",

		Flags: func(fs *flag.FlagSet) {
			fs.String("name", "World", "Name to greet")
			fs.Int("count", 1, "Number of times to greet")
		},

		Run: func(ctx context.Context, fs *flag.FlagSet) error {
			// Extract flag values inside Run for thread-safety
			name := fs.Lookup("name").Value.String()
			count := 1
			if countFlag := fs.Lookup("count"); countFlag != nil {
				// flag package stores int as string internally
				_, _ = fmt.Sscanf(countFlag.Value.String(), "%d", &count)
			}

			for i := 0; i < count; i++ {
				fmt.Printf("Hello, %s!\n", name)
			}
			return nil
		},
	}

	tree := clitree.New(root, os.Stdout, os.Stderr)
	tree.Execute(context.Background(), []string{"--name", "Alice", "--count", "2"})

	// Output:
	// Hello, Alice!
	// Hello, Alice!
}

// ExampleCommand_subcommands shows nested subcommands.
func ExampleCommand_subcommands() {
	root := &clitree.Command{
		Name:  "myapp",
		Short: "My application",

		Subcommands: []*clitree.Command{
			{
				Name:  "config",
				Short: "Manage configuration",

				Subcommands: []*clitree.Command{
					{
						Name:  "get",
						Short: "Get configuration value",
						Flags: func(fs *flag.FlagSet) {
							fs.String("file", "config.yaml", "Config file")
						},
						Run: func(ctx context.Context, fs *flag.FlagSet) error {
							// Extract flag value inside Run for thread-safety
							configFile := fs.Lookup("file").Value.String()
							key := fs.Arg(0)
							fmt.Printf("Getting %s from %s\n", key, configFile)
							return nil
						},
					},
					{
						Name:  "set",
						Short: "Set configuration value",
						Run: func(ctx context.Context, fs *flag.FlagSet) error {
							key := fs.Arg(0)
							value := fs.Arg(1)
							fmt.Printf("Setting %s = %s\n", key, value)
							return nil
						},
					},
				},
			},
		},
	}

	tree := clitree.New(root, os.Stdout, os.Stderr)
	tree.Execute(context.Background(), []string{"config", "get", "database.host"})

	// Output:
	// Getting database.host from config.yaml
}

// ExampleCommand_examples shows how to add usage examples.
func ExampleCommand_examples() {
	root := &clitree.Command{
		Name:  "deploy",
		Short: "Deploy application",
		Long:  "Deploy application to various environments with different configurations.",

		Examples: []string{
			"deploy --env production",
			"deploy --env staging --dry-run",
			"deploy --env dev --skip-tests",
		},

		Run: func(ctx context.Context, fs *flag.FlagSet) error {
			return nil
		},
	}

	tree := clitree.New(root, os.Stdout, os.Stderr)
	tree.Execute(context.Background(), []string{"--help"})
	// Help output will include the examples section
}

// ExampleTree_Find shows how to lookup commands programmatically.
func ExampleTree_Find() {
	root := &clitree.Command{
		Name: "app",
		Subcommands: []*clitree.Command{
			{
				Name: "config",
				Subcommands: []*clitree.Command{
					{Name: "get"},
				},
			},
		},
	}

	tree := clitree.New(root, os.Stdout, os.Stderr)

	// Find commands by path
	if cmd := tree.Find("config"); cmd != nil {
		fmt.Printf("Found: %s\n", cmd.Name)
	}

	if cmd := tree.Find("config get"); cmd != nil {
		fmt.Printf("Found: %s\n", cmd.Name)
	}

	// Output:
	// Found: config
	// Found: get
}

// ExampleCommand_context shows context usage for cancellation.
func ExampleCommand_context() {
	root := &clitree.Command{
		Name:  "myapp",
		Short: "My application",
		Subcommands: []*clitree.Command{
			{
				Name:  "task",
				Short: "Long running task",
				Run: func(ctx context.Context, fs *flag.FlagSet) error {
					// Check context cancellation
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						fmt.Println("Task completed")
						return nil
					}
				},
			},
		},
	}

	tree := clitree.New(root, os.Stdout, os.Stderr)
	ctx := context.Background()
	tree.Execute(ctx, []string{"task"})

	// Output:
	// Task completed
}
