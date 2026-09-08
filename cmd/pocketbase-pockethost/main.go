package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/8bit-interactive/pocketbase-pockethost-skills/internal/deploy"
	"github.com/8bit-interactive/pocketbase-pockethost-skills/internal/keys"
	"github.com/8bit-interactive/pocketbase-pockethost-skills/internal/pocketbase"
	"github.com/8bit-interactive/pocketbase-pockethost-skills/internal/project"
)

const cliVersion = "0.1.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		usage()
		return nil
	}
	switch args[0] {
	case "version":
		fmt.Println(cliVersion)
		return nil
	case "init":
		return initProject(args[1:])
	case "key":
		return keyCommand(args[1:])
	case "pocketbase":
		return pocketbaseCommand(args[1:])
	case "validate":
		return validateCommand(args[1:])
	case "doctor":
		return doctorCommand(args[1:])
	case "deploy":
		return deployCommand(args[1:])
	case "health":
		return healthCommand(args[1:])
	case "workflow":
		return workflowCommand(args[1:])
	default:
		return fmt.Errorf("unknown command %q; use --help", args[0])
	}
}

func usage() {
	fmt.Println(`pocketbase-pockethost ` + cliVersion + `

Usage:
  pocketbase-pockethost init [--dir PATH] [--version VERSION]
  pocketbase-pockethost key generate [--output PATH]
  pocketbase-pockethost pocketbase install [--version VERSION]
  pocketbase-pockethost validate
  pocketbase-pockethost doctor [--env NAME] [--json]
  pocketbase-pockethost deploy [--env NAME] [--dry-run] [--clean-slate --yes]
  pocketbase-pockethost health [--env NAME]
  pocketbase-pockethost workflow install [--force]

Deployment variables:
  POCKETHOST_SFTP_USERNAME, POCKETHOST_SFTP_PRIVATE_KEY_PATH,
  POCKETHOST_SFTP_PRIVATE_KEY, POCKETHOST_INSTANCE_NAME,
  POCKETHOST_SFTP_HOST_KEY, and HEALTHCHECK_BASE_URL`)
}

func initProject(args []string) error {
	set := flag.NewFlagSet("init", flag.ContinueOnError)
	directory := set.String("dir", ".", "project directory")
	version := set.String("version", pocketbase.DefaultVersion, "PocketBase version")
	force := set.Bool("force", false, "overwrite generated files")
	if err := set.Parse(args); err != nil {
		return err
	}
	root, err := filepath.Abs(*directory)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"pb_public", "pb_hooks", "pb_migrations"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			return err
		}
	}
	if err := writeGenerated(filepath.Join(root, ".pb_version"), strings.TrimPrefix(*version, "v")+"\n", *force); err != nil {
		return err
	}
	config := project.Config{ProjectName: filepath.Base(root)}
	config.Pockethost.BranchEnvironmentMap = map[string]string{"main": "production", "master": "production", "staging": "staging"}
	config.Pockethost.Environments = map[string]project.Environment{"production": {}, "staging": {}}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := writeGenerated(filepath.Join(root, ".pb_config.json"), string(data)+"\n", *force); err != nil {
		return err
	}
	workflowPath := filepath.Join(root, ".github", "workflows", "pockethost-deploy.yml")
	if err := writeGenerated(workflowPath, workflowText(), *force); err != nil {
		return err
	}
	fmt.Printf("Initialized PocketBase project in %s\n", root)
	return nil
}

func keyCommand(args []string) error {
	if len(args) == 0 || args[0] == "help" {
		return errors.New("usage: key generate [--output PATH]")
	}
	if args[0] != "generate" {
		return fmt.Errorf("unknown key command %q", args[0])
	}
	set := flag.NewFlagSet("key generate", flag.ContinueOnError)
	output := set.String("output", "", "private key output path")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	path := *output
	if path == "" {
		var err error
		path, err = keys.DefaultPrivatePath()
		if err != nil {
			return err
		}
	}
	pair, err := keys.Generate(path)
	if err != nil {
		return err
	}
	fmt.Printf("Private key: %s\nPublic key: %s\nFingerprint: %s\n", pair.PrivatePath, pair.PublicPath, pair.Fingerprint)
	fmt.Println("Register the public key in PocketHost Account -> Keys before deploying.")
	return nil
}

