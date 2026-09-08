package deploy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/8bit-interactive/pocketbase-pockethost-skills/internal/keys"
	"github.com/8bit-interactive/pocketbase-pockethost-skills/internal/project"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	DefaultHost          = "ftp.pockethost.io"
	DefaultPort          = 2222
	SyncStateName        = ".ftp-deploy-sync-state.json"
	SyncStateVersion     = "1.0.0"
	SyncStateDescription = "DO NOT DELETE THIS FILE. It tracks the files managed by pocketbase-pockethost."
)

type Options struct {
	Project         project.Project
	Environment     string
	DryRun          bool
	CleanSlate      bool
	Yes             bool
	Host            string
	Port            int
	Username        string
	PrivateKeyPath  string
	PrivateKeyValue string
	HostKey         string
	KnownHosts      string
	Instance        string
}

type Entry struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
	Hash string `json:"hash,omitempty"`
}

type State struct {
	Description   string  `json:"description"`
	Version       string  `json:"version"`
	GeneratedTime int64   `json:"generatedTime"`
	Data          []Entry `json:"data"`
}

type Result struct {
	Uploaded  []string
	Replaced  []string
	Deleted   []string
	Unchanged []string
}

func Run(options Options) (Result, error) {
	if options.Instance == "" {
		return Result{}, errors.New("missing POCKETHOST_INSTANCE_NAME or configured instanceName")
	}
	if options.Username == "" {
		return Result{}, errors.New("missing POCKETHOST_SFTP_USERNAME")
	}
	signer, err := loadSigner(options)
	if err != nil {
		return Result{}, err
	}
	callback, err := hostKeyCallback(options)
	if err != nil {
		return Result{}, err
	}
	localEntries, err := collectEntries(options.Project.Root)
	if err != nil {
		return Result{}, err
	}
	localState := State{Description: SyncStateDescription, Version: SyncStateVersion, GeneratedTime: time.Now().UnixMilli(), Data: localEntries}

	sshConfig := &ssh.ClientConfig{
		User:            options.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: callback,
		Timeout:         30 * time.Second,
	}
	address := net.JoinHostPort(options.Host, fmt.Sprintf("%d", options.Port))
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return Result{}, fmt.Errorf("connect to %s: %w", address, err)
	}
	defer client.Close()
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return Result{}, fmt.Errorf("start SFTP session: %w", err)
	}
	defer sftpClient.Close()

	remoteRoot := path.Join("/", options.Instance)
	if err := sftpClient.MkdirAll(remoteRoot); err != nil {
		return Result{}, fmt.Errorf("create remote instance directory: %w", err)
	}
	if options.CleanSlate {
		if !options.Yes {
			return Result{}, errors.New("--clean-slate requires --yes")
		}
		if !options.DryRun {
			if err := cleanManagedRoots(sftpClient, remoteRoot); err != nil {
				return Result{}, err
			}
		}
	}
	remoteState, err := readState(sftpClient, path.Join(remoteRoot, SyncStateName))
	if err != nil && !os.IsNotExist(err) {
		return Result{}, fmt.Errorf("read remote sync state: %w", err)
	}
	result := diff(localState.Data, remoteState.Data)
	printResult(result, options.DryRun)
	if options.DryRun {
		return result, nil
	}
	if err := apply(sftpClient, options.Project.Root, remoteRoot, localState, remoteState, result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func loadSigner(options Options) (ssh.Signer, error) {
	data := []byte(options.PrivateKeyValue)
	if len(data) == 0 && options.PrivateKeyPath != "" {
		var err error
		data, err = os.ReadFile(options.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key: %w", err)
		}
	}
	if len(data) == 0 {
		return nil, errors.New("missing POCKETHOST_SFTP_PRIVATE_KEY_PATH or POCKETHOST_SFTP_PRIVATE_KEY")
	}
	data = []byte(strings.ReplaceAll(string(data), `\n`, "\n"))
	return keys.Signer(data)
}

func hostKeyCallback(options Options) (ssh.HostKeyCallback, error) {
	if options.HostKey != "" {
		expected := options.HostKey
		return func(_ string, _ net.Addr, key ssh.PublicKey) error {
			if expected == ssh.FingerprintSHA256(key) || strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))) == strings.TrimSpace(expected) {
				return nil
			}
			return fmt.Errorf("unexpected SFTP host key %s", ssh.FingerprintSHA256(key))
		}, nil
	}
	knownHostsPath := options.KnownHosts
	if knownHostsPath == "" {
		if configDir, err := os.UserHomeDir(); err == nil {
			knownHostsPath = filepath.Join(configDir, ".ssh", "known_hosts")
		}
	}
	if knownHostsPath != "" {
		if _, err := os.Stat(knownHostsPath); err == nil {
			return knownhosts.New(knownHostsPath)
		}
	}
	return nil, errors.New("SFTP host key is not pinned; set POCKETHOST_SFTP_HOST_KEY or provide --known-hosts")
}

