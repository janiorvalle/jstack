package tools

import "testing"

const sample = "# Tools\n\nIntro with a `- Check:` mention that is not a section.\n\n" +
	"## git and gh\n\n- Check: `command -v git && gh auth status`\n- Install: `brew install git gh`, then `gh auth login`\n\n" +
	"## The work tracker\n\nProse.\n\n**Quest** (current)\n- Repo: https://github.com/x/quest\n- Check: `command -v quest`\n- Version: `quest --version`\n- Install: `curl -fsSL https://x/install.sh | sh`\n- Skill install: `quest skill install`\n- Skill folder: `quest`\n\n" +
	"## no check here\n\n- Install: `something`\n\n" +
	"## bare\n\n- Check: `command -v bare`\n"

func TestParseKeepsSectionsWithACheckLine(t *testing.T) {
	parsed := Parse(sample)
	if len(parsed) != 3 {
		t.Fatalf("parsed %d tools, want 3: %+v", len(parsed), parsed)
	}
	git := parsed[0]
	if git.Title != "git and gh" || git.Check != "command -v git && gh auth status" {
		t.Fatalf("git = %+v", git)
	}
	if git.Install != "brew install git gh, then gh auth login" || git.Command != "brew install git gh && gh auth login" {
		t.Fatalf("git install = %q, command = %q", git.Install, git.Command)
	}
	if git.SkillInstall != "" || git.SkillFolder != "" || git.Version != "" || git.Repo != "" {
		t.Fatalf("git skill = %q %q, version = %q, repo = %q", git.SkillInstall, git.SkillFolder, git.Version, git.Repo)
	}
	quest := parsed[1]
	if quest.Title != "The work tracker" || quest.Command != "curl -fsSL https://x/install.sh | sh" || quest.SkillInstall != "quest skill install" || quest.SkillFolder != "quest" {
		t.Fatalf("quest = %+v", quest)
	}
	if quest.Version != "quest --version" || quest.Repo != "https://github.com/x/quest" {
		t.Fatalf("quest version = %q, repo = %q", quest.Version, quest.Repo)
	}
	bare := parsed[2]
	if bare.Install != "see https://github.com/janiorvalle/jstack/blob/main/tools.md#bare" || bare.Command != "" {
		t.Fatalf("bare = %+v", bare)
	}
}

func TestPrerequisiteLinksToItsOwnHeading(t *testing.T) {
	prerequisite := "## git and gh\n\nProse.\n\n- Check: `command -v git`\n- macOS: `brew install git gh`\n- Then `gh auth login`\n"
	parsed := Parse(prerequisite)
	if len(parsed) != 1 {
		t.Fatalf("parsed %d tools, want 1: %+v", len(parsed), parsed)
	}
	if parsed[0].Command != "" {
		t.Fatalf("a prerequisite has a command to run: %q", parsed[0].Command)
	}
	if parsed[0].Install != "see https://github.com/janiorvalle/jstack/blob/main/tools.md#git-and-gh" {
		t.Fatalf("install = %q", parsed[0].Install)
	}
}

func TestParseReadsThePinFromAnNpmInstallLine(t *testing.T) {
	for line, want := range map[string]string{
		"`npm install -g agent-browser@0.36.0 && agent-browser install`": "v0.36.0",
		"`npm install -g agent-browser && agent-browser install`":        "",
		"`npm install -g agent-browser@latest`":                          "",
		"`curl -fsSL https://x/install.sh | sh`":                         "",
	} {
		parsed := Parse("## t\n\n- Check: `command -v t`\n- Install: " + line + "\n")
		if got := parsed[0].Pin; got != want {
			t.Errorf("Pin for %s = %q, want %q", line, got, want)
		}
	}
}

func TestParseEmptyText(t *testing.T) {
	if got := Parse(""); len(got) != 0 {
		t.Fatalf("parsed %+v", got)
	}
}

func TestParseVersionFindsTheVersionInWhatToolsPrint(t *testing.T) {
	for output, want := range map[string]string{
		"quest 0.24.0\n":                 "v0.24.0",
		"0.2.5":                          "v0.2.5",
		"bgr 1.7.0":                      "v1.7.0",
		"tokenomnom version 0.6.6":       "v0.6.6",
		"agent-browser 0.27.0":           "v0.27.0",
		"v1.2.3-rc.1":                    "v1.2.3-rc.1",
		"roast 1.2.3 (commit abc1234)\n": "v1.2.3",
		"usage: roast [command]":         "",
		"":                               "",
		"go version go1.24":              "",
		"version 01.2.3 is not semver":   "",
	} {
		if got := ParseVersion(output); got != want {
			t.Errorf("ParseVersion(%q) = %q, want %q", output, got, want)
		}
	}
}

func TestOutdatedNeedsBothVersions(t *testing.T) {
	for name, tc := range map[string]struct {
		installed, latest string
		want              bool
	}{
		"behind":            {"v1.0.0", "v1.1.0", true},
		"behind by a patch": {"v1.7.0", "v1.7.1", true},
		"current":           {"v1.1.0", "v1.1.0", false},
		"ahead":             {"v1.2.0", "v1.1.0", false},
		"prerelease behind": {"v1.1.0-rc.1", "v1.1.0", true},
		"latest unknown":    {"v1.0.0", "", false},
		"installed unknown": {"", "v1.1.0", false},
		"both unknown":      {"", "", false},
	} {
		if got := Outdated(tc.installed, tc.latest); got != tc.want {
			t.Errorf("%s: Outdated(%q, %q) = %v, want %v", name, tc.installed, tc.latest, got, tc.want)
		}
	}
}

func TestDisplayDropsTheV(t *testing.T) {
	if got := Display("v1.2.3"); got != "1.2.3" {
		t.Fatalf("Display = %q", got)
	}
}
