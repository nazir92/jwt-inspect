package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// version can be set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `Usage: jwt-inspect [TOKEN | "Bearer TOKEN" | Bearer TOKEN]
       jwt-inspect --help | -help | -h
       jwt-inspect --version | -v

Without arguments, read a token (optionally prefixed by Bearer) from stdin.
The signature and claims are not verified.
`

func main() {
	interactive := false
	if len(os.Args) == 1 {
		info, err := os.Stdin.Stat()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error checking stdin")
			os.Exit(1)
		}
		interactive = info.Mode()&os.ModeCharDevice != 0
	}
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, interactive))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, interactive bool) int {
	fail := func(message string) int {
		fmt.Fprintln(stderr, message)
		return 1
	}

	var token string
	switch len(args) {
	case 0:
		if interactive {
			return fail("No token provided; pass a token or pipe one through stdin. See --help for usage.")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return fail("Error reading stdin")
		}
		token = strings.TrimSpace(string(data))
		token = strings.TrimPrefix(token, "Bearer ")
	case 1:
		switch args[0] {
		case "--help", "-help", "-h":
			fmt.Fprint(stdout, usage)
			return 0
		case "--version", "-v":
			fmt.Fprintln(stdout, version)
			return 0
		}
		if strings.HasPrefix(args[0], "-") {
			return fail("Invalid argument. See --help for usage.")
		}
		token = args[0]
		token = strings.TrimPrefix(token, "Bearer ")
	case 2:
		if args[0] != "Bearer" {
			return fail("Invalid arguments. See --help for usage.")
		}
		token = args[1]
	default:
		return fail("Invalid arguments. See --help for usage.")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fail("Incorrect JWT token: expected three sections")
	}

	formatted := make([]string, 2)
	for i, name := range []string{"header", "payload"} {
		decoded, err := decodeValue(parts[i])
		if err != nil {
			return fail("Invalid base64url in JWT " + name)
		}
		formatted[i], err = formatJSON(decoded)
		if err != nil {
			return fail("Invalid JSON in JWT " + name)
		}
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, formatted[0])
	fmt.Fprintln(stdout, formatted[1])
	return 0
}

func decodeValue(value string) ([]byte, error) {
	if strings.ContainsAny(value, "\r\n") {
		return nil, fmt.Errorf("base64url contains a newline")
	}
	return base64.RawURLEncoding.Strict().DecodeString(value)
}

func formatJSON(jsonData []byte) (string, error) {
	var buf bytes.Buffer
	err := json.Indent(&buf, jsonData, "", " ")
	return buf.String(), err
}
