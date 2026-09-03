#!/usr/bin/env python3
"""Keep the vendored skills in skills/ in step with their upstream folders.

  vendor-bump.py bump <name>     copy the skill at its upstream head and move the pin, if upstream moved
  vendor-bump.py restore <name>  copy the skill at its pinned ref, for a fresh or suspect checkout

A skill's version is the last upstream commit that touched its folder, not the
repo head, so an unrelated upstream commit never moves the pin. Both commands
replace skills/<name> wholesale, so upstream deletions land too, and put the
license file from that same commit next to SKILL.md, since that is the license
governing the copied text. bump always copies, so a hand edit to a vendored file
shows up as a change even when the pin didn't move. It prints one JSON object
for the workflow to read: name, repo, path, old, new, changed.

Set GITHUB_TOKEN to raise the API rate limit. The vendor-bump workflow does.
"""
import io, json, os, shutil, sys, tarfile, urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
VENDOR = os.path.join(ROOT, "vendor.json")
SKILLS = os.path.join(ROOT, "skills")

def github(url):
    headers = {"User-Agent": "jstack-vendor-bump"}
    token = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
    if token:
        headers["Authorization"] = f"Bearer {token}"
    return urllib.request.urlopen(urllib.request.Request(url, headers=headers), timeout=60).read()

def upstream_head(entry):
    """The last commit on the default branch that touched the skill folder."""
    commits = json.loads(github(f"https://api.github.com/repos/{entry['repo']}/commits?path={entry['path']}&per_page=1"))
    if not commits:
        sys.exit(f"ERROR {entry['name']}: no commit on {entry['repo']} touches {entry['path']}. Upstream moved or removed the folder. Fix the path in vendor.json, or drop the entry and the skill.")
    return commits[0]["sha"]

def copy_from_upstream(entry, ref):
    """Replace skills/<name> with the upstream folder at ref, license file alongside. Returns whether any file changed."""
    tarball = github(f"https://codeload.github.com/{entry['repo']}/tar.gz/{ref}")
    folder = entry["path"].strip("/") + "/"
    target = os.path.join(SKILLS, entry["name"])
    staging = target + ".incoming"
    shutil.rmtree(staging, ignore_errors=True)
    with tarfile.open(fileobj=io.BytesIO(tarball), mode="r:gz") as tar:
        for member in tar.getmembers():
            inside = member.name.split("/", 1)[1] if "/" in member.name else ""
            if inside == entry["license_file"] and member.isfile():
                write(os.path.join(staging, "LICENSE"), tar.extractfile(member).read(), member.mode)
            if inside.startswith(folder) and member.isfile():
                write(os.path.join(staging, inside[len(folder):]), tar.extractfile(member).read(), member.mode)
    if not os.path.isfile(os.path.join(staging, "SKILL.md")):
        sys.exit(f"ERROR {entry['name']}: no SKILL.md under {entry['path']} in {entry['repo']} at {ref[:12]}. "
                 "Upstream moved or removed the folder. Fix the path in vendor.json, or drop the entry and the skill.")
    if not os.path.isfile(os.path.join(staging, "LICENSE")):
        sys.exit(f"ERROR {entry['name']}: {entry['license_file']} is not in {entry['repo']} at {ref[:12]}. Fix license_file in vendor.json.")
    changed = tree(staging) != tree(target)
    shutil.rmtree(target, ignore_errors=True)
    os.rename(staging, target)
    return changed

def tree(folder):
    files = {}
    for dirpath, _, names in os.walk(folder):
        for name in names:
            path = os.path.join(dirpath, name)
            files[os.path.relpath(path, folder)] = open(path, "rb").read()
    return files

def write(path, data, mode):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(data)
    os.chmod(path, mode & 0o777)

def load():
    return json.load(open(VENDOR))

def entry_named(vendor, name):
    for entry in vendor["skills"]:
        if entry["name"] == name:
            return entry
    sys.exit(f"ERROR no vendored skill named {name}. vendor.json has: {', '.join(e['name'] for e in vendor['skills'])}")

def bump(name):
    vendor = load()
    entry = entry_named(vendor, name)
    old, new = entry["ref"], upstream_head(entry)
    changed = copy_from_upstream(entry, new) or new != old
    if new != old:
        entry["ref"] = new
        with open(VENDOR, "w") as f:
            json.dump(vendor, f, indent=2)
            f.write("\n")
    print(json.dumps({"name": name, "repo": entry["repo"], "path": entry["path"], "old": old, "new": new, "changed": changed}))

def restore(name):
    entry = entry_named(load(), name)
    copy_from_upstream(entry, entry["ref"])
    print(f"restored skills/{name} from {entry['repo']}@{entry['ref'][:12]}")

def main():
    commands = {"bump": bump, "restore": restore}
    if len(sys.argv) != 3 or sys.argv[1] not in commands:
        sys.exit(__doc__.strip())
    commands[sys.argv[1]](sys.argv[2])

if __name__ == "__main__":
    main()
