package provisioner

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Account struct {
	Username string
	Password string
	IsAdmin  bool
}

func ParseProvisioningScript(workDir string) ([]Account, error) {
	scriptPath := filepath.Join(workDir, "provisioning", "provisioning.sh")
	log.Printf("Parsing provisioning script: %s", scriptPath)

	file, err := os.Open(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open provisioning script: %w", err)
	}
	defer file.Close()

	var accounts []Account
	scanner := bufio.NewScanner(file)
	inHeredoc := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "<<EOF") {
			inHeredoc = true
			continue
		}
		if strings.Contains(line, "EOF") {
			inHeredoc = false
			continue
		}

		if !inHeredoc {
			continue
		}

		if !strings.HasPrefix(line, "ca ") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			log.Printf("Skipping malformed account line: %s", line)
			continue
		}

		username := parts[1]
		password := parts[2]
		isAdmin := strings.Contains(line, "zimbraIsAdminAccount TRUE")

		accounts = append(accounts, Account{
			Username: username,
			Password: password,
			IsAdmin:  isAdmin,
		})

		log.Printf("Found account: %s (admin=%v)", username, isAdmin)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading provisioning script: %w", err)
	}

	log.Printf("Parsed %d accounts from provisioning script", len(accounts))
	return accounts, nil
}
