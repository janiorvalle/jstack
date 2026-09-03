#!/usr/bin/env python3
"""Keep the pinned npm tools in tools.md in step with the npm registry.

  tool-bump.py list            the packages tools.md pins, as a JSON list, for the workflow matrix
  tool-bump.py bump <package>  move the pin to npm's latest, if npm moved

A pinned tool is one whose Install line is `npm install -g <package>@<version>`.
The pin is the version setup installs and compares against, so moving it is a
PR, the same as a vendored skill. bump rewrites that one version in tools.md and
prints one JSON object for the workflow to read: name, repo, old, new, changed.
repo is the section's Repo line, so the PR can link the upstream release.
"""
import json, os, re, sys, urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TOOLS = os.path.join(ROOT, "tools.md")
PINNED = re.compile(r"^- Install: `npm install -g ([^@\s`]+)@(\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)", re.M)
REPO = re.compile(r"^- Repo: (\S+)$", re.M)

def npm_latest(package):
    request = urllib.request.Request(f"https://registry.npmjs.org/{package}/latest", headers={"User-Agent": "jstack-tool-bump"})
    return json.load(urllib.request.urlopen(request, timeout=60))["version"]

def sections(text):
    """Each `## ` section of tools.md, so a pin is read next to its Repo line."""
    return re.split(r"(?m)^## ", text)[1:]

def pins(text):
    return {name: version for section in sections(text) for name, version in PINNED.findall(section)}

def repo_of(text, package):
    for section in sections(text):
        if package in dict(PINNED.findall(section)):
            match = REPO.search(section)
            return match.group(1) if match else ""
    return ""

def list_pinned():
    print(json.dumps(sorted(pins(open(TOOLS).read()))))

def bump(package):
    text = open(TOOLS).read()
    pinned = pins(text)
    if package not in pinned:
        sys.exit(f"ERROR tools.md pins no npm package named {package}. Pinned: {', '.join(sorted(pinned)) or 'none'}. The install line must read `npm install -g {package}@<version>`.")
    old, new = pinned[package], npm_latest(package)
    if new != old:
        text = text.replace(f"npm install -g {package}@{old}", f"npm install -g {package}@{new}")
        open(TOOLS, "w").write(text)
    print(json.dumps({"name": package, "repo": repo_of(text, package), "old": old, "new": new, "changed": new != old}))

def main():
    if len(sys.argv) == 2 and sys.argv[1] == "list":
        return list_pinned()
    if len(sys.argv) == 3 and sys.argv[1] == "bump":
        return bump(sys.argv[2])
    sys.exit(__doc__.strip())

if __name__ == "__main__":
    main()
