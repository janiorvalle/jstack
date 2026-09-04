package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Lookup finds the newest published version of a tool. Our own tools come
// from the GitHub releases of their Repo line; a tool installed with npm
// comes from the npm registry. One HTTP call per tool, so a run of setup
// makes as many calls as tools.md has Version lines, six today, well inside
// GitHub's sixty an hour for unauthenticated callers.
type Lookup struct {
	Client *http.Client
}

var (
	githubRepo = regexp.MustCompile(`^https://github\.com/([^/\s]+/[^/\s]+?)/?$`)
	npmPackage = regexp.MustCompile(`^npm install -g (\S+)`)
)

const maximumMetadata = 1 << 20

// Latest returns the newest version as "v1.2.3", or an error when the lookup
// failed or the tool has no known source. Setup shows a failure as "latest
// unknown" and carries on, so the error is for the log, not the human.
func (lookup Lookup) Latest(ctx context.Context, tool Tool) (string, error) {
	if match := npmPackage.FindStringSubmatch(tool.Command); match != nil {
		return lookup.fetch(ctx, "https://registry.npmjs.org/"+match[1]+"/latest", "version")
	}
	if match := githubRepo.FindStringSubmatch(tool.Repo); match != nil {
		return lookup.fetch(ctx, "https://api.github.com/repos/"+match[1]+"/releases/latest", "tag_name")
	}
	return "", fmt.Errorf("%s has no Repo line and no npm install line, so there is nowhere to look up its latest version", tool.Title)
}

func (lookup Lookup) fetch(ctx context.Context, url, field string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "jstack-setup")
	response, err := lookup.Client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s returned %s", url, response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maximumMetadata))
	if err != nil {
		return "", err
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(content, &metadata); err != nil {
		return "", fmt.Errorf("%s returned invalid JSON: %w", url, err)
	}
	var value string
	if err := json.Unmarshal(metadata[field], &value); err != nil {
		return "", fmt.Errorf("%s has no %s string: %w", url, field, err)
	}
	version := ParseVersion(strings.TrimSpace(value))
	if version == "" {
		return "", fmt.Errorf("%s has %s %q, which is not a version", url, field, value)
	}
	return version, nil
}
