#!/usr/bin/env python3
"""Install or update jstack skills into a harness's skills folder, then check tools.

  sync.py [--agent auto|codex|claude|both] [--repo PATH] [--dest PATH] [--skill NAME ...] [--apply]

Dry run by default. Backs up overwritten skills to <dest>/.jstack-backup/<timestamp>/.
"""
import argparse, filecmp, os, re, shutil, subprocess, sys
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

def plan(src_skills, dest, only):
    names = sorted(d for d in os.listdir(src_skills) if os.path.isfile(os.path.join(src_skills, d, "SKILL.md")))
    if only: names = [n for n in names if n in only]
    groups = {"new": [], "changed": [], "same": []}
    for n in names:
        target = os.path.join(dest, n)
        if not os.path.isdir(target): groups["new"].append(n)
        elif dir_same(os.path.join(src_skills, n), target): groups["same"].append(n)
        else: groups["changed"].append(n)
    local_only = sorted(d for d in os.listdir(dest) if os.path.isdir(os.path.join(dest, d)) and d not in names) if os.path.isdir(dest) else []
    return groups, local_only

def apply(src_skills, dest, groups):
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    backup = os.path.join(dest, ".jstack-backup", stamp)
    for n in groups["changed"]:
        os.makedirs(backup, exist_ok=True)
        shutil.move(os.path.join(dest, n), os.path.join(backup, n))
    for n in groups["new"] + groups["changed"]:
        shutil.copytree(os.path.join(src_skills, n), os.path.join(dest, n))
    return backup if groups["changed"] else None

def check_tools(root):
    tools_md = os.path.join(root, "tools.md")
    if not os.path.isfile(tools_md): return []
    text = open(tools_md).read()
    missing = []
    for section in re.split(r"^## ", text, flags=re.M)[1:]:
        title = section.splitlines()[0].strip()
        m = re.search(r"- Check: `([^`]+)`", section)
        if not m: continue
        ok = subprocess.run(m.group(1), shell=True, capture_output=True).returncode == 0
        if not ok:
            inst = re.search(r"- Install: (.+)", section)
            missing.append((title, inst.group(1).strip() if inst else "see tools.md"))
    return missing

def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--agent", default="auto", choices=["auto", "codex", "claude", "both"])
    p.add_argument("--repo"); p.add_argument("--dest"); p.add_argument("--skill", action="append"); p.add_argument("--apply", action="store_true")
    a = p.parse_args()
    root = repo_root(a.repo); src_skills = os.path.join(root, "skills")
    if not os.path.isdir(src_skills): sys.exit(f"ERROR no skills/ under {root}. Pass --repo or set JSTACK_REPO.")
    agents = ["codex", "claude"] if a.agent == "both" else [detect_agent() if a.agent == "auto" else a.agent]
    for agent in agents:
        dest = a.dest or dest_for(agent)
        os.makedirs(dest, exist_ok=True)
        groups, local_only = plan(src_skills, dest, a.skill)
        print(f"\n[{agent}] {dest}")
        for k in ("new", "changed", "same"): print(f"  {k:8} {', '.join(groups[k]) or '-'}")
        print(f"  {'local':8} {', '.join(local_only) or '-'} (untouched)")
        if a.apply and (groups["new"] or groups["changed"]):
            backup = apply(src_skills, dest, groups)
            print(f"  applied. backup: {backup or 'none needed'}")
            groups, _ = plan(src_skills, dest, a.skill)
            print(f"  remaining drift: {', '.join(groups['new'] + groups['changed']) or 'none'}")
    missing = check_tools(root)
    print("\ntools:", "all present" if not missing else "")
    for title, inst in missing: print(f"  missing {title}. install: {inst}")
    if a.apply: print("\nrestart the harness so the skills load.")

if __name__ == "__main__": main()
