#!/usr/bin/env python3
"""Put jstack on this machine, or bring it up to date.

  setup.py [--agent auto|codex|claude|both] [--repo PATH] [--dest PATH]
           [--skill NAME ...] [--apply] [--install-tools]

Dry run by default. Steps: jstack skills, vendored skills, tool checks,
tool-owned skills, git hooks, report. Overwritten skills are backed up to
<dest>/.jstack-backup/<timestamp>/.
"""
import argparse, filecmp, io, json, os, re, shutil, subprocess, sys, tarfile, urllib.request
from datetime import datetime

def repo_root(explicit):
    if explicit: return os.path.abspath(explicit)
    if os.environ.get("JSTACK_REPO"): return os.path.abspath(os.environ["JSTACK_REPO"])
    here = os.path.dirname(os.path.abspath(__file__))
    return os.path.abspath(os.path.join(here, "..", "..", ".."))

def detect_agent():
    if os.environ.get("CLAUDECODE") or os.environ.get("CLAUDE_CODE"): return "claude"
    if os.environ.get("CODEX_HOME") or os.environ.get("CODEX_THREAD_ID"): return "codex"
    if os.environ.get("CURSOR_TRACE_ID") or os.environ.get("CURSOR_AGENT"): return "cursor"
    return "claude"

def dest_for(agent):
    if agent == "codex": return os.path.join(os.environ.get("CODEX_HOME", os.path.expanduser("~/.codex")), "skills")
    if agent == "cursor": return os.path.expanduser("~/.cursor/skills")
    return os.path.join(os.environ.get("CLAUDE_HOME", os.path.expanduser("~/.claude")), "skills")

def dir_same(a, b):
    cmp = filecmp.dircmp(a, b)
    if cmp.left_only or cmp.right_only or cmp.diff_files: return False
    return all(dir_same(os.path.join(a, d), os.path.join(b, d)) for d in cmp.common_dirs)

def plan(sources, dest, only):
    """sources: {name: source_dir}. Returns groups and the local-only list."""
    names = sorted(n for n in sources if not only or n in only)
    groups = {"new": [], "changed": [], "same": []}
    for n in names:
        target = os.path.join(dest, n)
        if not os.path.isdir(target): groups["new"].append(n)
        elif dir_same(sources[n], target): groups["same"].append(n)
        else: groups["changed"].append(n)
    local_only = sorted(d for d in os.listdir(dest) if os.path.isdir(os.path.join(dest, d)) and d not in sources and not d.startswith(".")) if os.path.isdir(dest) else []
    return groups, local_only

def apply(sources, dest, groups, stamp):
    backup = os.path.join(dest, ".jstack-backup", stamp)
    for n in groups["changed"]:
        os.makedirs(backup, exist_ok=True)
        shutil.move(os.path.join(dest, n), os.path.join(backup, n))
    for n in groups["new"] + groups["changed"]:
        shutil.copytree(sources[n], os.path.join(dest, n))
    return backup if groups["changed"] else None

def fetch_vendor(root, agent, cache):
    """Download each vendor.json entry for this harness into cache. Returns {name: dir}."""
    path = os.path.join(root, "vendor.json")
    if not os.path.isfile(path): return {}
    out = {}
    for entry in json.load(open(path))["skills"]:
        if agent not in entry["harness"]: continue
        name, repo, sub, ref = entry["name"], entry["repo"], entry["path"].strip("/"), entry["ref"]
        target = os.path.join(cache, f"{name}-{ref[:12]}")
        if not os.path.isdir(target):
            url = f"https://codeload.github.com/{repo}/tar.gz/{ref}"
            data = urllib.request.urlopen(url, timeout=60).read()
            with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as tar:
                members = [m for m in tar.getmembers() if "/" in m.name and m.name.split("/", 1)[1].startswith(sub + "/")]
                if not members: sys.exit(f"ERROR vendor {name}: {sub} not found in {repo}@{ref[:12]}")
                for m in members:
                    rel = m.name.split("/", 1)[1][len(sub) + 1:]
                    if not rel or m.isdir(): continue
                    dst = os.path.join(target, rel); os.makedirs(os.path.dirname(dst), exist_ok=True)
                    with open(dst, "wb") as f: f.write(tar.extractfile(m).read())
        out[name] = target
    return out

def parse_tools(root):
    """Sections of tools.md that carry a check line. Returns [(title, check, install, skill_install, skill_folder)]."""
    path = os.path.join(root, "tools.md")
    if not os.path.isfile(path): return []
    tools = []
    for section in re.split(r"^## ", open(path).read(), flags=re.M)[1:]:
        title = section.splitlines()[0].strip()
        get = lambda key: (re.search(rf"^- {key}: `([^`]+)`", section, re.M) or [None, None])[1]
        check = get("Check")
        if not check: continue
        inst = re.search(r"^- Install: (.+)$", section, re.M)
        install = inst.group(1).strip().strip("`") if inst else "see tools.md"
        tools.append((title, check, install, get("Skill install"), get("Skill folder")))
    return tools

INSTRUCTIONS = {"claude": "~/.claude/CLAUDE.md", "codex": "~/.codex/AGENTS.md", "cursor": "~/.cursor/rules/jstack.mdc"}
CURSOR_FRONTMATTER = "---\ndescription: jstack, how the human you work for works\nalwaysApply: true\n---\n\n"
START, END = "<!-- jstack:start -->", "<!-- jstack:end -->"

