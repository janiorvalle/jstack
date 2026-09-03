// Package letter puts AGENTS.md into a harness's instructions file as a marked
// block, and decides what happens to whatever else the file holds.
package letter

import "strings"

const (
	Start = "<!-- jstack:start -->"
	End   = "<!-- jstack:end -->"
)

// Outcome names what Plan decided to do with the instructions file.
type Outcome string

const (
	// Create writes a new file.
	Create Outcome = "create"
	// Update changes only the text between the markers.
	Update Outcome = "update"
	// Same leaves the file as it is.
	Same Outcome = "same"
	// Replace makes the file the letter and backs up what was there.
	Replace Outcome = "replace"
	// Append keeps the file's own content and adds the block after it.
	Append Outcome = "append"
)

// Change is the decided outcome and the file content after it.
type Change struct {
	Outcome Outcome
	Content string
}

// Block wraps the letter in the markers.
func Block(text string) string {
	return Start + "\n" + strings.TrimRight(text, "\n") + "\n" + End + "\n"
}

// Plan decides how the letter lands in a file that currently holds current.
// lead is text the harness needs at the top of the file, such as frontmatter.
// This is an opinionated stack, so a file with other content becomes the letter
// and the old file is backed up, unless keepExisting appends the block instead.
// A file that already carries the markers only changes between them.
func Plan(current, text, lead string, keepExisting bool) Change {
	block := Block(text)
	startAt := strings.Index(current, Start)
	endAt := strings.Index(current, End)
	if startAt >= 0 && endAt > startAt {
		head := current[:startAt]
		tail := current[endAt+len(End):]
		outside := strings.TrimSpace(head + tail)
		if outside != "" && !keepExisting && outside != strings.TrimSpace(lead) {
			return Change{Outcome: Replace, Content: lead + block}
		}
		updated := head + strings.TrimRight(block, "\n") + tail
		if updated == current {
			return Change{Outcome: Same, Content: current}
		}
		return Change{Outcome: Update, Content: updated}
	}
	if strings.TrimSpace(current) == "" {
		return Change{Outcome: Create, Content: lead + block}
	}
	if keepExisting {
		return Change{Outcome: Append, Content: strings.TrimRight(current, "\n") + "\n\n" + block}
	}
	return Change{Outcome: Replace, Content: lead + block}
}