func pocketbaseCommand(args []string) error {
	if len(args) == 0 || args[0] != "install" {
		return errors.New("usage: pocketbase install [--version VERSION]")
	}
	set := flag.NewFlagSet("pocketbase install", flag.ContinueOnError)
	version := set.String("version", "", "PocketBase version override")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	p, err := project.LoadFrom(".")
	if err != nil {
		return err
	}
	selected := p.Version
	if *version != "" {
		selected = *version
	}
	binary, err := pocketbase.Ensure(p.Root, selected)
	if err != nil {
		return err
	}
	fmt.Println(binary)
	return nil
}

func validateCommand(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: validate")
	}
	p, err := project.LoadFrom(".")
	if err != nil {
		return err
	}
	return pocketbase.Validate(p.Root, p.Version)
}

func doctorCommand(args []string) error {
	set := flag.NewFlagSet("doctor", flag.ContinueOnError)
	environment := set.String("env", "", "environment name")
	jsonOutput := set.Bool("json", false, "print machine-readable output")
	if err := set.Parse(args); err != nil {
		return err
	}
	p, err := project.LoadFrom(".")
	if err != nil {
		return err
	}
	result := map[string]any{"version": p.Version, "root": p.Root, "environment": p.ResolveEnvironment(*environment)}
	surface, err := p.Surface()
	if err != nil {
		return err
	}
	result["surface"] = surface
	result["instance"] = p.ResolveInstance(p.ResolveEnvironment(*environment))
	result["sftpUsername"] = os.Getenv("POCKETHOST_SFTP_USERNAME") != ""
	keyPath := os.Getenv("POCKETHOST_SFTP_PRIVATE_KEY_PATH")
	result["privateKey"] = keyPath != "" || os.Getenv("POCKETHOST_SFTP_PRIVATE_KEY") != ""
	if *jsonOutput {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Project: %s\nPocketBase: %s\nEnvironment: %s\nInstance: %s\n", p.Root, p.Version, result["environment"], result["instance"])
	fmt.Printf("SFTP username: %t\nPrivate key: %t\n", result["sftpUsername"], result["privateKey"])
	for name, present := range surface {
		fmt.Printf("%s: %t\n", name, present)
	}
	if !result["sftpUsername"].(bool) || !result["privateKey"].(bool) || result["instance"] == "" {
		return errors.New("deployment configuration is incomplete")
	}
	return nil
}

func deployCommand(args []string) error {
	set := flag.NewFlagSet("deploy", flag.ContinueOnError)
	environment := set.String("env", "", "environment name")
	dryRun := set.Bool("dry-run", false, "show changes without uploading")
	cleanSlate := set.Bool("clean-slate", false, "remove all files in managed directories")
	yes := set.Bool("yes", false, "confirm destructive operations")
	host := set.String("host", envOr("POCKETHOST_SFTP_HOST", deploy.DefaultHost), "SFTP host")
	port := set.Int("port", envInt("POCKETHOST_SFTP_PORT", deploy.DefaultPort), "SFTP port")
	if err := set.Parse(args); err != nil {
		return err
	}
	p, err := project.LoadFrom(".")
	if err != nil {
		return err
	}
	envName := p.ResolveEnvironment(*environment)
	options := deploy.Options{
		Project: p, Environment: envName, DryRun: *dryRun, CleanSlate: *cleanSlate, Yes: *yes,
		Host: *host, Port: *port, Username: os.Getenv("POCKETHOST_SFTP_USERNAME"),
		PrivateKeyPath: os.Getenv("POCKETHOST_SFTP_PRIVATE_KEY_PATH"), PrivateKeyValue: os.Getenv("POCKETHOST_SFTP_PRIVATE_KEY"),
		HostKey: os.Getenv("POCKETHOST_SFTP_HOST_KEY"), KnownHosts: os.Getenv("POCKETHOST_SFTP_KNOWN_HOSTS"), Instance: p.ResolveInstance(envName),
	}
	_, err = deploy.Run(options)
	return err
}

func healthCommand(args []string) error {
	set := flag.NewFlagSet("health", flag.ContinueOnError)
	environment := set.String("env", "", "environment name")
	if err := set.Parse(args); err != nil {
		return err
	}
	p, err := project.LoadFrom(".")
	if err != nil {
		return err
	}
	envName := p.ResolveEnvironment(*environment)
	url, marker := p.ResolveHealthcheck(envName, p.ResolveInstance(envName))
	if url == "" {
		return errors.New("missing healthcheck URL")
	}
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("healthcheck failed: HTTP %s", response.Status)
	}
	if marker != "" {
		data := make([]byte, 4*1024*1024)
		n, _ := response.Body.Read(data)
		if !strings.Contains(string(data[:n]), marker) {
			return fmt.Errorf("healthcheck marker %q was not found", marker)
		}
	}
	fmt.Printf("Healthcheck passed for %s\n", url)
	return nil
}

