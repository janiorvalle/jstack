#!/usr/bin/env python3
"""Checks a ticket has the shape the tracker skill asks for, before it's filed.

Reads the ticket from a file path, or from stdin when the path is - or absent.
Exits 0 when the ticket has a Problem, a Fix, and a Done when label, one to four
Done when lines, and at most 120 words. Frontmatter doesn't count. Each failure
prints a code, what was wrong, and what to write instead.
"""
import sys

WORD_CAP = 120
DONE_WHEN_CAP = 4
LABELS = ("Problem:", "Fix:", "Done when:")
OPTIONAL_LABEL = "Out of scope:"
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


def body_lines(text):
    return [line.strip() for line in strip_frontmatter(text).splitlines()]


def missing_labels(lines):
    return [label for label in LABELS if not any(line.startswith(label) for line in lines)]


def done_when_lines(lines):
    inside = False
    count = 0
    for line in lines:
        if line.startswith("Done when:"):
            inside = True
            continue
        if line.startswith(LABELS) or line.startswith(OPTIONAL_LABEL):
            inside = False
        if inside and line:
            count += 1
    return count


def word_count(lines):
    return sum(len(line.split()) for line in lines)


def problems(text):
    lines = body_lines(text)
    found = []
    for label in missing_labels(lines):
        found.append(
            f"TICKET-MISSING-LABEL: no line starts with '{label}'. "
            f"Every ticket has Problem, Fix, and Done when. The shape:\n{SHAPE}"
        )
    if "Done when:" not in missing_labels(lines):
        count = done_when_lines(lines)
        if count == 0:
            found.append(
                "TICKET-EMPTY-DONE-WHEN: nothing under 'Done when:'. "
                "Write one to four lines a reviewer can observe, one per line, starting with '- '."
            )
        elif count > DONE_WHEN_CAP:
            found.append(
                f"TICKET-DONE-WHEN-TOO-LONG: {count} lines under 'Done when:', the cap is {DONE_WHEN_CAP}. "
                "Keep the ones a reviewer can observe and drop the ones that restate the fix."
            )
    words = word_count(lines)
    if words > WORD_CAP:
        found.append(
            f"TICKET-TOO-LONG: {words} words, the cap is {WORD_CAP}. "
            "Cut the narrative. Problem is two sentences from the person who hits it, "
            "Fix is two or three. Files to touch and design notes go in the PR, not the ticket."
        )
    return found, words


def main():
    found, words = problems(read_ticket(sys.argv))
    if found:
        print("\n\n".join(found))
        sys.exit(1)
    print(f"ticket: ok, {words} words")


if __name__ == "__main__":
    main()
