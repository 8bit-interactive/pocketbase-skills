package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigFile  = ".pb_config.json"
	VersionFile = ".pb_version"
)

type Environment struct {
	InstanceName      string `json:"instanceName,omitempty"`
	TenantID          string `json:"tenantId,omitempty"`
	HealthcheckURL    string `json:"healthcheckBaseUrl,omitempty"`
	HealthcheckMarker string `json:"healthcheckMarker,omitempty"`
}

type Config struct {
	ProjectName string `json:"projectName,omitempty"`
	Pockethost  struct {
		BranchEnvironmentMap map[string]string      `json:"branchEnvironmentMap,omitempty"`
		HealthcheckBaseURL   string                 `json:"healthcheckBaseUrl,omitempty"`
		Environments         map[string]Environment `json:"environments,omitempty"`
	} `json:"pockethost,omitempty"`
}

type Project struct {
	Root    string
	Config  Config
	Version string
}

func FindRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		configExists := fileExists(filepath.Join(current, ConfigFile))
		versionExists := fileExists(filepath.Join(current, VersionFile))
		if configExists || versionExists {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find %s or %s from %s", ConfigFile, VersionFile, start)
		}
		current = parent
	}
}

func Load(root string) (Project, error) {
	configPath := filepath.Join(root, ConfigFile)
	versionPath := filepath.Join(root, VersionFile)
	if !fileExists(configPath) {
		return Project{}, fmt.Errorf("missing %s in %s", ConfigFile, root)
	}
	if !fileExists(versionPath) {
		return Project{}, fmt.Errorf("missing %s in %s", VersionFile, root)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return Project{}, err
	}
	var config Config
	if err := json.Unmarshal(configData, &config); err != nil {
		return Project{}, fmt.Errorf("parse %s: %w", configPath, err)
	}
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		return Project{}, err
	}
	version := strings.TrimSpace(string(versionData))
	if version == "" {
		return Project{}, errors.New(".pb_version is empty")
	}
	return Project{Root: root, Config: config, Version: version}, nil
}

func LoadFrom(start string) (Project, error) {
	root, err := FindRoot(start)
	if err != nil {
		return Project{}, err
	}
	return Load(root)
}

func (p Project) Environment(name string) Environment {
	if p.Config.Pockethost.Environments == nil {
		return Environment{}
	}
	return p.Config.Pockethost.Environments[name]
}

func (p Project) ResolveEnvironment(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if value := os.Getenv("POCKETHOST_ENVIRONMENT"); value != "" {
		return value
	}
	branch := os.Getenv("GITHUB_REF_NAME")
	if branch == "" {
		branch = currentBranch(p.Root)
	}
	if mapped := p.Config.Pockethost.BranchEnvironmentMap[branch]; mapped != "" {
		return mapped
	}
	if branch == "staging" {
		return "staging"
	}
	return "production"
}

func (p Project) ResolveInstance(environment string) string {
	if value := os.Getenv("POCKETHOST_INSTANCE_NAME"); value != "" {
		return value
	}
	configured := p.Environment(environment)
	if configured.InstanceName != "" {
		return configured.InstanceName
	}
	if configured.TenantID != "" {
		return configured.TenantID
	}
	return ""
}

func (p Project) ResolveHealthcheck(environment, instance string) (string, string) {
	if value := os.Getenv("HEALTHCHECK_BASE_URL"); value != "" {
		return value, p.Environment(environment).HealthcheckMarker
	}
	configured := p.Environment(environment)
	if configured.HealthcheckURL != "" {
		return configured.HealthcheckURL, configured.HealthcheckMarker
	}
	if p.Config.Pockethost.HealthcheckBaseURL != "" {
		return p.Config.Pockethost.HealthcheckBaseURL, configured.HealthcheckMarker
	}
	if instance != "" {
		return "https://" + instance + ".pockethost.io", configured.HealthcheckMarker
	}
	return "", configured.HealthcheckMarker
}

func (p Project) Surface() (map[string]bool, error) {
	surface := map[string]bool{}
	for _, name := range []string{"pb_public", "pb_hooks", "pb_migrations"} {
		info, err := os.Stat(filepath.Join(p.Root, name))
		if err != nil {
			if os.IsNotExist(err) {
				surface[name] = false
				continue
			}
			return nil, err
		}
		surface[name] = info.IsDir()
	}
	return surface, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func currentBranch(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "ref: ") {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(line, "ref: "), "/")
	return parts[len(parts)-1]
}
