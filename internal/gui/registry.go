package gui

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type dockerConfig struct {
	Auths map[string]struct {
		Auth string `json:"auth"`
	} `json:"auths"`
	CredsStore string `json:"credsStore"`
}

type tagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

const registryHost = "registry.dev.zextras.com"

func IsOurRegistry(imageBase string) bool {
	return strings.HasPrefix(imageBase, registryHost+"/")
}

// CheckRegistryAuth verifies that we can authenticate to the registry.
// Returns nil on success, or an error describing what went wrong.
func CheckRegistryAuth() error {
	auth, err := readDockerAuth(registryHost)
	if err != nil {
		return fmt.Errorf("cannot read Docker credentials for %s: %w\n\nRun \"docker login %s\" in a terminal first.", registryHost, err, registryHost)
	}

	// Verify credentials with a lightweight /v2/ ping
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/v2/", registryHost), nil)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach registry %s: %w", registryHost, err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Docker credentials for %s are invalid or expired.\n\nRun \"docker login %s\" to refresh them.", registryHost, registryHost)
	}

	return nil
}

func FetchTags(imageBase string) ([]string, error) {
	parts := strings.SplitN(imageBase, "/", 2)
	if len(parts) != 2 {
		return nil, nil
	}
	host := parts[0]
	name := parts[1]

	if host != registryHost {
		return nil, nil
	}

	auth, err := readDockerAuth(host)
	if err != nil {
		return nil, fmt.Errorf("failed to read docker auth for %s: %w", host, err)
	}

	url := fmt.Sprintf("https://%s/v2/%s/tags/list", host, name)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed for %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, name)
	}

	var result tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response for %s: %w", name, err)
	}

	return sortTags(result.Tags), nil
}

func readDockerAuth(host string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get home dir: %w", err)
	}

	configPath := filepath.Join(home, ".docker", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("cannot read docker config: %w", err)
	}

	var cfg dockerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("cannot parse docker config: %w", err)
	}

	// Try inline auth first
	if entry, ok := cfg.Auths[host]; ok && entry.Auth != "" {
		return entry.Auth, nil
	}

	// Try credential helper (credsStore: "osxkeychain", "desktop", etc.)
	if cfg.CredsStore != "" {
		auth, err := readFromCredentialHelper(cfg.CredsStore, host)
		if err != nil {
			return "", fmt.Errorf("credential helper %q failed for %s: %w", cfg.CredsStore, host, err)
		}
		return auth, nil
	}

	return "", fmt.Errorf("no auth for host %s (no inline token and no credsStore configured)", host)
}

// readFromCredentialHelper invokes docker-credential-<helper> get
// and returns the base64-encoded "user:password" token.
func readFromCredentialHelper(helper, host string) (string, error) {
	helperBin := "docker-credential-" + helper
	cmd := exec.Command(helperBin, "get")
	cmd.Stdin = strings.NewReader("https://" + host)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %v (stderr: %s)", helperBin, err, strings.TrimSpace(stderr.String()))
	}

	var creds struct {
		Username string `json:"Username"`
		Secret   string `json:"Secret"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &creds); err != nil {
		return "", fmt.Errorf("cannot parse %s output: %w", helperBin, err)
	}

	if creds.Username == "" || creds.Secret == "" {
		return "", fmt.Errorf("%s returned empty credentials for %s", helperBin, host)
	}

	token := base64.StdEncoding.EncodeToString([]byte(creds.Username + ":" + creds.Secret))
	return token, nil
}

func sortTags(tags []string) []string {
	var priority []string
	var semver []string
	var rest []string

	for _, t := range tags {
		switch t {
		case "devel", "latest":
			priority = append(priority, t)
		default:
			if looksLikeSemver(t) {
				semver = append(semver, t)
			} else {
				rest = append(rest, t)
			}
		}
	}

	sort.Slice(priority, func(i, j int) bool {
		order := map[string]int{"devel": 0, "latest": 1}
		return order[priority[i]] < order[priority[j]]
	})

	sort.Sort(sort.Reverse(sort.StringSlice(semver)))
	sort.Strings(rest)

	result := make([]string, 0, len(tags))
	result = append(result, priority...)
	result = append(result, semver...)
	result = append(result, rest...)
	return result
}

func looksLikeSemver(tag string) bool {
	if len(tag) == 0 {
		return false
	}
	for _, c := range tag {
		if c != '.' && !(c >= '0' && c <= '9') {
			return false
		}
	}
	return strings.Contains(tag, ".")
}
