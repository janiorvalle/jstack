#!/usr/bin/env python3
"""Checks a ticket has the shape the tracker skill asks for, before it's filed.

Reads the ticket from a file path, or from stdin when the path is - or absent.
Exits 0 when the ticket has a Problem and a Fix with words in them, a Done when
with one to four lines that each start with "- ", and at most 120 words in all.
Frontmatter doesn't count. Each failure prints a code, what was wrong, and what
to write instead.
"""
import sys

WORD_CAP = 120
DONE_WHEN_CAP = 4
REQUIRED = ("Problem:", "Fix:", "Done when:")
LABELS = REQUIRED + ("Out of scope:",)
SHAPE = """Problem: Two sentences, from the person who hits it.
Fix: Two or three sentences.
Done when:
- Up to four observable lines.
Out of scope: Optional, one line."""


def read_ticket(argv):
    if len(argv) > 1 and argv[1] != "-":
        return open(argv[1]).read()
    if sys.stdin.isatty():
        sys.exit("usage: ticket-lint.py <ticket.md>, or pipe the ticket on stdin")
    return sys.stdin.read()


def strip_frontmatter(text):
    if not text.startswith("---\n"):
        return text
    _, _, rest = text[4:].partition("\n---\n")
    return rest


def sections(text):
    """Each label maps to its lines: the rest of the label's own line, then every
    line until the next label. Text before the first label is dropped."""
    found = {}
    current = None
    for line in strip_frontmatter(text).splitlines():
        line = line.strip()
        label = next((label for label in LABELS if line.startswith(label)), None)
        if label:
            current = label
            found[label] = [line[len(label):].strip()]
        elif current:
            found[current].append(line)
    return {label: [line for line in lines if line] for label, lines in found.items()}


def word_count(text):
    return len(strip_frontmatter(text).split())


def missing(parts):
    return [
        f"TICKET-MISSING-LABEL: no line starts with '{label}'. "
        f"Every ticket has Problem, Fix, and Done when. The shape:\n{SHAPE}"
        for label in REQUIRED if label not in parts
    ]


def empty(parts):
    return [
        f"TICKET-EMPTY-SECTION: nothing after '{label}'. "
        "Problem is two sentences from the person who hits it. Fix is two or three."
        for label in ("Problem:", "Fix:") if label in parts and not parts[label]
    ]


def done_when(parts):
    if "Done when:" not in parts:
        return []
    lines = parts["Done when:"]
    if not lines:
        return ["TICKET-EMPTY-DONE-WHEN: nothing under 'Done when:'. "
                "Write one to four lines a reviewer can observe, one per line, starting with '- '."]
    problems = []
    for line in lines:
        if not line.startswith("- "):
            problems.append(f"TICKET-DONE-WHEN-NOT-A-LIST: '{line[:60]}' under 'Done when:' doesn't start with '- '. "
                            "One observable outcome per line, each starting with '- '.")
    if len(lines) > DONE_WHEN_CAP:
        problems.append(f"TICKET-DONE-WHEN-TOO-LONG: {len(lines)} lines under 'Done when:', the cap is {DONE_WHEN_CAP}. "
                        "Keep the ones a reviewer can observe and drop the ones that restate the fix.")
    return problems


def too_long(words):
    if words <= WORD_CAP:
        return []
    return [f"TICKET-TOO-LONG: {words} words, the cap is {WORD_CAP}. "
            "Cut the narrative. Problem is two sentences from the person who hits it, "
            "Fix is two or three. Design notes go in the PR, not the ticket."]


def lint(text):
    parts = sections(text)
    return missing(parts) + empty(parts) + done_when(parts) + too_long(word_count(text))


def main():
    text = read_ticket(sys.argv)
    problems = lint(text)
    if problems:
        print("\n\n".join(problems))
        sys.exit(1)
    print(f"ticket: ok, {word_count(text)} words")


if __name__ == "__main__":
    main()
