package tools

import "testing"

const sample = "# Tools\n\nIntro with a `- Check:` mention that is not a section.\n\n" +
	"## git and gh\n\n- Check: `command -v git && gh auth status`\n- Install: `brew install git gh`, then `gh auth login`\n\n" +
	"## The work tracker\n\nProse.\n\n**Quest** (current)\n- Check: `command -v quest`\n- Install: `curl -fsSL https://x/install.sh | sh`\n- Skill install: `quest skill install`\n- Skill folder: `quest`\n\n" +
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
	if git.SkillInstall != "" || git.SkillFolder != "" {
		t.Fatalf("git skill = %q %q", git.SkillInstall, git.SkillFolder)
	}
	quest := parsed[1]
	if quest.Title != "The work tracker" || quest.Command != "curl -fsSL https://x/install.sh | sh" || quest.SkillInstall != "quest skill install" || quest.SkillFolder != "quest" {
		t.Fatalf("quest = %+v", quest)
	}
	bare := parsed[2]
	if bare.Install != "see tools.md" || bare.Command != "" {
		t.Fatalf("bare = %+v", bare)
	}
}

func TestParseEmptyText(t *testing.T) {
	if got := Parse(""); len(got) != 0 {
		t.Fatalf("parsed %+v", got)
	}
}
