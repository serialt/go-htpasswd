// Command htpasswd manages Apache-compatible htpasswd files.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	htpasswd "github.com/krabt/go-htpasswd"
)

const usageText = `Usage:
  htpasswd [options] password-file username
  htpasswd -b [options] password-file username password
  htpasswd -n [-b] [options] username [password]
  htpasswd -D password-file username
  htpasswd -v password-file username

Options:
  -c        Create a new file (replace it if it already exists)
  -n        Print the result instead of changing a file
  -b        Read the password from the command line
  -i        Read the password from standard input
  -D        Delete the named user
  -v        Verify the named user's password
  -m        Use APR1-MD5 (default)
  -B        Use bcrypt
  -C cost   Set bcrypt cost (4-31, default 10)
  -s        Use SHA-1
  -2        Use SHA-256 crypt
  -5        Use SHA-512 crypt
  -p        Store plain text (insecure)
`

type config struct {
	create, stdout, batch, stdin, delete, verify bool
	algorithm                                    htpasswd.Algorithm
	cost                                         int
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "htpasswd: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("htpasswd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var cfg config
	var md5Mode, bcryptMode, shaMode, sha256Mode, sha512Mode, plainMode bool
	fs.BoolVar(&cfg.create, "c", false, "create file")
	fs.BoolVar(&cfg.stdout, "n", false, "write to stdout")
	fs.BoolVar(&cfg.batch, "b", false, "password on command line")
	fs.BoolVar(&cfg.stdin, "i", false, "password on stdin")
	fs.BoolVar(&cfg.delete, "D", false, "delete user")
	fs.BoolVar(&cfg.verify, "v", false, "verify user")
	fs.BoolVar(&md5Mode, "m", false, "APR1-MD5")
	fs.BoolVar(&bcryptMode, "B", false, "bcrypt")
	fs.IntVar(&cfg.cost, "C", 10, "bcrypt cost")
	fs.BoolVar(&shaMode, "s", false, "SHA-1")
	fs.BoolVar(&sha256Mode, "2", false, "SHA-256 crypt")
	fs.BoolVar(&sha512Mode, "5", false, "SHA-512 crypt")
	fs.BoolVar(&plainMode, "p", false, "plain text")
	fs.Usage = func() { fmt.Fprint(stderr, usageText) }
	if err := fs.Parse(args); err != nil {
		return err
	}

	algorithms := 0
	cfg.algorithm = htpasswd.AlgorithmAPR1
	choices := []struct {
		selected  bool
		algorithm htpasswd.Algorithm
	}{{md5Mode, htpasswd.AlgorithmAPR1}, {bcryptMode, htpasswd.AlgorithmBcrypt}, {shaMode, htpasswd.AlgorithmSHA}, {sha256Mode, htpasswd.AlgorithmCryptSHA256}, {sha512Mode, htpasswd.AlgorithmCryptSHA512}, {plainMode, htpasswd.AlgorithmPlain}}
	for _, choice := range choices {
		if choice.selected {
			algorithms++
			cfg.algorithm = choice.algorithm
		}
	}
	if algorithms > 1 {
		return errors.New("choose only one password algorithm")
	}
	if cfg.cost < 4 || cfg.cost > 31 {
		return errors.New("bcrypt cost must be between 4 and 31")
	}
	if count(cfg.delete, cfg.verify) > 1 || (cfg.delete && (cfg.create || cfg.stdout || cfg.batch || cfg.stdin)) {
		return errors.New("incompatible operation options")
	}
	if count(cfg.batch, cfg.stdin) > 1 {
		return errors.New("-b and -i cannot be used together")
	}
	if cfg.stdout && (cfg.create || cfg.delete || cfg.verify) {
		return errors.New("-n cannot be combined with -c, -D, or -v")
	}
	if cfg.create && cfg.verify {
		return errors.New("-c and -v cannot be used together")
	}

	positional := fs.Args()
	needFile := !cfg.stdout
	minimum := 1
	if needFile {
		minimum = 2
	}
	if len(positional) < minimum {
		fs.Usage()
		return errors.New("missing password-file or username")
	}
	var filename, username string
	if needFile {
		filename, username = positional[0], positional[1]
		positional = positional[2:]
	} else {
		username = positional[0]
		positional = positional[1:]
	}
	if username == "" || strings.ContainsAny(username, ":\r\n") {
		return errors.New("username must be non-empty and cannot contain ':' or a newline")
	}

	if cfg.delete {
		if len(positional) != 0 {
			return errors.New("delete does not accept a password")
		}
		return deleteUser(filename, username)
	}

	password, err := readPassword(cfg, positional, stdin, stderr, !cfg.verify)
	if err != nil {
		return err
	}
	if cfg.verify {
		file, err := htpasswd.New(filename, htpasswd.DefaultSystems, nil)
		if err != nil {
			return err
		}
		if !file.Match(username, password) {
			return errors.New("password verification failed")
		}
		fmt.Fprintf(stdout, "Password for user %s correct.\n", username)
		return nil
	}

	encoded, err := htpasswd.Encode(password, htpasswd.EncodeOptions{Algorithm: cfg.algorithm, Cost: cfg.cost})
	if err != nil {
		return err
	}
	line := username + ":" + encoded
	if cfg.stdout {
		fmt.Fprintln(stdout, line)
		return nil
	}
	return updateUser(filename, username, line, cfg.create)
}