func collectEntries(root string) ([]Entry, error) {
	var entries []Entry
	for _, top := range []string{"pb_public", "pb_hooks", "pb_migrations"} {
		directory := filepath.Join(root, top)
		if _, err := os.Stat(directory); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		err := filepath.Walk(directory, func(localPath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			relative, err := filepath.Rel(root, localPath)
			if err != nil {
				return err
			}
			name := filepath.ToSlash(relative)
			if info.IsDir() {
				entries = append(entries, Entry{Type: "folder", Name: name})
				return nil
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			hash, err := fileHash(localPath)
			if err != nil {
				return err
			}
			entries = append(entries, Entry{Type: "file", Name: name, Size: info.Size(), Hash: hash})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries, nil
}

func fileHash(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func readState(client *sftp.Client, filename string) (State, error) {
	file, err := client.Open(filename)
	if err != nil {
		return State{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse sync state: %w", err)
	}
	return state, nil
}

func diff(local, remote []Entry) Result {
	localByName := map[string]Entry{}
	remoteByName := map[string]Entry{}
	for _, entry := range local {
		localByName[entry.Name] = entry
	}
	for _, entry := range remote {
		remoteByName[entry.Name] = entry
	}
	var result Result
	for _, entry := range local {
		previous, exists := remoteByName[entry.Name]
		if !exists || entry.Type != previous.Type {
			if exists && entry.Type != previous.Type {
				result.Deleted = append(result.Deleted, stateName(previous))
			}
			if entry.Type == "file" {
				result.Uploaded = append(result.Uploaded, entry.Name)
			} else {
				result.Uploaded = append(result.Uploaded, entry.Name+"/")
			}
			continue
		}
		if entry.Type == "file" && entry.Hash != previous.Hash {
			result.Replaced = append(result.Replaced, entry.Name)
		} else {
			result.Unchanged = append(result.Unchanged, entry.Name)
		}
	}
	for _, entry := range remote {
		if _, exists := localByName[entry.Name]; !exists {
			result.Deleted = append(result.Deleted, stateName(entry))
		}
	}
	sort.Strings(result.Uploaded)
	sort.Strings(result.Replaced)
	sort.Strings(result.Deleted)
	sort.Strings(result.Unchanged)
	return result
}

func stateName(entry Entry) string {
	if entry.Type == "folder" {
		return strings.TrimSuffix(entry.Name, "/") + "/"
	}
	return entry.Name
}

func apply(client *sftp.Client, root, remoteRoot string, local, remote State, result Result) error {
	deleted := append([]string(nil), result.Deleted...)
	sort.SliceStable(deleted, func(i, j int) bool {
		leftDepth := strings.Count(deleted[i], "/")
		rightDepth := strings.Count(deleted[j], "/")
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		return deleted[i] > deleted[j]
	})
	for _, name := range deleted {
		if name == SyncStateName || strings.HasPrefix(name, "pb_data/") {
			continue
		}
		remotePath := path.Join(remoteRoot, name)
		if strings.HasSuffix(name, "/") {
			if err := client.RemoveDirectory(remotePath); err != nil {
				return err
			}
		} else if err := client.Remove(remotePath); err != nil {
			return err
		}
	}
	for _, name := range append(append([]string{}, result.Uploaded...), result.Replaced...) {
		if strings.HasSuffix(name, "/") {
			continue
		}
		localPath := filepath.Join(root, filepath.FromSlash(name))
		remotePath := path.Join(remoteRoot, name)
		if err := client.MkdirAll(path.Dir(remotePath)); err != nil {
			return err
		}
		input, err := os.Open(localPath)
		if err != nil {
			return err
		}
		output, err := client.Create(remotePath)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		inputErr := input.Close()
		outputErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		if inputErr != nil {
			return inputErr
		}
		if outputErr != nil {
			return outputErr
		}
	}
	stateData, err := json.MarshalIndent(local, "", "  ")
	if err != nil {
		return err
	}
	statePath := path.Join(remoteRoot, SyncStateName)
	temporaryPath := statePath + ".tmp"
	file, err := client.Create(temporaryPath)
	if err != nil {
		return err
	}
	_, writeErr := io.Copy(file, bytes.NewReader(stateData))
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return client.Rename(temporaryPath, statePath)
}

func cleanManagedRoots(client *sftp.Client, remoteRoot string) error {
	for _, name := range []string{"pb_public", "pb_hooks", "pb_migrations"} {
		remotePath := path.Join(remoteRoot, name)
		if _, err := client.Stat(remotePath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := removeTree(client, remotePath); err != nil {
			return err
		}
	}
	return nil
}

func removeTree(client *sftp.Client, remotePath string) error {
	entries, err := client.ReadDir(remotePath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := path.Join(remotePath, entry.Name())
		if entry.IsDir() {
			if err := removeTree(client, child); err != nil {
				return err
			}
		} else if err := client.Remove(child); err != nil {
			return err
		}
	}
	return client.RemoveDirectory(remotePath)
}

func printResult(result Result, dryRun bool) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	for _, name := range result.Uploaded {
		fmt.Printf("%screate %s\n", prefix, name)
	}
	for _, name := range result.Replaced {
		fmt.Printf("%sreplace %s\n", prefix, name)
	}
	for _, name := range result.Deleted {
		fmt.Printf("%sdelete %s\n", prefix, name)
	}
}
