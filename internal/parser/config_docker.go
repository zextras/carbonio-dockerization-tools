package parser

// DockerConfig contiene tutte le assunzioni e regole sui docker compose
// Modificare questo file quando cambiano i docker-compose files
type DockerConfig struct {
	// Servizi nascosti dalla UI (non mostrati all'utente)
	HiddenServices []string

	// Servizi sempre obbligatori (checkbox [●], non deselezionabili)
	RequiredServices []string

	// Servizi auto-inclusi (nascosti ma sempre aggiunti automaticamente)
	AutoIncludedServices []string

	// Mappa registrator → servizio parent
	// I registrator vengono auto-aggiunti quando si seleziona il parent
	RegistratorToService map[string]string

	// Tag che quando presenti rendono il tag non editabile
	LockedTagValues []string

	// Servizi specifici con tag sempre bloccato (override)
	ForcedLockedTagServices []string

	// UI specifiche con tag sempre bloccato (override)
	ForcedLockedTagUIs []string
}

// DefaultDockerConfig restituisce la configurazione di default
// basata sulla struttura attuale dei docker-compose files
func DefaultDockerConfig() *DockerConfig {
	return &DockerConfig{
		// Servizi nascosti dalla UI
		HiddenServices: []string{
			// === REGISTRATORS (auto-gestiti) ===
			// CE registrators
			"mailbox-registrator",
			"user-management-registrator",
			"catalog-registrator",
			"storages-registrator",
			"docs-connector-registrator",
			"docs-editor-registrator",
			"preview-registrator",
			"files-registrator",
			"tasks-registrator",
			"message-dispatcher-registrator",
			"wsc-registrator",
			// Advanced registrators
			"advanced-registrator",
			"address-book-registrator",
			"auth-registrator",

			// === SERVIZI AUTO-INCLUSI ===
			"carbonio-composed-ui", // Proxy, sempre necessario
		},

		// Servizi sempre obbligatori (non deselezionabili)
		RequiredServices: []string{
			"carbonio-mailbox",
			"carbonio-openldap",
			"carbonio-postfix",
			"carbonio-mariadb",
			"consul",
			"consul-register",
			"traefik",
			"memcached",
		},

		// Servizi auto-inclusi (nascosti ma sempre aggiunti)
		AutoIncludedServices: []string{
			"carbonio-composed-ui", // Il proxy è sempre necessario
		},

		// Mappa registrator → servizio parent
		RegistratorToService: map[string]string{
			// CE registrators
			"mailbox-registrator":            "carbonio-mailbox",
			"user-management-registrator":    "carbonio-user-management",
			"catalog-registrator":            "carbonio-catalog",
			"storages-registrator":           "carbonio-storages",
			"docs-connector-registrator":     "carbonio-docs-connector",
			"docs-editor-registrator":        "carbonio-docs-editor",
			"preview-registrator":            "carbonio-preview",
			"files-registrator":              "carbonio-files",
			"tasks-registrator":              "carbonio-tasks",
			"message-dispatcher-registrator": "carbonio-message-dispatcher",
			"wsc-registrator":                "carbonio-ws-collaboration",

			// Advanced registrators
			"advanced-registrator":     "carbonio-mailbox",
			"address-book-registrator": "carbonio-mailbox",
			"auth-registrator":         "carbonio-mailbox",
		},

		// Tag values che bloccano l'editing
		LockedTagValues: []string{
			"local", // Servizi build-only (non pullabili da registry)
		},

		// Servizi con tag sempre bloccato (per override espliciti)
		ForcedLockedTagServices: []string{
			// Aggiungi qui servizi specifici se necessario
		},

		// UI con tag sempre bloccato (per override espliciti)
		ForcedLockedTagUIs: []string{
			// Aggiungi qui UI specifiche se necessario
		},
	}
}

// GlobalDockerConfig è l'istanza globale della configurazione
var GlobalDockerConfig = DefaultDockerConfig()

// === HELPER FUNCTIONS ===

// IsServiceHidden controlla se un servizio deve essere nascosto dalla UI
func (dc *DockerConfig) IsServiceHidden(serviceName string) bool {
	for _, hidden := range dc.HiddenServices {
		if serviceName == hidden {
			return true
		}
	}
	return false
}