func readPassword(cfg config, args []string, stdin io.Reader, stderr io.Writer, confirm bool) (string, error) {
	if cfg.batch {
		if len(args) != 1 {
			return "", errors.New("-b requires exactly one password argument")
		}
		return validatePassword(args[0])
	}
	if len(args) != 0 {
		return "", errors.New("unexpected password argument (use -b)")
	}
	if cfg.stdin {
		password, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		return validatePassword(strings.TrimSuffix(strings.TrimSuffix(password, "\n"), "\r"))
	}

	reader := bufio.NewReader(stdin)
	first, err := promptPassword(reader, stdin, stderr, "New password: ")
	if cfg.verify {
		first, err = promptPassword(reader, stdin, stderr, "Password: ")
	}
	if err != nil {
		return "", err
	}
	if confirm {
		second, err := promptPassword(reader, stdin, stderr, "Re-type new password: ")
		if err != nil {
			return "", err
		}
		if first != second {
			return "", errors.New("passwords do not match")
		}
	}
	return validatePassword(first)
}

func promptPassword(reader *bufio.Reader, stdin io.Reader, stderr io.Writer, prompt string) (string, error) {
	fmt.Fprint(stderr, prompt)
	file, isFile := stdin.(*os.File)
	var restore func()
	if isFile && file == os.Stdin && fileStatIsTerminal(file) {
		cmd := exec.Command("stty", "-echo")
		cmd.Stdin = file
		if cmd.Run() == nil {
			restore = func() {
				restoreCmd := exec.Command("stty", "echo")
				restoreCmd.Stdin = file
				_ = restoreCmd.Run()
				fmt.Fprintln(stderr)
			}
			defer restore()
		}
	}
	password, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(password, "\n"), "\r"), nil
}

func fileStatIsTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func validatePassword(password string) (string, error) {
	if strings.ContainsAny(password, "\r\n") {
		return "", errors.New("password cannot contain a newline")
	}
	return password, nil
}

func updateUser(filename, username, replacement string, create bool) error {
	var lines []string
	if !create {
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		lines = splitLines(data)
	}
	updated := false
	result := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		if lineUsername(line) == username {
			if !updated {
				result = append(result, replacement)
				updated = true
			}
			continue
		}
		result = append(result, line)
	}
	if !updated {
		result = append(result, replacement)
	}
	return atomicWrite(filename, []byte(strings.Join(result, "\n")+"\n"))
}

func deleteUser(filename, username string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := splitLines(data)
	result := make([]string, 0, len(lines))
	found := false
	for _, line := range lines {
		if lineUsername(line) == username {
			found = true
			continue
		}
		result = append(result, line)
	}
	if !found {
		return fmt.Errorf("user %q not found", username)
	}
	contents := ""
	if len(result) > 0 {
		contents = strings.Join(result, "\n") + "\n"
	}
	return atomicWrite(filename, []byte(contents))
}

func atomicWrite(filename string, data []byte) error {
	mode := os.FileMode(0o600)
	if info, err := os.Stat(filename); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(filename)
	tmp, err := os.CreateTemp(dir, ".htpasswd-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filename)
}

func splitLines(data []byte) []string {
	text := strings.TrimSuffix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func lineUsername(line string) string {
	username, _, ok := strings.Cut(line, ":")
	if !ok {
		return ""
	}
	return username
}

func count(values ...bool) int {
	total := 0
	for _, value := range values {
		if value {
			total++
		}
	}
	return total
}
