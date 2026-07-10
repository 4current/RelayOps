package agent

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func runServiceCommand(ctx context.Context, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("empty service command")
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("service command timed out")
	}

	if err != nil {
		return output, fmt.Errorf("%s failed: %w", argv[0], err)
	}

	return output, nil
}
