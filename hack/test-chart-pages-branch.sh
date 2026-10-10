#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ensure_script="$script_dir/ensure-chart-pages-branch.sh"

tmpdir=$(mktemp -d)
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

remote="$tmpdir/remote.git"
repo="$tmpdir/repo"

git init --bare -b main "$remote" >/dev/null
git init -b main "$repo" >/dev/null

(
  cd "$repo"

  git config user.name "Test User"
  git config user.email "test@example.com"

  printf "main\n" > README.md
  git add README.md
  git -c commit.gpgsign=false commit -m "Initial commit" >/dev/null
  git remote add origin "$remote"
  git push -u origin main >/dev/null 2>&1

  "$ensure_script" origin gh-pages >/dev/null 2>&1
  git rev-parse --verify refs/remotes/origin/gh-pages >/dev/null
  git ls-remote --exit-code --heads origin gh-pages >/dev/null
  git cat-file -e origin/gh-pages:.nojekyll
  git show origin/gh-pages:README.md | grep -q "managed by chart-releaser"

  initialized_commit=$(git rev-parse origin/gh-pages)

  git update-ref -d refs/remotes/origin/gh-pages
  "$ensure_script" origin gh-pages >/dev/null 2>&1
  git rev-parse --verify refs/remotes/origin/gh-pages >/dev/null

  refetched_commit=$(git rev-parse origin/gh-pages)
  test "$initialized_commit" = "$refetched_commit"

  "$ensure_script" origin gh-pages >/dev/null 2>&1

  idempotent_commit=$(git rev-parse origin/gh-pages)
  test "$initialized_commit" = "$idempotent_commit"
)
