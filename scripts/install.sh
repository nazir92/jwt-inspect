#!/bin/sh
set -eu
umask 077

fail() { printf 'install: %s\n' "$*" >&2; exit 1; }

force=false
case ${1:-} in
    '') ;;
    --force) force=true; shift ;;
    --help)
        printf 'Usage: sh install.sh [--force]\n'
        printf 'Use --force to replace an existing, different user installation.\n'
        exit 0
        ;;
    *) fail 'usage: sh install.sh [--force]' ;;
esac
[ "$#" -eq 0 ] || fail 'usage: sh install.sh [--force]'

[ -n "${HOME:-}" ] || fail 'HOME is not set'
case $HOME in
    /*) ;;
    *) fail 'HOME must be an absolute path' ;;
esac
case $HOME in
    *:*) fail 'HOME must not contain a colon' ;;
    *'
'*) fail 'HOME must not contain a newline' ;;
esac

script_dir=$(CDPATH= cd -P "$(dirname "$0")" && pwd) || fail 'cannot locate installer'
source_bin=$script_dir/jwt-inspect
if [ ! -f "$source_bin" ] && [ "${script_dir##*/}" = scripts ]; then
    source_bin=$script_dir/../jwt-inspect
fi
[ -f "$source_bin" ] && [ -x "$source_bin" ] || fail "executable jwt-inspect not found next to the installer or in its archive root"
[ ! -L "$source_bin" ] || fail "refusing symbolic link: $source_bin"

# Only modify files in directories owned by this user and not writable by others.
safe_existing() {
    [ ! -L "$1" ] || fail "refusing symbolic link: $1"
    [ -n "$(find "$1" -prune -user "$(id -u)" ! -perm -020 ! -perm -002 -print)" ] ||
        fail "must be owned by you and not group- or world-writable: $1"
}
safe_dir() {
    [ -d "$1" ] || fail "not a directory: $1"
    safe_existing "$1"
}
safe_dir "$HOME"

case ${SHELL:-} in
    */zsh)
        zsh_dir=${ZDOTDIR:-$HOME}
        case $zsh_dir in
            /*) ;;
            *) fail 'ZDOTDIR must be an absolute path' ;;
        esac
        safe_dir "$zsh_dir"
        startup1=$zsh_dir/.zshrc; startup2=
        ;;
    */bash)
        startup1=$HOME/.bashrc
        if [ -e "$HOME/.bash_profile" ] || [ -L "$HOME/.bash_profile" ]; then
            startup2=$HOME/.bash_profile
        elif [ -e "$HOME/.bash_login" ] || [ -L "$HOME/.bash_login" ]; then
            startup2=$HOME/.bash_login
        else
            startup2=$HOME/.profile
        fi
        ;;
    */sh|*/dash) startup1=$HOME/.profile; startup2= ;;
    *) fail 'unsupported SHELL; use zsh, bash, or sh and rerun' ;;
esac

for startup in "$startup1" "$startup2"; do
    [ -n "$startup" ] || continue
    if [ -e "$startup" ] || [ -L "$startup" ]; then
        [ -f "$startup" ] || fail "not a regular startup file: $startup"
        safe_existing "$startup"
    fi
done

for dir in "$HOME/.local" "$HOME/.local/bin"; do
    if [ ! -e "$dir" ] && [ ! -L "$dir" ]; then
        mkdir "$dir"
    fi
    safe_dir "$dir"
done

dest=$HOME/.local/bin/jwt-inspect
if [ -e "$dest" ] || [ -L "$dest" ]; then
    [ -f "$dest" ] || fail "destination is not a regular file: $dest"
    safe_existing "$dest"
    if ! cmp -s "$source_bin" "$dest" || [ ! -x "$dest" ]; then
        "$force" || fail "destination differs or is not executable: $dest (use --force to replace it)"
        case $(uname -s) in
            Darwin) move_flag=-h ;;
            Linux) move_flag=-T ;;
            *) fail 'safe replacement is supported only on macOS and Linux' ;;
        esac
        stage_dir=$HOME/.local/bin/.jwt-inspect-install-$$
        mkdir "$stage_dir" || fail "could not create staging directory: $stage_dir"
        stage=$stage_dir/jwt-inspect
        trap 'rm -f "$stage"; rmdir "$stage_dir"' 0
        trap 'exit 1' HUP INT TERM
        (set -C; cat "$source_bin" > "$stage") || fail "could not create staging file: $stage"
        chmod 700 "$stage"
        mv -f "$move_flag" "$stage" "$dest"
    fi
else
    # noclobber ensures a file that appears after the check is not overwritten.
    (set -C; cat "$source_bin" > "$dest") || fail "could not create $dest"
    chmod 700 "$dest"
fi

path_line='export PATH="$HOME/.local/bin:$PATH"'
for startup in "$startup1" "$startup2"; do
    [ -n "$startup" ] || continue
    if [ ! -e "$startup" ] && [ ! -L "$startup" ]; then
        (set -C; : > "$startup") || fail "could not create $startup"
    fi
    safe_existing "$startup"
    if ! grep -F -x -q "$path_line" "$startup"; then
        printf '\n# jwt-inspect user binaries\n%s\n' "$path_line" >> "$startup"
        printf 'Updated %s\n' "$startup"
    fi
done

printf 'Installed %s\n' "$dest"
printf 'Open a new terminal (for sh, a login shell) for the PATH change to take effect.\n'
printf 'If you customize shell startup, ensure it reads the updated startup file(s).\n'