// IsServiceRequired controlla se un servizio è obbligatorio
func (dc *DockerConfig) IsServiceRequired(serviceName string) bool {
	for _, required := range dc.RequiredServices {
		if serviceName == required {
			return true
		}
	}
	return false
}

// IsServiceAutoIncluded controlla se un servizio viene auto-incluso
func (dc *DockerConfig) IsServiceAutoIncluded(serviceName string) bool {
	for _, autoIncluded := range dc.AutoIncludedServices {
		if serviceName == autoIncluded {
			return true
		}
	}
	return false
}

// IsRegistrator controlla se un servizio è un registrator
func (dc *DockerConfig) IsRegistrator(serviceName string) bool {
	_, exists := dc.RegistratorToService[serviceName]
	return exists
}

// GetParentService restituisce il servizio parent di un registrator
func (dc *DockerConfig) GetParentService(registratorName string) string {
	return dc.RegistratorToService[registratorName]
}

// GetRegistratorsForService restituisce tutti i registrator per un servizio
func (dc *DockerConfig) GetRegistratorsForService(serviceName string) []string {
	var registrators []string
	for regName, parentService := range dc.RegistratorToService {
		if parentService == serviceName {
			registrators = append(registrators, regName)
		}
	}
	return registrators
}

// IsTagLocked controlla se il tag di un servizio deve essere bloccato
func (dc *DockerConfig) IsTagLocked(serviceName, tag string, isBackend bool) bool {
	// Check forced locked services/UIs
	if isBackend {
		for _, locked := range dc.ForcedLockedTagServices {
			if serviceName == locked {
				return true
			}
		}
	} else {
		for _, locked := range dc.ForcedLockedTagUIs {
			if serviceName == locked {
				return true
			}
		}
	}

	// Check tag value rules
	for _, lockedTag := range dc.LockedTagValues {
		if tag == lockedTag {
			return true
		}
	}

	return false
}

// === BACKWARD COMPATIBILITY ===
// Queste funzioni mantengono la compatibilità con il codice esistente

// IsServiceRequired (funzione globale per compatibilità)
func IsServiceRequired(serviceName string) bool {
	return GlobalDockerConfig.IsServiceRequired(serviceName)
}

// IsRegistrator (funzione globale per compatibilità)
func IsRegistrator(serviceName string) bool {
	return GlobalDockerConfig.IsRegistrator(serviceName)
}

// GetParentServiceFromMap (funzione globale per compatibilità)
func GetParentServiceFromMap(registratorName string) string {
	return GlobalDockerConfig.GetParentService(registratorName)
}

// GetRegistratorsForService (funzione globale per compatibilità)
func GetRegistratorsForService(serviceName string) []string {
	return GlobalDockerConfig.GetRegistratorsForService(serviceName)
}

// === NOTE PER FUTURE MODIFICHE ===
//
// 1. RIMUOVERE LOGICA REGISTRATOR:
//    - Svuota la mappa RegistratorToService
//    - Rimuovi i registrator da HiddenServices
//    - Il codice in services.go smetterà automaticamente di auto-aggiungere registrator
//
// 2. CAMBIARE SERVIZI REQUIRED:
//    - Modifica la lista RequiredServices
//    - I servizi required hanno checkbox [●] e non sono deselezionabili
//
// 3. AGGIUNGERE SERVIZI NASCOSTI:
//    - Aggiungi a HiddenServices
//    - Se devono essere inclusi automaticamente, aggiungi anche a AutoIncludedServices
//
// 4. BLOCCARE/SBLOCCARE TAG:
//    - Per bloccare tutti i servizi con un certo tag: aggiungi a LockedTagValues
//    - Per bloccare un servizio specifico: aggiungi a ForcedLockedTagServices/ForcedLockedTagUIs
//
// 5. PROXY/UI OBBLIGATORIE:
//    - Le UI proxy sono marcate come required nel parsing (IsProxy = true)
//    - Non c'è bisogno di configurazione aggiuntiva
