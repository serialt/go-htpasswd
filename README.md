# htpasswd for Go

![](https://github.com/krabt/go-htpasswd/workflows/Go/badge.svg)
[![Go Doc](https://godoc.org/github.com/krabt/go-htpasswd?status.svg)](https://godoc.org/github.com/krabt/go-htpasswd)
[![Go Report Card](https://goreportcard.com/badge/github.com/krabt/go-htpasswd)](https://goreportcard.com/report/github.com/krabt/go-htpasswd)


This is a libary to validate user credentials against an HTTPasswd file.

This was forked from <https://github.com/jimstudt/http-authentication/tree/master/basic>
with modifications by @brian-avery to support SSHA, Md5Crypt, and Bcrypt and @jespersoderlund to support Crypt with SHA-256 and SHA-512 support.

## Currently, this supports:
* SSHA
* MD5Crypt
* APR1Crypt
* SHA
* Bcrypt
* Plain text
* Crypt with SHA-256 and SHA-512

## Command-line tool

The repository also includes an Apache-compatible `htpasswd` command for
creating, updating, verifying, and deleting users.

### Install

```sh
# Install the latest release into GOBIN.
go install github.com/krabt/go-htpasswd/cmd/htpasswd@latest

# Or build the command from a source checkout.
go build -o ./bin/htpasswd ./cmd/htpasswd

# The equivalent Makefile targets.
make build
make install

# Display the command reference.
htpasswd -h
```

Run `make help` to list development targets. Common checks can be run together
with `make check`. A release archive can be built locally with, for example:

```sh
make package VERSION=v1.3.0 GOOS=linux GOARCH=amd64
make package VERSION=v1.3.0 GOOS=windows GOARCH=amd64

# Build every Linux, macOS, and Windows release package plus checksums.
make release VERSION=v1.3.0
```

Place options before the password file and username, as shown below.

### Create and update a password file

```sh
# Create (or replace) a password file and interactively add alice.
# APR1-MD5 is the default encoding.
htpasswd -c ./users.htpasswd alice

# Add bob to an existing file using bcrypt.
htpasswd -B ./users.htpasswd bob

# Change alice's password. Existing unrelated entries are preserved.
htpasswd -B ./users.htpasswd alice
```

`-c` replaces an existing file, so use it only when creating a new password
file. Without `-c`, the file must already exist. Interactive passwords are
requested twice and are hidden while entered in a terminal.

### Select a password encoding

```sh
# APR1-MD5 (the default).
htpasswd -m ./users.htpasswd alice

# bcrypt with the default cost of 10.
htpasswd -B ./users.htpasswd alice

# bcrypt with a custom cost from 4 through 31.
htpasswd -B -C 12 ./users.htpasswd alice

# SHA-1.
htpasswd -s ./users.htpasswd alice

# SHA-256 crypt.
htpasswd -2 ./users.htpasswd alice

# SHA-512 crypt.
htpasswd -5 ./users.htpasswd alice

# Plain text (not recommended).
htpasswd -p ./users.htpasswd alice
```

Only one encoding option may be selected. Bcrypt is recommended for new
password files. SHA-1, APR1-MD5, and plain text are retained for compatibility
with existing systems.

### Non-interactive use

```sh
# Read one password line from standard input.
printf '%s\n' 'secret' | htpasswd -i -B ./users.htpasswd alice

# Read a password from an environment-backed secret without adding it to the
# command-line arguments.
printf '%s\n' "$APP_PASSWORD" | htpasswd -i -B ./users.htpasswd alice

# Supply a password as a command-line argument.
htpasswd -b -B ./users.htpasswd alice 'secret'
```

Prefer `-i` in scripts. With `-b`, the password may be exposed through shell
history, process listings, logs, or CI output.

### Generate an entry without changing a file

```sh
# Prompt interactively and print an APR1-MD5 entry.
htpasswd -n alice

# Read from standard input and print a bcrypt entry.
printf '%s\n' 'secret' | htpasswd -n -i -B alice

# Use a command-line password and print a SHA-512 crypt entry.
htpasswd -n -b -5 alice 'secret'
```

The output can be redirected or appended as needed:

```sh
printf '%s\n' 'secret' | htpasswd -n -i -B alice > ./users.htpasswd
printf '%s\n' 'another-secret' | htpasswd -n -i -B bob >> ./users.htpasswd
```

### Verify and delete users

```sh
# Verify alice interactively.
htpasswd -v ./users.htpasswd alice

# Verify using standard input.
printf '%s\n' 'secret' | htpasswd -v -i ./users.htpasswd alice

# Verify using a command-line password.
htpasswd -v -b ./users.htpasswd alice 'secret'

# Delete alice. All matching entries are removed.
htpasswd -D ./users.htpasswd alice
```

File updates use an atomic replacement, preserve the permissions of an existing
file, and create new files with mode `0600`.
