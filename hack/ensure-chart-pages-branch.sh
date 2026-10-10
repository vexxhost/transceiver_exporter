#!/usr/bin/env bash
set -euo pipefail

remote="${1:-origin}"
pages_branch="${2:-gh-pages}"

fetch_pages_branch() {
  git fetch "$remote" "+refs/heads/${pages_branch}:refs/remotes/${remote}/${pages_branch}"
}

if git ls-remote --exit-code --heads "$remote" "$pages_branch" >/dev/null 2>&1; then
  fetch_pages_branch
  exit 0
fi

printf 'Initializing %s/%s for chart-releaser\n' "$remote" "$pages_branch"

nojekyll_blob=$(printf "" | git hash-object -w --stdin)
readme_blob=$({
  printf "# Helm charts\n\n"
  printf "This branch is managed by chart-releaser.\n"
} | git hash-object -w --stdin)

tree=$({
  printf "100644 blob %s\t.nojekyll\n" "$nojekyll_blob"
  printf "100644 blob %s\tREADME.md\n" "$readme_blob"
} | git mktree)

commit=$(
  GIT_AUTHOR_NAME="${GIT_AUTHOR_NAME:-github-actions[bot]}" \
    GIT_AUTHOR_EMAIL="${GIT_AUTHOR_EMAIL:-41898282+github-actions[bot]@users.noreply.github.com}" \
    GIT_COMMITTER_NAME="${GIT_COMMITTER_NAME:-github-actions[bot]}" \
    GIT_COMMITTER_EMAIL="${GIT_COMMITTER_EMAIL:-41898282+github-actions[bot]@users.noreply.github.com}" \
    git -c commit.gpgsign=false commit-tree "$tree" -m "Initialize Helm chart repository"
)

if ! git push "$remote" "$commit:refs/heads/${pages_branch}"; then
  if ! git ls-remote --exit-code --heads "$remote" "$pages_branch" >/dev/null 2>&1; then
    printf 'Failed to initialize %s/%s\n' "$remote" "$pages_branch" >&2
    exit 1
  fi
fi

fetch_pages_branch
