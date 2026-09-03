package letter

import "testing"

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
		{name: "block plus other content is replaced", current: "# mine\n" + Block("old"), outcome: Replace, content: block},
		{name: "block plus other content is updated with keep", current: "# mine\n" + Block("old"), keep: true, outcome: Update, content: "# mine\n" + block},
		{name: "block plus kept content that matches is same", current: "# mine\n" + block, keep: true, outcome: Same, content: "# mine\n" + block},
		{name: "lead outside the block is not other content", current: "---\nalwaysApply: true\n---\n\n" + Block("old"), lead: "---\nalwaysApply: true\n---\n\n", outcome: Update, content: "---\nalwaysApply: true\n---\n\n" + block},
		{name: "lead is written on create", current: "", lead: "lead\n", outcome: Create, content: "lead\n" + block},
		{name: "lead is written on replace", current: "# mine\n", lead: "lead\n", outcome: Replace, content: "lead\n" + block},
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

func TestBlockNormalizesTrailingNewlines(t *testing.T) {
	if Block("x\n\n\n") != Start+"\nx\n"+End+"\n" {
		t.Fatalf("block = %q", Block("x\n\n\n"))
	}
}
