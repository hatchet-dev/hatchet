# Release Instructions — Hatchet VS Code Extension

## Before your first release

### 1. Add the icon

`assets/icon.png` must be a real **128×128 PNG** before `vsce package` will succeed.
The file is currently a placeholder (`.gitkeep` only). Add the icon and commit it:

```bash
# Drop your 128x128 PNG in place, then:
git add frontend/vscode-extension/assets/icon.png
```

`vsce package` will error with "The icon must be a PNG file" / "file not found" if the icon is missing or wrong size.

### 2. Obtain a VS Code Marketplace Personal Access Token (PAT)

1. Go to https://marketplace.visualstudio.com/manage
2. Sign in with the `hatchet-dev` publisher account
3. In Azure DevOps, create a PAT with **Marketplace → Manage** scope
4. Add it as a repository secret named `VSCE_PAT`

## Release process

### Option A — Automated via git tag (recommended)

Pushing a tag of the form `vscode/vX.Y.Z` triggers `.github/workflows/vscode-release.yml`, which:
- installs deps, runs `pnpm build:prod`, packages the `.vsix`, publishes to the Marketplace, and uploads the `.vsix` as a GitHub Release artifact.

```bash
# 1. Bump the version in package.json
#    (update "version": "X.Y.Z")
vim frontend/vscode-extension/package.json

# 2. Commit the version bump
git add frontend/vscode-extension/package.json
git commit -m "chore(vscode): bump extension version to X.Y.Z"

# 3. Push the tag
git tag vscode/vX.Y.Z
git push origin vscode/vX.Y.Z
```

The workflow runs automatically. Monitor it at:
https://github.com/hatchet-dev/hatchet/actions/workflows/vscode-release.yml

### Option B — Manual release

```bash
cd frontend/vscode-extension

# 1. Install deps
pnpm install

# 2. Production build
pnpm build:prod

# 3. Package into a .vsix
pnpm package
# Produces: hatchet-X.Y.Z.vsix

# 4. Smoke-test the packaged extension locally
code --install-extension hatchet-X.Y.Z.vsix
# Open a workflow file and verify the CodeLens and DAG panel work.

# 5. Publish to the Marketplace
VSCE_PAT=<your-token> pnpm publish
```

## Versioning convention

Follow semver. The extension version lives only in `frontend/vscode-extension/package.json` and is independent of the main Hatchet server version.

| Change type | Example |
|---|---|
| Bug fix, minor improvement | patch: `0.1.0` → `0.1.1` |
| New language support, new feature | minor: `0.1.0` → `0.2.0` |
| Breaking change to config/API | major: `0.1.0` → `1.0.0` |
