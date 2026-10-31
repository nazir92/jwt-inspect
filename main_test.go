package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func encoded(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

func TestCLIHelper(t *testing.T) {
	if os.Getenv("JWT_INSPECT_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"jwt-inspect"}, os.Args[i+1:]...)
			main()
			return
		}
	}
	t.Fatal("missing helper argument separator")
}

func TestCLI(t *testing.T) {
	header := encoded(`{"alg":"none"}`)
	payload := encoded(`{"sub":"123"}`)
	token := header + "." + payload + ".signature"
	wantOutput := "\n{\n \"alg\": \"none\"\n}\n{\n \"sub\": \"123\"\n}\n"

	tests := []struct {
		name       string
		args       []string
		stdin      string
		wantStdout string
		wantError  string
		status     int
	}{
		{name: "token", args: []string{token}, wantStdout: wantOutput},
		{name: "quoted bearer", args: []string{"Bearer " + token}, wantStdout: wantOutput},
		{name: "separate bearer", args: []string{"Bearer", token}, wantStdout: wantOutput},
		{name: "stdin", stdin: token + "\n", wantStdout: wantOutput},
		{name: "stdin bearer", stdin: "  Bearer " + token + "\n", wantStdout: wantOutput},
		{name: "empty signature", args: []string{header + "." + payload + "."}, wantStdout: wantOutput},
		{name: "signature not checked", args: []string{header + "." + payload + ".not_base64!"}, wantStdout: wantOutput},
		{name: "help", args: []string{"--help"}, wantStdout: usage},
		{name: "single-dash help", args: []string{"-help"}, wantStdout: usage},
		{name: "short help", args: []string{"-h"}, wantStdout: usage},
		{name: "version", args: []string{"--version"}, wantStdout: version + "\n"},
		{name: "short version", args: []string{"-v"}, wantStdout: version + "\n"},
		{name: "empty stdin", wantError: "Incorrect JWT token", status: 1},
		{name: "empty token", args: []string{""}, wantError: "Incorrect JWT token", status: 1},
		{name: "missing bearer token", args: []string{"Bearer"}, wantError: "Incorrect JWT token", status: 1},
		{name: "unknown flag", args: []string{"--unknown"}, wantError: "Invalid argument", status: 1},
		{name: "flag with token", args: []string{"--help", token}, wantError: "Invalid arguments", status: 1},
		{name: "wrong prefix", args: []string{"Basic", token}, wantError: "Invalid arguments", status: 1},
		{name: "extra arguments", args: []string{"Bearer", token, "extra"}, wantError: "Invalid arguments", status: 1},
		{name: "too few sections", args: []string{header + "." + payload}, wantError: "Incorrect JWT token", status: 1},
		{name: "too many sections", args: []string{token + ".extra"}, wantError: "Incorrect JWT token", status: 1},
		{name: "invalid header base64", args: []string{"!secret." + payload + ".sig"}, wantError: "Invalid base64url in JWT header", status: 1},
		{name: "invalid payload base64", args: []string{header + ".!secret.sig"}, wantError: "Invalid base64url in JWT payload", status: 1},
		{name: "noncanonical base64", args: []string{"Zh." + payload + ".sig"}, wantError: "Invalid base64url in JWT header", status: 1},
		{name: "newline in base64", args: []string{header + "\n." + payload + ".sig"}, wantError: "Invalid base64url in JWT header", status: 1},
		{name: "invalid header json", args: []string{encoded("secret invalid") + "." + payload + ".sig"}, wantError: "Invalid JSON in JWT header", status: 1},
		{name: "invalid payload json", args: []string{header + "." + encoded("secret invalid") + ".sig"}, wantError: "Invalid JSON in JWT payload", status: 1},
		{name: "empty header", args: []string{"." + payload + ".sig"}, wantError: "Invalid JSON in JWT header", status: 1},
		{name: "empty payload", args: []string{header + "..sig"}, wantError: "Invalid JSON in JWT payload", status: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"-test.run=^TestCLIHelper$", "--"}, tt.args...)
			cmd := exec.Command(os.Args[0], args...)
			cmd.Env = append(os.Environ(), "JWT_INSPECT_TEST_HELPER=1")
			cmd.Stdin = strings.NewReader(tt.stdin)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			status := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("execute CLI: %v", err)
				}
				status = exitErr.ExitCode()
			}
			if status != tt.status || stdout.String() != tt.wantStdout {
				t.Errorf("status = %d, stdout = %q; want %d, %q", status, stdout.String(), tt.status, tt.wantStdout)
			}
			if tt.wantError == "" {
				if stderr.Len() != 0 {
					t.Errorf("unexpected stderr: %q", stderr.String())
				}
			} else if !strings.Contains(stderr.String(), tt.wantError) {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.wantError)
			}
			if strings.Contains(stderr.String(), "secret") || strings.Contains(stderr.String(), token) {
				t.Errorf("stderr leaks token data: %q", stderr.String())
			}
		})
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("secret read error") }

func TestStdinErrorsAndInteractive(t *testing.T) {
	for _, tt := range []struct {
		name        string
		interactive bool
		message     string
	}{
		{"read failure", false, "Error reading stdin"},
		{"interactive", true, "No token provided"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			status := run(nil, errorReader{}, &out, &stderr, tt.interactive)
			if status != 1 || out.Len() != 0 || !strings.Contains(stderr.String(), tt.message) || strings.Contains(stderr.String(), "secret") {
				t.Fatalf("status %d, stdout %q, stderr %q", status, out.String(), stderr.String())
			}
		})
	}
}

func TestCharacterDeviceStdin(t *testing.T) {
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()

	cmd := exec.Command(os.Args[0], "-test.run=^TestCLIHelper$", "--")
	cmd.Env = append(os.Environ(), "JWT_INSPECT_TEST_HELPER=1")
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || stdout.Len() != 0 ||
		!strings.Contains(stderr.String(), "No token provided") {
		t.Fatalf("error %v, stdout %q, stderr %q", err, stdout.String(), stderr.String())
	}
}

var _ io.Reader = errorReader{}
