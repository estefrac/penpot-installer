package updater

import (
	"encoding/json"
	"net/http"
	"time"
)

const latestReleaseURL = "https://api.github.com/repos/estefrac/penpot-installer/releases/latest"

// CheckResult es el resultado de verificar si hay una versión nueva
type CheckResult struct {
	HasUpdate      bool
	LatestVersion  string
	CurrentVersion string
}

// Check consulta la GitHub API y compara con la versión actual.
// Tiene timeout de 3 segundos para no bloquear el arranque.
func Check(currentVersion string) CheckResult {
	result := CheckResult{CurrentVersion: currentVersion}

	if currentVersion == "dev" {
		return result
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(latestReleaseURL)
	if err != nil {
		return result
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return result
	}

	result.LatestVersion = release.TagName
	result.HasUpdate = release.TagName != "" && release.TagName != currentVersion

	return result
}
