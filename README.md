# assets branch

This is an **orphan branch** (no shared history with `main`) that holds binary
media — the AI preview images — *out of* the main code history. It exists so
`git clone` of the code stays light while the docs can still embed pictures.

## Why a separate branch

Binary files (`.gif`, `.png`) cannot be diffed or merged, and git keeps a full
copy of every version forever. Committing them on `main` bloats its history
permanently. Keeping them on an orphan branch means:

- the main code history carries no image blobs;
- docs reference them by absolute `raw.githubusercontent.com` URL, so they still
  render inline on GitHub;
- no Git LFS, no extra tooling, no storage/bandwidth limits.

## How docs reference these

```markdown
![sunset](https://raw.githubusercontent.com/svend4/infon/assets/previews/sunset_real.png)
```

## Updating an image

```bash
git switch assets           # or: git worktree add ../infon-assets assets
cp new_capture.png previews/whatever.png
git add previews && git commit -m "assets: update whatever.png" && git push
```

Do **not** merge this branch into `main`.
