package letter

import (
	"strings"
	"testing"
)

const text = "# letter\n\nhello\n"

func TestPlanDecidesFromWhatTheFileHolds(t *testing.T) {
	block := Block(text)
	cases := []struct {
		name    string
		current string
		lead    string
		keep    bool
		outcome Outcome
		content string
	}{
		{name: "empty file is created", current: "", outcome: Create, content: block},
		{name: "blank file is created", current: "  \n", outcome: Create, content: block},
		{name: "same block is left alone", current: block, outcome: Same, content: block},
		{name: "old block is updated between the markers", current: Block("old letter"), outcome: Update, content: block},
		{name: "other content is replaced", current: "# mine\n", outcome: Replace, content: block},
		{name: "other content is kept with keep", current: "# mine\n", keep: true, outcome: Append, content: "# mine\n\n" + block},
		{name: "block plus kept content only changes between the markers", current: "# mine\n" + Block("old"), outcome: Update, content: "# mine\n" + block},
		{name: "block plus kept content with keep is the same", current: "# mine\n" + Block("old"), keep: true, outcome: Update, content: "# mine\n" + block},
		{name: "block plus kept content that matches is same", current: "# mine\n" + block, outcome: Same, content: "# mine\n" + block},
		{name: "lead outside the block is not other content", current: "---\nalwaysApply: true\n---\n\n" + Block("old"), lead: "---\nalwaysApply: true\n---\n\n", outcome: Update, content: "---\nalwaysApply: true\n---\n\n" + block},
		{name: "lead is written on create", current: "", lead: "lead\n", outcome: Create, content: "lead\n" + block},
		{name: "lead is written on replace", current: "# mine\n", lead: "lead\n", outcome: Replace, content: "lead\n" + block},
		{name: "keep on a file that starts with the lead appends", current: "lead\nmine\n", lead: "lead\n", keep: true, outcome: Append, content: "lead\nmine\n\n" + block},
		{name: "a marked file that lost the lead is replaced", current: "---\nalwaysApply: false\n---\n" + Block("old"), lead: "lead\n", outcome: Replace, content: "lead\n" + block},
		{name: "keep on a file without the lead is replaced", current: "---\nalwaysApply: false\n---\nmine\n", lead: "lead\n", keep: true, outcome: Replace, content: "lead\n" + block},
		{name: "a CRLF file with an old block is updated and written with LF", current: "# mine\r\n" + strings.ReplaceAll(Block("old"), "\n", "\r\n"), outcome: Update, content: "# mine\n" + block},
		{name: "a CRLF file with the current block is the same", current: strings.ReplaceAll(block, "\n", "\r\n"), outcome: Same, content: block},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Plan(tc.current, text, tc.lead, tc.keep)
			if got.Outcome != tc.outcome {
				t.Fatalf("outcome = %s, want %s", got.Outcome, tc.outcome)
			}
			if got.Content != tc.content {
				t.Fatalf("content = %q, want %q", got.Content, tc.content)
			}
		})
	}
}

func TestBlockLeavesTheTrackerLineToTheRepo(t *testing.T) {
	letter := "# letter\n\nTracker: markdown tasks/\n\nhello\n"
	if Block(letter) != Start+"\n# letter\n\nhello\n"+End+"\n" {
		t.Fatalf("block = %q", Block(letter))
	}
}

func TestBlockNormalizesTrailingNewlines(t *testing.T) {
	if Block("x\n\n\n") != Start+"\nx\n"+End+"\n" {
		t.Fatalf("block = %q", Block("x\n\n\n"))
	}
}
