package gui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type dockerConfig struct {
	Auths map[string]struct {
		Auth string `json:"auth"`
	} `json:"auths"`
}

type tagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

const registryHost = "registry.dev.zextras.com"

func IsOurRegistry(imageBase string) bool {
	return strings.HasPrefix(imageBase, registryHost+"/")
}

func FetchTags(imageBase string) []string {
	parts := strings.SplitN(imageBase, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	host := parts[0]
	name := parts[1]

	if host != registryHost {
		return nil
	}

	auth, err := readDockerAuth(host)
	if err != nil {
		log.Printf("FetchTags: failed to read docker auth for %s: %v", host, err)
		return nil
	}

	url := fmt.Sprintf("https://%s/v2/%s/tags/list", host, name)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("FetchTags: failed to create request: %v", err)
		return nil
	}
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("FetchTags: request failed for %s: %v", name, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("FetchTags: unexpected status %d for %s", resp.StatusCode, name)
		return nil
	}

	var result tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("FetchTags: failed to decode response for %s: %v", name, err)
		return nil
	}

	return sortTags(result.Tags)
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

	entry, ok := cfg.Auths[host]
	if !ok {
		return "", fmt.Errorf("no auth entry for host %s", host)
	}

	if entry.Auth == "" {
		return "", fmt.Errorf("empty auth for host %s", host)
	}

	return entry.Auth, nil
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
