# jwt-inspect

`jwt-inspect` is a small, dependency-free Go CLI that decodes and displays a
JSON Web Token's header and payload as formatted JSON.

> **Warning:** This tool does not verify signatures or validate claims such as
> `exp`, `iss`, or `aud`. Decoded output does not prove a token is authentic or
> valid. JWT payloads are encoded, not encrypted.

## Installation

Download the archive for your operating system and CPU architecture from
[GitHub Releases](https://github.com/nazir92/jwt-inspect/releases), extract it,
and place `jwt-inspect` (or `jwt-inspect.exe` on Windows) somewhere on your
`PATH`. Releases also include a checksum file for verifying downloads.

For example, after extracting the archive on macOS or Linux, install the binary
in your user directory:

```bash
mkdir -p "$HOME/.local/bin"
install -m 755 jwt-inspect "$HOME/.local/bin/jwt-inspect"
export PATH="$HOME/.local/bin:$PATH"
jwt-inspect -help
```

The `export` applies to the current shell. To make it permanent, add that line
to your shell's startup file, such as `~/.zshrc` or `~/.bashrc`. On Windows,
place `jwt-inspect.exe` in a directory on your user `Path`, or add its directory
to `Path` in Windows environment variable settings, then open a new terminal.

For example, on Linux or macOS, calculate the archive's SHA-256 checksum with
`sha256sum ARCHIVE` or `shasum -a 256 ARCHIVE` and compare it with the matching
entry in the checksum file. In PowerShell, use
`Get-FileHash .\ARCHIVE -Algorithm SHA256` and compare its hash.

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

If publication fails after a version has been tagged, later releases stop until
the incomplete release is retried. A maintainer can run the Release workflow
from the Actions tab on `main` and supply the failed tag as `retry_tag`. The
workflow reruns checks on the tagged commit before trying again.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
