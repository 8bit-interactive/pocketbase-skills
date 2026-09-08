package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndResolveEnvironment(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, VersionFile), []byte("0.40.3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"projectName":"demo","pockethost":{"branchEnvironmentMap":{"staging":"staging"},"environments":{"staging":{"instanceName":"demo-staging"}}}}`
	if err := os.WriteFile(filepath.Join(root, ConfigFile), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_REF_NAME", "staging")
	if got := p.ResolveEnvironment(""); got != "staging" {
		t.Fatalf("environment = %q, want staging", got)
	}
	if got := p.ResolveInstance("staging"); got != "demo-staging" {
		t.Fatalf("instance = %q, want demo-staging", got)
	}
}

func TestFindRootWalksParents(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "nested", "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, VersionFile), []byte("0.40.3"), 0o644); err != nil {
		t.Fatal(err)
	}
	found, err := FindRoot(child)
	if err != nil {
		t.Fatal(err)
	}
	if found != root {
		t.Fatalf("root = %q, want %q", found, root)
	}
}
