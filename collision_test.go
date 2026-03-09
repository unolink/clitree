package clitree

import (
	"bytes"
	"testing"
)

// TestIndexCollision_RelativePaths verifies that two commands with identical
// relative paths (e.g., "config get" and "network get") are stored as distinct
// entries in the global index using the first-wins strategy for collisions.
func TestIndexCollision_RelativePaths(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name: "config",
				Subcommands: []*Command{
					{Name: "get", Short: "Get config value"},
				},
			},
			{
				Name: "network",
				Subcommands: []*Command{
					{Name: "get", Short: "Get network info"},
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	configGet := tree.Find("config get")
	if configGet == nil {
		t.Fatal("Find('config get') returned nil")
	}
	if configGet.Short != "Get config value" {
		t.Errorf("Find('config get').Short = %q, want 'Get config value'", configGet.Short)
	}

	networkGet := tree.Find("network get")
	if networkGet == nil {
		t.Fatal("Find('network get') returned nil")
	}
	if networkGet.Short != "Get network info" {
		t.Errorf("Find('network get').Short = %q, want 'Get network info'", networkGet.Short)
	}

	if configGet == networkGet {
		t.Error("config get and network get point to the same command (collision detected)")
	}
}

// TestIndexCollision_Aliases verifies that when two commands at different
// nesting levels share the same alias, the first-registered command wins.
func TestIndexCollision_Aliases(t *testing.T) {
	t.Parallel()

	root := &Command{
		Name: "app",
		Subcommands: []*Command{
			{
				Name:    "version",
				Aliases: []string{"-v"},
				Short:   "App version",
			},
			{
				Name: "config",
				Subcommands: []*Command{
					{
						Name:    "verbose",
						Aliases: []string{"-v"},
						Short:   "Config verbose mode",
					},
				},
			},
		},
	}

	var stdout, stderr bytes.Buffer
	tree := New(root, &stdout, &stderr)

	// First-registered command should win the alias
	cmd := tree.Find("-v")
	if cmd == nil {
		t.Fatal("Find('-v') returned nil")
	}

	// Both commands should still be accessible by full path
	version := tree.Find("version")
	verbose := tree.Find("config verbose")

	if version == nil {
		t.Error("version command should be accessible")
	}
	if verbose == nil {
		t.Error("verbose command should be accessible")
	}

	if cmd != version {
		t.Errorf("Alias '-v' should point to 'version' command (first registered), but points to %s", cmd.Name)
	}

	if cmd == verbose {
		t.Error("Alias '-v' should not point to 'verbose' (second command with same alias)")
	}
}
