#!/usr/bin/env python3
"""Rebuild the skill index inside skills/jstack-mode/SKILL.md from each skill's description line.

Run after adding, renaming, or re-describing any skill. The index lives between
<!-- index:start --> and <!-- index:end --> and is never edited by hand.
"""
import os, re, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SKILLS = os.path.join(ROOT, "skills")
MODE = os.path.join(SKILLS, "jstack-mode", "SKILL.md")

def description(path):
    text = open(path).read()
    m = re.search(r'^description:\s*"?(.*?)"?\s*$', text, re.M)
    if not m:
        sys.exit(f"ERROR {path} has no description line")
    return m.group(1).replace('\\"', '"')

def build():
    principles, workflows = [], []
    for name in sorted(os.listdir(SKILLS)):
        if name == "jstack-mode":
            continue
        skill = os.path.join(SKILLS, name, "SKILL.md")
        if not os.path.isfile(skill):
            continue
        row = f"- **{name}**. {description(skill)}"
        (principles if re.search(r"^kind: principle$", open(skill).read(), re.M) else workflows).append(row)
    return "### Principles\n\n" + "\n".join(principles) + "\n\n### Workflows\n\n" + "\n".join(workflows)

def main():
    index = build()
    if not os.path.isfile(MODE):
        print(index); return
    text = open(MODE).read()
    start, end = "<!-- index:start -->", "<!-- index:end -->"
    if start not in text or end not in text:
        sys.exit(f"ERROR {MODE} is missing the index markers")
    head, rest = text.split(start, 1)
    _, tail = rest.split(end, 1)
    open(MODE, "w").write(f"{head}{start}\n{index}\n{end}{tail}")
    print(f"index rebuilt, {index.count(chr(10)+chr(45))} skills")

if __name__ == "__main__":
    main()
