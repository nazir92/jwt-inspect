# jwt-inspect

`jwt-inspect` is a small, dependency-free Go CLI that decodes and displays a
JSON Web Token's header and payload as formatted JSON.

> **Warning:** This tool does not verify signatures or validate claims such as
> `exp`, `iss`, or `aud`. Decoded output does not prove a token is authentic or
> valid. JWT payloads are encoded, not encrypted.

## Installation

**Ubuntu:** Download the `.deb` for your architecture from
[GitHub Releases](https://github.com/nazir92/jwt-inspect/releases), verify its
checksum as described below, then replace `PACKAGE.deb` with the downloaded
filename:

```bash
sudo apt install ./PACKAGE.deb
```

The Ubuntu package installs the command in `/usr/bin`, which is normally on
your `PATH`.

### macOS, Linux, and Windows archives

Download the archive for your operating system and CPU architecture from
[GitHub Releases](https://github.com/nazir92/jwt-inspect/releases). Verify its
checksum before extracting it or installing the binary.

On Linux or macOS, calculate the archive's SHA-256 checksum with
`sha256sum ARCHIVE` or `shasum -a 256 ARCHIVE` and compare it with the matching
entry in the release's `checksums.txt`. In PowerShell, use
`Get-FileHash .\ARCHIVE -Algorithm SHA256` and compare its hash. A checksum
from the same release detects corruption but does not independently verify the
publisher.

After extracting the archive on macOS or Linux, run:

```bash
sh ./install.sh
```

On Windows, run the installer in PowerShell from the extracted directory:

```powershell
.\install.ps1
```

The installers copy the binary to your user directory and configure your user
`PATH`: on Unix they update shell startup files; on Windows they update the user
environment. Open a new terminal and run `jwt-inspect --help`. The Unix
installer supports zsh, bash, sh, and dash. On Windows, the PowerShell execution
policy must permit running the local script.

To upgrade an existing installation, run `sh ./install.sh --force` on macOS or
Linux, or `.\install.ps1 -Force` on Windows. These options replace the existing
binary in the installer-managed location.

Alternatively, build from source with Go 1.26.2 or newer:

```bash
git clone https://github.com/nazir92/jwt-inspect.git
cd jwt-inspect
go build .
```

## Usage

Replace `TOKEN` below with a three-part JWT:

```bash
jwt-inspect TOKEN
jwt-inspect Bearer TOKEN
```

The CLI also accepts the `Bearer` prefix and token together as one argument.
Use `jwt-inspect -help` (also `--help` or `-h`) for the supported forms and `jwt-inspect --version`
to display the version (`dev` for local builds).

To read from standard input instead of putting the token directly in a command:

```bash
printf '%s\n' "$JWT_TOKEN" | jwt-inspect
```

Standard input can contain the token alone or include the `Bearer` prefix.
Keep tokens out of shell history, shared terminals, logs, and online tools.
Environment variables and command arguments may also be visible to other
processes, depending on your system.

The tool expects three dot-separated sections, Base64URL-decodes the first two,
and prints each as indented JSON. It does not decode or check the signature.
Invalid input is reported on standard error with a non-zero exit status.

## Contributing

Contributions are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md) for the
development and pull request workflow.

## Releases

After changes are merged to `main` and checks pass, release automation uses the
squash commit's Conventional Commit type to determine whether to publish a new
version. Features, fixes, performance improvements, and breaking changes
trigger releases; documentation or CI changes alone do not. The first feature
release is `v0.1.0`.

The workflow publishes release assets to this repository using its normal
`GITHUB_TOKEN`. No additional package repositories or tokens are required.

If publication fails after a version has been tagged, later releases stop until
the incomplete release is retried. A maintainer can run the Release workflow
from the Actions tab on `main` and supply the failed tag as `retry_tag`. The
workflow reruns checks on the tagged commit before trying again.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