def install_instructions(root, agent, apply, keep_existing, stamp):
    """Put AGENTS.md into the harness's user-level instructions file as a marked block.

    Default: the file becomes the block, and whatever was there is backed up next to it.
    This is an opinionated stack, and two letters side by side is the drift it exists to prevent.
    --keep-instructions appends the block instead and leaves the rest of the file alone.
    On later runs only the text between the markers changes."""
    src = os.path.join(root, "AGENTS.md")
    if not os.path.isfile(src): return
    lead = CURSOR_FRONTMATTER if agent == "cursor" else ""
    block = f"{START}\n{open(src).read().rstrip()}\n{END}\n"
    path = os.path.expanduser(INSTRUCTIONS[agent])
    current = open(path).read() if os.path.isfile(path) else ""
    backup = None
    if START in current and END in current:
        head, rest = current.split(START, 1); _, tail = rest.split(END, 1)
        outside = (head + tail).strip()
        if outside and not keep_existing and outside != CURSOR_FRONTMATTER.strip():
            updated = lead + block; backup = f"{path}.bak-{stamp}"
            verb = "replaced, old file backed up" if apply else "would replace and back up the old file"
        else:
            updated = head + block.rstrip("\n") + tail
            verb = "left as is" if updated == current else ("updated" if apply else "would update")
    elif current.strip() and keep_existing:
        updated = current.rstrip("\n") + "\n\n" + block
        verb = "added" if apply else "would add"
    elif current.strip():
        updated = lead + block; backup = f"{path}.bak-{stamp}"
        verb = "replaced, old file backed up" if apply else "would replace and back up the old file"
    else:
        updated = lead + block
        verb = "created" if apply else "would create"
    if apply and updated != current:
        os.makedirs(os.path.dirname(path), exist_ok=True)
        if backup: shutil.copy2(path, backup)
        open(path, "w").write(updated)
    where = f" ({backup})" if backup and apply else ""
    print(f"  {agent}: {verb} {INSTRUCTIONS[agent]}{where}")

def run_ok(cmd):
    return subprocess.run(cmd, shell=True, capture_output=True).returncode == 0

def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--agent", default="auto", choices=["auto", "codex", "claude", "cursor", "both", "all"])
    p.add_argument("--repo"); p.add_argument("--dest"); p.add_argument("--skill", action="append")
    p.add_argument("--apply", action="store_true"); p.add_argument("--install-tools", action="store_true")
    p.add_argument("--keep-instructions", action="store_true", help="append the letter to an existing instructions file instead of replacing it")
    a = p.parse_args()
    root = repo_root(a.repo); src_skills = os.path.join(root, "skills")
    if not os.path.isdir(src_skills): sys.exit(f"ERROR no skills/ under {root}. Pass --repo or set JSTACK_REPO.")
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    cache = os.path.join(os.path.expanduser("~/.cache/jstack/vendor"))
    os.makedirs(cache, exist_ok=True)
    agents = {"both": ["codex", "claude"], "all": ["codex", "claude", "cursor"]}.get(a.agent) or [detect_agent() if a.agent == "auto" else a.agent]
    own = {n: os.path.join(src_skills, n) for n in os.listdir(src_skills) if os.path.isfile(os.path.join(src_skills, n, "SKILL.md"))}

    for agent in agents:
        dest = a.dest or dest_for(agent)
        os.makedirs(dest, exist_ok=True)
        vendor = fetch_vendor(root, agent, cache)
        sources = {**own, **vendor}
        groups, local_only = plan(sources, dest, a.skill)
        print(f"\n[{agent}] {dest}")
        for k in ("new", "changed", "same"): print(f"  {k:8} {', '.join(groups[k]) or '-'}")
        print(f"  {'vendor':8} {', '.join(sorted(vendor)) or '-'}")
        print(f"  {'local':8} {', '.join(local_only) or '-'} (untouched)")
        if a.apply and (groups["new"] or groups["changed"]):
            backup = apply(sources, dest, groups, stamp)
            print(f"  applied. backup: {backup or 'none needed'}")
            groups, _ = plan(sources, dest, a.skill)
            print(f"  remaining drift: {', '.join(groups['new'] + groups['changed']) or 'none'}")

    print("\ninstructions:")
    for agent in agents:
        install_instructions(root, agent, a.apply, a.keep_instructions, stamp)

    print("\ntools:")
    for title, check, inst, skill_cmd, skill_folder in parse_tools(root):
        if not run_ok(check):
            if a.install_tools and a.apply and not inst.startswith("see "):
                print(f"  installing {title}: {inst}")
                ok = subprocess.run(inst, shell=True).returncode == 0
                print(f"  {'installed' if ok else 'FAILED'} {title}")
                if not ok: continue
            else:
                print(f"  missing {title}. install: {inst}"); continue
        line = f"  ok {title}"
        if skill_cmd and skill_folder:
            shared = os.path.join(os.path.expanduser("~/.agents/skills"), skill_folder)
            present = all(os.path.isdir(os.path.join(a.dest or dest_for(ag), skill_folder)) or os.path.isdir(shared) for ag in agents)
            if present: line += ", skill present"
            elif a.apply:
                ok = subprocess.run(skill_cmd, shell=True, capture_output=True).returncode == 0
                line += f", skill {'installed' if ok else 'install FAILED'} via {skill_cmd}"
            else: line += f", skill missing, would run: {skill_cmd}"
        print(line)

    hooks = os.path.join(root, ".githooks")
    if os.path.isdir(hooks) and os.path.isdir(os.path.join(root, ".git")):
        if a.apply: subprocess.run(["git", "-C", root, "config", "core.hooksPath", ".githooks"]); print("\nhooks: installed")
        else: print("\nhooks: would install")
    if a.apply: print("\nrestart the harness so the skills load.")

if __name__ == "__main__": main()
