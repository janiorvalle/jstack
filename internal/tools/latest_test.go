package tools

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func respond(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func lookupWith(t *testing.T, want string, status int, body string) Lookup {
	t.Helper()
	return Lookup{Client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != want {
			t.Errorf("url = %q, want %q", request.URL, want)
		}
		if request.Header.Get("User-Agent") == "" {
			t.Error("no User-Agent, GitHub rejects those")
		}
		return respond(status, body), nil
	})}}
}

func TestLatestReadsTheGitHubReleaseTag(t *testing.T) {
	lookup := lookupWith(t, "https://api.github.com/repos/janiorvalle/roast/releases/latest", http.StatusOK, `{"tag_name":"v0.2.7","assets":[]}`)
	got, err := lookup.Latest(context.Background(), Tool{Title: "roast", Repo: "https://github.com/janiorvalle/roast", Command: "curl -fsSL https://x/install.sh | sh"})
	if err != nil || got != "v0.2.7" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestLatestReadsTheNpmVersionForAnNpmInstall(t *testing.T) {
	lookup := lookupWith(t, "https://registry.npmjs.org/agent-browser/latest", http.StatusOK, `{"name":"agent-browser","version":"0.36.0"}`)
	got, err := lookup.Latest(context.Background(), Tool{Title: "agent-browser", Repo: "https://github.com/vercel-labs/agent-browser", Command: "npm install -g agent-browser && agent-browser install"})
	if err != nil || got != "v0.36.0" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestLatestFailuresAreErrorsNotVersions(t *testing.T) {
	roast := Tool{Title: "roast", Repo: "https://github.com/janiorvalle/roast"}
	for name, lookup := range map[string]Lookup{
		"rate limited":   lookupWith(t, "https://api.github.com/repos/janiorvalle/roast/releases/latest", http.StatusForbidden, `{"message":"API rate limit exceeded"}`),
		"not json":       lookupWith(t, "https://api.github.com/repos/janiorvalle/roast/releases/latest", http.StatusOK, `<html>`),
		"no tag":         lookupWith(t, "https://api.github.com/repos/janiorvalle/roast/releases/latest", http.StatusOK, `{"assets":[]}`),
		"tag not semver": lookupWith(t, "https://api.github.com/repos/janiorvalle/roast/releases/latest", http.StatusOK, `{"tag_name":"nightly"}`),
		"network down": {Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: connection refused")
		})}},
	} {
		if got, err := lookup.Latest(context.Background(), roast); err == nil || got != "" {
			t.Errorf("%s: got %q, err %v", name, got, err)
		}
	}
}

func TestLatestWithNoSourceIsAnError(t *testing.T) {
	lookup := Lookup{Client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		t.Errorf("unexpected request to %s", request.URL)
		return nil, errors.New("no")
	})}}
	if got, err := lookup.Latest(context.Background(), Tool{Title: "git and gh", Command: "brew install git gh"}); err == nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}
