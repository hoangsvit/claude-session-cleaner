package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const npmPackage = "claude-cleaner"

// CheckLatestVersion queries the npm registry and returns the latest published
// version and whether it is newer than current.
func CheckLatestVersion(current string) (latest string, hasUpdate bool) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://registry.npmjs.org/" + npmPackage + "/latest")
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return "", false
	}

	latest = strings.TrimPrefix(pkg.Version, "v")
	current = strings.TrimPrefix(current, "v")
	if latest == "" || latest == current {
		return latest, false
	}
	return latest, semverGT(latest, current)
}

// semverGT returns true if a > b using simple major.minor.patch comparison.
func semverGT(a, b string) bool {
	pa := splitVer(a)
	pb := splitVer(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return true
		}
		if pa[i] < pb[i] {
			return false
		}
	}
	return false
}

func splitVer(v string) [3]int {
	// strip any pre-release suffix (e.g. "1.2.0-beta" → "1.2.0")
	v = strings.SplitN(v, "-", 2)[0]
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}
