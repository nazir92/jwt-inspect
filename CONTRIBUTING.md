# Contributing

Contributions are welcome! You can report bugs, suggest improvements, or open
a pull request. By participating, you agree that your contributions are subject
to the repository's [Apache 2.0 license](LICENSE).

## Submitting a change

1. Fork the repository and create a branch from `main`.
2. Make your changes, adding tests where appropriate.
3. Format and check your code:

   ```bash
   gofmt -w .
   go test ./...
   go vet ./...
   ```

4. Push your branch to your fork and open a pull request against `main`.
5. Address review feedback and wait for the required checks and a maintainer
   review before the change is merged.

## Pull request titles

Pull requests are squash-merged. Use a
[Conventional Commit](https://www.conventionalcommits.org/) title because it
becomes the commit message used to determine whether to create a release:

```text
fix: report invalid tokens
feat: read tokens from stdin
docs: improve installation instructions
ci: run tests on Windows
feat!: change the output format
```

Scopes are welcome, for example `fix(cli): handle invalid input`. Use `!` after
the type or scope for a breaking change. The squash commit message may also use
a `BREAKING CHANGE:` footer to describe a breaking change.

The maintainer reviews and merges pull requests. A valid title does not
guarantee that a pull request will be accepted or immediately released.
