# sopsy

Set and use secrets from a [SOPS](https://github.com/getsops/sops) + [age](https://age-encryption.org/)
encrypted dotenv file — one binary for macOS, Linux and Windows, with a guided local web UI.

> **Status: pre-release.** The CLI is implemented; the web UI (`sopsy ui`) is in progress.
> See [`docs/design.md`](docs/design.md) and the [issues](https://github.com/niclasedge/sopsy/issues).

## Usage

```bash
sopsy init                         # create age key, .sops.yaml and secrets.env (only what is missing)
sopsy set GITHUB_TOKEN             # hidden prompt, asked twice
printf '%s' "$TOKEN" | sopsy set GITHUB_TOKEN   # or from stdin; never as an argument
sopsy keys                         # names only, never values
sopsy run -- ./deploy.sh           # run a command with the secrets in its environment
sopsy doctor                       # check key, config, recipients and decryption
```

PowerShell:

```powershell
sopsy run -- python app.py
```

Sharing with another machine or a teammate:

```bash
sopsy pubkey                       # on the new machine: print its age public key
sopsy recipients add age1...       # on a machine that can decrypt: re-encrypt for it
sopsy recipients remove age1...    # revoke; then rotate every value
```

- No `sops` or `age` install needed — both are compiled in.
- Files stay fully SOPS-compatible; the regular `sops` CLI keeps working.
- sopsy never prints a secret value.
- `sopsy run` exits with the command's exit code; 125 means sopsy itself failed.

## Install

### Release download

Archives for macOS, Linux and Windows (amd64 and arm64) are on the
[releases page](https://github.com/niclasedge/sopsy/releases), with a
`checksums.txt` (SHA-256).

macOS and Linux:

```bash
v=0.1.0 os=darwin arch=arm64       # os: darwin|linux, arch: amd64|arm64
base=https://github.com/niclasedge/sopsy/releases/download/v$v
curl -fsSLO "$base/sopsy_${v}_${os}_${arch}.tar.gz"
curl -fsSLO "$base/checksums.txt"
shasum -a 256 --ignore-missing -c checksums.txt
tar xzf "sopsy_${v}_${os}_${arch}.tar.gz" sopsy
mkdir -p ~/.local/bin && mv sopsy ~/.local/bin/   # must be on your PATH
sopsy version
```

Windows (PowerShell):

```powershell
$v = '0.1.0'; $arch = 'amd64'      # or arm64
$base = "https://github.com/niclasedge/sopsy/releases/download/v$v"
Invoke-WebRequest "$base/sopsy_${v}_windows_$arch.zip" -OutFile sopsy.zip
Invoke-WebRequest "$base/checksums.txt" -OutFile checksums.txt
(Get-FileHash sopsy.zip -Algorithm SHA256).Hash.ToLower()   # must match the line in checksums.txt
$dir = "$env:LOCALAPPDATA\Programs\sopsy"
Expand-Archive sopsy.zip -DestinationPath $dir -Force
[Environment]::SetEnvironmentVariable('Path', "$([Environment]::GetEnvironmentVariable('Path', 'User'));$dir", 'User')
# open a new terminal, then:
sopsy version
```

### Unsigned binaries

The release binaries are not signed or notarized.

- **macOS Gatekeeper:** a file downloaded with a browser is quarantined and
  macOS refuses to open it ("Apple could not verify…"). Remove the flag with
  `xattr -d com.apple.quarantine sopsy`. Downloads with `curl` as above are not
  quarantined.
- **Windows SmartScreen:** a file downloaded with a browser may show "Windows
  protected your PC". Choose *More info* → *Run anyway*, or run
  `Unblock-File sopsy.exe` once. Downloads with `Invoke-WebRequest` as above are
  not marked.

### With Go

```bash
go install github.com/niclasedge/sopsy@latest
```

Requires the Go version in [`go.mod`](go.mod) or newer.

## License

[Apache-2.0](LICENSE). Release binaries include SOPS (MPL-2.0), age (BSD-3-Clause)
and further Go modules; see `THIRD_PARTY_LICENSES` in each release archive for
every license text and the source location of the MPL-2.0 code.
