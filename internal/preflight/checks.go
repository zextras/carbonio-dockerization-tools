package preflight

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const RegistryHost = "registry.dev.zextras.com:443"
const MinDockerComposeVersion = "2.30.0"

func CheckDockerComposeVersion() error {
	cmd := exec.Command("docker", "compose", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker compose not found or not working: %w", err)
	}

	versionStr := string(output)
	log.Printf("Docker Compose version output: %s", versionStr)

	re := regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 4 {
		return fmt.Errorf("unable to parse docker compose version from: %s", versionStr)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	currentVersion := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	log.Printf("Detected Docker Compose version: %s", currentVersion)

	minParts := strings.Split(MinDockerComposeVersion, ".")
	minMajor, _ := strconv.Atoi(minParts[0])
	minMinor, _ := strconv.Atoi(minParts[1])
	minPatch, _ := strconv.Atoi(minParts[2])

	if major < minMajor {
		return fmt.Errorf("docker compose version %s is too old (minimum required: %s)", currentVersion, MinDockerComposeVersion)
	}
	if major == minMajor && minor < minMinor {
		return fmt.Errorf("docker compose version %s is too old (minimum required: %s)", currentVersion, MinDockerComposeVersion)
	}
	if major == minMajor && minor == minMinor && patch < minPatch {
		return fmt.Errorf("docker compose version %s is too old (minimum required: %s)", currentVersion, MinDockerComposeVersion)
	}

	return nil
}

func CheckRegistryConnectivity() error {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", RegistryHost, timeout)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()
	return nil
}

// DetectNATIP returns the local IP address used to reach the registry host.
// This is typically the VPN interface IP, since the registry is only reachable via VPN.
func DetectNATIP() (string, error) {
	conn, err := net.Dial("udp", RegistryHost)
	if err != nil {
		return "", fmt.Errorf("failed to detect NAT IP: %w", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := localAddr.IP.String()
	log.Printf("Detected NAT IP: %s", ip)
	return ip, nil
}
