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
    return "claude"

def dest_for(agent):
    if agent == "codex": return os.path.join(os.environ.get("CODEX_HOME", os.path.expanduser("~/.codex")), "skills")
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
        tools.append((title, check, inst.group(1).strip() if inst else "see tools.md", get("Skill install"), get("Skill folder")))
    return tools

def run_ok(cmd):
    return subprocess.run(cmd, shell=True, capture_output=True).returncode == 0

def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--agent", default="auto", choices=["auto", "codex", "claude", "both"])
    p.add_argument("--repo"); p.add_argument("--dest"); p.add_argument("--skill", action="append")
    p.add_argument("--apply", action="store_true"); p.add_argument("--install-tools", action="store_true")
    a = p.parse_args()
    root = repo_root(a.repo); src_skills = os.path.join(root, "skills")
    if not os.path.isdir(src_skills): sys.exit(f"ERROR no skills/ under {root}. Pass --repo or set JSTACK_REPO.")
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    cache = os.path.join(os.path.expanduser("~/.cache/jstack/vendor"))
    os.makedirs(cache, exist_ok=True)
    agents = ["codex", "claude"] if a.agent == "both" else [detect_agent() if a.agent == "auto" else a.agent]
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
            present = all(os.path.isdir(os.path.join(a.dest or dest_for(ag), skill_folder)) for ag in agents)
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
