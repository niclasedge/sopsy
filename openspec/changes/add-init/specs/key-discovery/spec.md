# Spec Delta

## Purpose

Defines where sopsy finds the age private key and the encrypted secrets file on macOS, Linux and Windows, so every command resolves them identically.

## ADDED Requirements

### Requirement: Age key lookup order
sopsy SHALL look for the age private key in this order and use the first file that exists: `$SOPS_AGE_KEY_FILE`, then `~/.config/sops/age/keys.txt`, then `<user config dir>/sops/age/keys.txt` where the user config dir is the operating system default (`~/Library/Application Support` on macOS, `$XDG_CONFIG_HOME` or `~/.config` on Linux, `%AppData%` on Windows).

#### Scenario: Explicit key file wins
- **WHEN** `SOPS_AGE_KEY_FILE` points to an existing file and a key also exists in `~/.config/sops/age/keys.txt`
- **THEN** sopsy uses the file named by `SOPS_AGE_KEY_FILE`

#### Scenario: macOS key in ~/.config is found
- **WHEN** on macOS the only key is at `~/.config/sops/age/keys.txt` and `SOPS_AGE_KEY_FILE` is unset
- **THEN** sopsy finds and uses that key

#### Scenario: Windows default location
- **WHEN** on Windows the only key is at `%AppData%\sops\age\keys.txt`
- **THEN** sopsy finds and uses that key

#### Scenario: Explicit key file missing
- **WHEN** `SOPS_AGE_KEY_FILE` is set but the file does not exist
- **THEN** sopsy fails with exit code 125 and names that path, without falling back to other locations

### Requirement: Secrets file resolution
sopsy SHALL resolve the secrets file from `--file`, else `$SOPSY_FILE`, else `secrets.env` in the working directory, and SHALL resolve `.sops.yaml` by searching from the working directory upwards.

#### Scenario: Flag overrides environment
- **WHEN** both `--file a.env` and `SOPSY_FILE=b.env` are given
- **THEN** sopsy operates on `a.env`

#### Scenario: Config found in a parent directory
- **WHEN** sopsy runs in a subdirectory and `.sops.yaml` exists only in a parent directory
- **THEN** sopsy uses that parent `.sops.yaml`
