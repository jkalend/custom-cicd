package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// runCommand executes a shell command with a timeout, capturing combined
// output into step.Output. On failure, step.Error gets the tail of the
// output and the returned error describes the cause.
func runCommand(ctx context.Context, command string, timeoutSec int, step *Step) error {
	if timeoutSec <= 0 {
		timeoutSec = 300
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	step.Output = stdout.String()
	if step.Output == "" && stderr.Len() > 0 {
		step.Output = stderr.String()
	}

	if err == nil {
		return nil
	}

	var detail string
	if ctx.Err() == context.DeadlineExceeded {
		detail = fmt.Sprintf("command timed out after %d seconds", timeoutSec)
	} else {
		detail = tail(stderr.String(), 2000)
		if detail == "" {
			detail = tail(stdout.String(), 2000)
		}
		if detail == "" {
			detail = err.Error()
		}
	}
	step.Error = detail
	return errors.New(detail)
}

// tail returns the last n characters of s.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
