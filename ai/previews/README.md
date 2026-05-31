# ai/previews

Rendered captures used by the AI docs ([`../README.md`](../README.md),
[`../IMPLEMENTED.md`](../IMPLEMENTED.md), [`../BRAIN_PROTOCOL.md`](../BRAIN_PROTOCOL.md))
and the gallery ([`../showcase.html`](../showcase.html)).

## Policy

- **Only keep images the docs actually reference.** Unreferenced throwaway
  captures were removed from the working tree; new ones should not be committed.
  `.gitignore` excludes the common throwaway patterns (`*_final.png`,
  `*_frame*.png`, `*_clamped.png`).
- Prefer regenerating previews from code, or attaching large media to a GitHub
  **release** rather than committing it into the repo.

## Note on repository size

`git rm` (used here) removes files from the working tree and future checkouts but
does **not** shrink the repository history — the original blobs remain reachable
from older commits. Actually reclaiming that space requires rewriting history,
e.g. with [`git filter-repo`](https://github.com/newren/git-filter-repo):

```bash
# DESTRUCTIVE: rewrites every commit SHA; coordinate with all clones first, and
# note the default branch may be a protected ref that rejects force-pushes.
git filter-repo --path ai/previews/aicam.gif --invert-paths
```

Because that is destructive and force-pushes to a protected branch, it was left
as a deliberate, separately-approved operation rather than done automatically.
