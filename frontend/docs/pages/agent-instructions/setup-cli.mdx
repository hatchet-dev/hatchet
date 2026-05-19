# Install and Set Up the Hatchet CLI

These are instructions for an AI agent to install the Hatchet CLI and configure a profile. Follow each step in order.

## Step 1: Check if Already Installed

```bash
hatchet --version
```

If this prints a version number, the CLI is already installed. Skip to Step 3 (profile setup).

If the command is not found, proceed to Step 2.

## Step 2: Install the CLI

On macOS, Linux, or WSL:

```bash
curl -fsSL https://install.hatchet.run/install.sh | bash
```

Alternatively, on macOS via Homebrew:

```bash
brew install hatchet-dev/hatchet/hatchet --cask
```

After installation, verify it worked:

```bash
hatchet --version
```

## Step 3: Check for Existing Profiles

```bash
hatchet profile list
```

If a profile already exists that connects to the correct Hatchet instance, note its name and use it as the `-p` flag in all subsequent commands. You are done.

If no profiles exist or the correct one is missing, proceed to Step 4.

## Step 4: Create a Profile

You need a Hatchet API token. Ask the user for one if you do not have it. Then create a profile:

```bash
hatchet profile add --name HATCHET_PROFILE --token <API_TOKEN>
```

Replace `HATCHET_PROFILE` with a descriptive name (e.g. `local`, `staging`, `production`) and `<API_TOKEN>` with the actual token.

To set it as the default profile (so `-p` is optional in future commands):

```bash
hatchet profile set-default --name HATCHET_PROFILE
```

## Step 5: Verify Connectivity

Test that the profile works by listing workflows:

```bash
hatchet runs list -o json -p HATCHET_PROFILE --since 1h --limit 1
```

If this returns a JSON response (even with an empty rows list), the profile is correctly configured and connected.

## Troubleshooting

- **"command not found"** after install: The CLI binary may not be on your PATH. Check `~/.local/bin/hatchet` or re-run the install script.
- **Authentication error**: The API token may be invalid or expired. Ask the user for a new token and run `hatchet profile update`.
- **Connection refused**: The Hatchet server may not be running. For local development, start it with `hatchet server start`.
