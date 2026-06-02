package media

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func formatToolFailure(tool string, err error, stderr string) error {
	if isMissingExecutable(err) {
		return missingToolError(tool)
	}
	stderr = strings.TrimSpace(stderr)
	if stderr != "" {
		return fmt.Errorf("%s failed: %s", tool, stderr)
	}
	return fmt.Errorf("%s failed: %w", tool, err)
}

func formatToolStartError(tool string, err error) error {
	if isMissingExecutable(err) {
		return missingToolError(tool)
	}
	return err
}

func missingToolError(tool string) error {
	return fmt.Errorf("%s is required; install %s and try again", tool, tool)
}

func isMissingExecutable(err error) bool {
	var execErr *exec.Error
	return errors.As(err, &execErr)
}