func workflowCommand(args []string) error {
	if len(args) == 0 || args[0] != "install" {
		return errors.New("usage: workflow install [--force]")
	}
	set := flag.NewFlagSet("workflow install", flag.ContinueOnError)
	force := set.Bool("force", false, "overwrite an existing workflow")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	p, err := project.LoadFrom(".")
	if err != nil {
		return err
	}
	target := filepath.Join(p.Root, ".github", "workflows", "pockethost-deploy.yml")
	if !*force {
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("workflow already exists at %s; use --force", target)
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(target, []byte(workflowText()), 0o644); err != nil {
		return err
	}
	fmt.Println(target)
	return nil
}

func writeGenerated(path, content string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists at %s; use --force", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return fallback
	}
	return parsed
}

func workflowText() string {
	return `name: Deploy to Pockethost

on:
  push:
    branches: [main, master, staging]
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: pockethost-deploy-${{ github.ref_name }}
  cancel-in-progress: true

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: ${{ github.ref_name == 'staging' && 'staging' || 'production' }}
    env:
      POCKETHOST_SFTP_USERNAME: ${{ secrets.POCKETHOST_SFTP_USERNAME }}
      POCKETHOST_SFTP_PRIVATE_KEY: ${{ secrets.POCKETHOST_SFTP_PRIVATE_KEY }}
      POCKETHOST_SFTP_HOST_KEY: ${{ vars.POCKETHOST_SFTP_HOST_KEY }}
      POCKETHOST_INSTANCE_NAME: ${{ vars.POCKETHOST_INSTANCE_NAME != '' && vars.POCKETHOST_INSTANCE_NAME || secrets.POCKETHOST_INSTANCE_NAME }}

    steps:
      - uses: actions/checkout@v4
      - name: Download pocketbase-pockethost
        env:
          VERSION: 0.1.0
        run: |
          curl --fail --location --silent --show-error \
            "https://github.com/8bit-interactive/pocketbase-pockethost-skills/releases/download/v${VERSION}/pocketbase-pockethost_${VERSION}_linux_amd64.tar.gz" \
            --output /tmp/pocketbase-pockethost.tar.gz
          curl --fail --location --silent --show-error \
            "https://github.com/8bit-interactive/pocketbase-pockethost-skills/releases/download/v${VERSION}/checksums.txt" \
            --output /tmp/pocketbase-pockethost-checksums.txt
          cd /tmp
          grep "pocketbase-pockethost_${VERSION}_linux_amd64.tar.gz" pocketbase-pockethost-checksums.txt | sha256sum --check
          tar -xzf /tmp/pocketbase-pockethost.tar.gz -C /tmp
          install /tmp/pocketbase-pockethost /usr/local/bin/pocketbase-pockethost
      - run: pocketbase-pockethost doctor
      - run: pocketbase-pockethost validate
      - run: pocketbase-pockethost deploy
      - run: pocketbase-pockethost health
`
}
