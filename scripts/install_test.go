//go:build unix

package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct {
	root, home, archive, installer string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	root := t.TempDir()
	f := fixture{
		root:    root,
		home:    filepath.Join(root, "home's files"),
		archive: filepath.Join(root, "archive's files"),
	}
	f.installer = filepath.Join(f.archive, "scripts", "install.sh")
	mkdir(t, f.home)
	mkdir(t, filepath.Dir(f.installer))
	script, err := os.ReadFile("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	write(t, f.installer, string(script), 0700)
	f.binary(t, "version one")
	return f
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) binary(t *testing.T, version string) {
	t.Helper()
	write(t, filepath.Join(f.archive, "jwt-inspect"), "#!/bin/sh\nprintf '"+version+"\\n'\n", 0700)
}

func (f fixture) run(t *testing.T, shell string, extraEnv []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("/bin/sh", append([]string{f.installer}, args...)...)
	cmd.Dir = f.root
	cmd.Env = append([]string{"HOME=" + f.home, "SHELL=" + shell, "PATH=/usr/bin:/bin"}, extraEnv...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func mustRun(t *testing.T, f fixture, shell string, env []string, args ...string) string {
	t.Helper()
	out, err := f.run(t, shell, env, args...)
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, out)
	}
	return out
}

func mustFail(t *testing.T, f fixture, shell, want string, args ...string) {
	t.Helper()
	out, err := f.run(t, shell, nil, args...)
	if err == nil || !strings.Contains(out, want) {
		t.Fatalf("wanted failure containing %q, got %v:\n%s", want, err, out)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInstallAndUpgrade(t *testing.T) {
	f := newFixture(t)
	out := mustRun(t, f, "/bin/bash", nil)
	if !strings.Contains(out, "Open a new terminal") {
		t.Fatal(out)
	}
	dest := filepath.Join(f.home, ".local", "bin", "jwt-inspect")
	rc := filepath.Join(f.home, ".bashrc")
	before := read(t, rc)
	mustRun(t, f, "/bin/bash", nil)
	if read(t, rc) != before || strings.Count(before, `export PATH="$HOME/.local/bin:$PATH"`) != 1 {
		t.Fatal("startup file is not idempotent")
	}
	f.binary(t, "version two")
	mustFail(t, f, "/bin/bash", "use --force")
	if !strings.Contains(read(t, dest), "version one") {
		t.Fatal("existing binary was modified without --force")
	}
	mustRun(t, f, "/bin/bash", nil, "--force")
	if !strings.Contains(read(t, dest), "version two") {
		t.Fatal("upgrade did not install the new binary")
	}
	info, err := os.Stat(dest)
	if err != nil || info.Mode()&0111 == 0 {
		t.Fatalf("upgraded binary is not executable: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(dest))
	if err != nil || len(entries) != 1 {
		t.Fatalf("upgrade left unexpected staging files: %v, %v", entries, err)
	}
	mustRun(t, f, "/bin/bash", nil, "--force")
	if read(t, rc) != before {
		t.Fatal("upgrade changed startup file again")
	}
	assertResolved(t, f, "/bin/bash", []string{"--noprofile", "--rcfile", rc, "-ic"}, nil)
}

func assertResolved(t *testing.T, f fixture, shell string, args, extraEnv []string) {
	t.Helper()
	args = append(args, "command -v jwt-inspect")
	cmd := exec.Command(shell, args...)
	cmd.Dir = f.root
	cmd.Env = append([]string{"HOME=" + f.home, "PATH=/usr/bin:/bin"}, extraEnv...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(f.home, ".local", "bin", "jwt-inspect")
	if strings.TrimSpace(string(out)) != want {
		t.Fatalf("resolved %q, want %q", out, want)
	}
}

func TestShellStartup(t *testing.T) {
	t.Run("archive-root", func(t *testing.T) {
		f := newFixture(t)
		script := read(t, f.installer)
		f.installer = filepath.Join(f.archive, "install.sh")
		write(t, f.installer, script, 0700)
		mustRun(t, f, "/bin/sh", nil)
		if _, err := os.Stat(filepath.Join(f.home, ".local", "bin", "jwt-inspect")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("sh", func(t *testing.T) {
		f := newFixture(t)
		mustRun(t, f, "/bin/sh", nil)
		assertResolved(t, f, "/bin/sh", []string{"-c", `. "$HOME/.profile"; command -v jwt-inspect`}, nil)
	})
	t.Run("zsh-custom-directory", func(t *testing.T) {
		f := newFixture(t)
		dir := filepath.Join(f.home, "zsh config")
		mkdir(t, dir)
		env := []string{"ZDOTDIR=" + dir}
		mustRun(t, f, "/bin/zsh", env)
		rc := filepath.Join(dir, ".zshrc")
		before := read(t, rc)
		mustRun(t, f, "/bin/zsh", env)
		if read(t, rc) != before {
			t.Fatal("zsh startup file is not idempotent")
		}
		if _, err := os.Stat("/bin/zsh"); err == nil {
			assertResolved(t, f, "/bin/zsh", []string{"-ic"}, env)
		}
	})
	t.Run("bash-existing-profile", func(t *testing.T) {
		f := newFixture(t)
		profile := filepath.Join(f.home, ".profile")
		bashProfile := filepath.Join(f.home, ".bash_profile")
		write(t, profile, "original profile\n", 0600)
		write(t, bashProfile, "existing bash profile\n", 0600)
		mustRun(t, f, "/bin/bash", nil)
		if read(t, profile) != "original profile\n" || !strings.Contains(read(t, bashProfile), `export PATH="$HOME/.local/bin:$PATH"`) {
			t.Fatal("incorrect bash login startup file")
		}
	})
}

func TestRefusesUnsafeFiles(t *testing.T) {
	t.Run("destination-symlink", func(t *testing.T) {
		f := newFixture(t)
		dest := filepath.Join(f.home, ".local", "bin", "jwt-inspect")
		mkdir(t, filepath.Dir(dest))
		target := filepath.Join(f.root, "target")
		write(t, target, "untouched", 0600)
		if err := os.Symlink(target, dest); err != nil {
			t.Fatal(err)
		}
		mustFail(t, f, "/bin/bash", "symbolic link", "--force")
		if read(t, target) != "untouched" {
			t.Fatal("symlink target was modified")
		}
	})
	t.Run("startup-symlink", func(t *testing.T) {
		f := newFixture(t)
		target := filepath.Join(f.root, "target")
		write(t, target, "untouched", 0600)
		if err := os.Symlink(target, filepath.Join(f.home, ".zshrc")); err != nil {
			t.Fatal(err)
		}
		mustFail(t, f, "/bin/zsh", "symbolic link")
		if read(t, target) != "untouched" {
			t.Fatal("symlink target was modified")
		}
	})
	t.Run("destination-directory-symlink", func(t *testing.T) {
		f := newFixture(t)
		dest := filepath.Join(f.home, ".local", "bin", "jwt-inspect")
		mkdir(t, filepath.Dir(dest))
		target := filepath.Join(f.root, "target")
		mkdir(t, target)
		if err := os.Symlink(target, dest); err != nil {
			t.Fatal(err)
		}
		mustFail(t, f, "/bin/bash", "not a regular file", "--force")
		entries, err := os.ReadDir(target)
		if err != nil || len(entries) != 0 {
			t.Fatalf("symlink target directory was modified: %v, %v", entries, err)
		}
	})
	t.Run("bin-directory-symlink", func(t *testing.T) {
		f := newFixture(t)
		local := filepath.Join(f.home, ".local")
		mkdir(t, local)
		if err := os.Symlink(f.root, filepath.Join(local, "bin")); err != nil {
			t.Fatal(err)
		}
		mustFail(t, f, "/bin/bash", "symbolic link", "--force")
	})
	t.Run("writable-startup", func(t *testing.T) {
		f := newFixture(t)
		path := filepath.Join(f.home, ".zshrc")
		write(t, path, "", 0600)
		if err := os.Chmod(path, 0666); err != nil {
			t.Fatal(err)
		}
		mustFail(t, f, "/bin/zsh", "not group- or world-writable")
	})
	t.Run("unsupported-shell", func(t *testing.T) {
		f := newFixture(t)
		mustFail(t, f, "/bin/fish", "unsupported SHELL")
	})
	t.Run("missing-binary", func(t *testing.T) {
		f := newFixture(t)
		if err := os.Remove(filepath.Join(f.archive, "jwt-inspect")); err != nil {
			t.Fatal(err)
		}
		mustFail(t, f, "/bin/sh", "executable jwt-inspect not found")
	})
	t.Run("source-symlink", func(t *testing.T) {
		f := newFixture(t)
		source := filepath.Join(f.archive, "jwt-inspect")
		other := filepath.Join(f.root, "other")
		if err := os.Rename(source, other); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(other, source); err != nil {
			t.Fatal(err)
		}
		mustFail(t, f, "/bin/sh", "symbolic link", "--force")
	})
	t.Run("help", func(t *testing.T) {
		f := newFixture(t)
		out := mustRun(t, f, "/bin/sh", nil, "--help")
		if !strings.Contains(out, "--force") {
			t.Fatal(out)
		}
	})
}
