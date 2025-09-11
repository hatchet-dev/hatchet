#!/bin/bash
set -e

# Required parameters (no defaults)
SOURCE_DIR=""
TARGET_REPO=""
GIT_USERNAME="Hatchet Actions Bot"
GIT_EMAIL="dev@hatchet.run"
TEMP_DIR=$(mktemp -d)
GITHUB_SHA=$(git rev-parse HEAD)

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --source-dir)
      SOURCE_DIR="$2"
      shift
      shift
      ;;
    --target-repo)
      TARGET_REPO="$2"
      shift
      shift
      ;;
    --token)
      GITHUB_TOKEN="$2"
      shift
      shift
      ;;
    --git-username)
      GIT_USERNAME="$2"
      shift
      shift
      ;;
    --git-email)
      GIT_EMAIL="$2"
      shift
      shift
      ;;
    *)
      shift
      ;;
  esac
done

# Validate required input
if [ -z "$SOURCE_DIR" ]; then
  echo "Error: Source directory (--source-dir) is required"
  exit 1
fi

if [ -z "$TARGET_REPO" ]; then
  echo "Error: Target repository (--target-repo) is required"
  exit 1
fi

if [ ! -d "$SOURCE_DIR" ]; then
  echo "Error: Source directory '$SOURCE_DIR' does not exist"
  exit 1
fi

# Check if running in GitHub Actions
if [ -z "$GITHUB_TOKEN" ] && [ -n "$GITHUB_ACTIONS" ]; then
  if [ -n "$SYNC_TOKEN" ]; then
    GITHUB_TOKEN="$SYNC_TOKEN"
  else
    echo "Error: No GitHub token provided. Please set GITHUB_TOKEN or SYNC_TOKEN environment variable."
    exit 1
  fi
fi

# Set the repository URL with token if available
if [ -n "$GITHUB_TOKEN" ]; then
  REPO_URL="https://${GITHUB_TOKEN}@github.com/${TARGET_REPO}.git"
else
  REPO_URL="https://github.com/${TARGET_REPO}.git"
  echo "Warning: No GitHub token provided. Using public URL. Push may fail if repository is private."
fi

echo "Syncing files from $SOURCE_DIR to $TARGET_REPO..."

# Clone the target repository
echo "Cloning target repository..."
git clone "$REPO_URL" "$TEMP_DIR/target"

# Remove all files from target except .git and .github
echo "Removing existing files from target..."
find "$TEMP_DIR/target" -mindepth 1 -maxdepth 1 ! -name '.git' ! -name '.github' -exec rm -rf {} +

# Copy all files from source directory to target
echo "Copying files from source to target..."
cp -r "$SOURCE_DIR"/* "$TEMP_DIR/target/"

# Set git config
cd "$TEMP_DIR/target"
git config user.name "$GIT_USERNAME"
git config user.email "$GIT_EMAIL"

# Add all changes (including removed files)
git add -A

# Check if there are changes to commit
if [ -z "$(git status --porcelain)" ]; then
  echo "No changes to commit"
  exit 0
fi

# Commit and push changes
echo "Committing and pushing changes..."
git commit -m "Sync with main repo: ${GITHUB_SHA}"
git push

echo "Sync completed successfully!"

# Clean up
cd - > /dev/null
rm -rf "$TEMP_DIR"
