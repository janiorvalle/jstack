#!/bin/sh
# List open PRs across every GitHub repo checked out under a directory.
# Output: repo|#number|title|url|author, one line per PR, sorted by repo.
#
#   scan.sh [--dir DIR] [--org ORG] [--exclude PATTERN ...]
#
# Defaults come from ~/.config/jstack/review-prs.json when present, else
# dir=~/code, no org filter, no excludes. Flags override the file.
set -eu

CONFIG="$HOME/.config/jstack/review-prs.json"
DIR=""; ORG=""; EXCLUDES=""
if [ -f "$CONFIG" ]; then
  DIR=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get('dir',''))" "$CONFIG")
  ORG=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get('org',''))" "$CONFIG")
  EXCLUDES=$(python3 -c "import json,sys;print(' '.join(json.load(open(sys.argv[1])).get('exclude',[])))" "$CONFIG")
fi
while [ $# -gt 0 ]; do
  case "$1" in
    --dir) DIR="$2"; shift 2 ;;
    --org) ORG="$2"; shift 2 ;;
    --exclude) EXCLUDES="$EXCLUDES $2"; shift 2 ;;
    *) echo "unknown flag $1" >&2; exit 2 ;;
  esac
done
DIR="${DIR:-$HOME/code}"
DIR=$(eval echo "$DIR")

gh auth status >/dev/null 2>&1 || { echo "gh is not authenticated. Run: gh auth login" >&2; exit 1; }

excluded() {
  for pattern in $EXCLUDES; do
    case "$1" in $pattern) return 0 ;; esac
  done
  return 1
}

for repo_dir in "$DIR"/*/; do
  [ -d "$repo_dir/.git" ] || continue
  remote=$(git -C "$repo_dir" remote get-url origin 2>/dev/null) || continue
  case "$remote" in *github.com*) ;; *) continue ;; esac
  org_repo=$(printf '%s' "$remote" | sed -E 's|.*github\.com[:/]||; s|\.git$||')
  owner="${org_repo%%/*}"; name="${org_repo#*/}"
  [ -n "$ORG" ] && [ "$owner" != "$ORG" ] && continue
  excluded "$name" && continue
  gh pr list --repo "$org_repo" --state open --json number,title,url,author \
    --jq ".[] | \"$name|#\(.number)|\(.title)|\(.url)|\(.author.login)\"" 2>/dev/null || true
done | sort -t'|' -k1,1 -k2,2
