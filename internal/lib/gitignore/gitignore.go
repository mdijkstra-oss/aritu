package gitignore

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// notARepository is what git answers when it was run outside a working tree,
// which is not a failure here: there are no ignore rules to respect.
const notARepository = 128

// Ignored answers for every path at once because git is a process, and returns
// the paths as they were given. A directory holding no repository, or a machine
// holding no git, ignores nothing.
func Ignored(dir string, paths []string) (map[string]struct{}, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	command := exec.Command("git", "check-ignore", "-z", "--stdin")
	command.Dir = dir
	command.Stdin = strings.NewReader(nulTerminated(paths))
	var answer, complaint bytes.Buffer
	command.Stdout = &answer
	command.Stderr = &complaint

	if err := checkRan(command.Run(), complaint.String()); err != nil {
		return nil, err
	}
	return setOf(answer.String()), nil
}

func checkRan(err error, complaint string) error {
	if err == nil {
		return nil
	}
	var exited *exec.ExitError
	if errors.As(err, &exited) {
		if isTolerable(exited.ExitCode()) {
			return nil
		}
		return fmt.Errorf("git check-ignore: %s", strings.TrimSpace(complaint))
	}
	if errors.Is(err, exec.ErrNotFound) {
		return nil
	}
	return fmt.Errorf("git check-ignore: %w", err)
}

// git answers 1 when it ignores none of the paths, which is an answer rather
// than a failure.
func isTolerable(code int) bool {
	return code == 1 || code == notARepository
}

func nulTerminated(paths []string) string {
	var written strings.Builder
	for _, path := range paths {
		written.WriteString(path)
		written.WriteByte(0)
	}
	return written.String()
}

func setOf(answer string) map[string]struct{} {
	ignored := make(map[string]struct{})
	for path := range strings.SplitSeq(answer, "\x00") {
		if path != "" {
			ignored[path] = struct{}{}
		}
	}
	return ignored
}
