#!/usr/bin/env python3
"""Keep the vendored skills in skills/ in step with their upstream folders.

  vendor-bump.py bump <name>     copy the skill at its upstream head and move the pin, if upstream moved
  vendor-bump.py restore <name>  copy the skill at its pinned ref, for a fresh or suspect checkout

A skill's version is the last upstream commit that touched its folder, not the
repo head, so an unrelated upstream commit never moves the pin. Both commands
replace skills/<name> wholesale, so upstream deletions land too, and put the
upstream license file next to SKILL.md. bump prints one JSON object for the
workflow to read: name, repo, path, old, new, changed.

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
        sys.exit(f"ERROR {entry['name']}: no commit on {entry['repo']} touches {entry['path']}")
    return commits[0]["sha"]

def copy_from_upstream(entry, ref):
    """Replace skills/<name> with the upstream folder at ref, license file alongside."""
    tarball = github(f"https://codeload.github.com/{entry['repo']}/tar.gz/{ref}")
    folder = entry["path"].strip("/") + "/"
    target = os.path.join(SKILLS, entry["name"])
    shutil.rmtree(target, ignore_errors=True)
    found = False
    with tarfile.open(fileobj=io.BytesIO(tarball), mode="r:gz") as tar:
        for member in tar.getmembers():
            inside = member.name.split("/", 1)[1] if "/" in member.name else ""
            if inside == entry["license_file"] and member.isfile():
                write(os.path.join(target, "LICENSE"), tar.extractfile(member).read(), member.mode)
            if not inside.startswith(folder) or not member.isfile():
                continue
            found = True
            write(os.path.join(target, inside[len(folder):]), tar.extractfile(member).read(), member.mode)
    if not found:
        sys.exit(f"ERROR {entry['name']}: {entry['path']} is not in {entry['repo']} at {ref[:12]}")
    if not os.path.isfile(os.path.join(target, "LICENSE")):
        sys.exit(f"ERROR {entry['name']}: {entry['license_file']} is not in {entry['repo']} at {ref[:12]}")

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
    if new != old:
        copy_from_upstream(entry, new)
        entry["ref"] = new
        with open(VENDOR, "w") as f:
            json.dump(vendor, f, indent=2)
            f.write("\n")
    print(json.dumps({"name": name, "repo": entry["repo"], "path": entry["path"], "old": old, "new": new, "changed": new != old}))

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
