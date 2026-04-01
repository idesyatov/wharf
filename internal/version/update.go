package version

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const githubReleasesURL = "https://api.github.com/repos/idesyatov/wharf/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// CheckUpdate checks GitHub Releases for a newer version.
// Returns the new version tag and URL, or empty strings if up to date.
func CheckUpdate() (newVersion, url string) {
	if Version == "dev" {
		return "", ""
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(githubReleasesURL)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", ""
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", ""
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(Version, "v")

	if latest != current && latest > current {
		return release.TagName, release.HTMLURL
	}
	return "", ""
}
