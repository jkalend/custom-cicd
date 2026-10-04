package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type lineCallback func(stream, line string)

type streamWriter struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	lineBuf bytes.Buffer
	stream  string
	onLine  lineCallback
}

func (w *streamWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf.Write(p)
	if w.onLine == nil {
		return len(p), nil
	}
	for _, b := range p {
		if b == '\n' {
			line := strings.TrimRight(w.lineBuf.String(), "\r")
			w.lineBuf.Reset()
			w.onLine(w.stream, line)
		} else {
			w.lineBuf.WriteByte(b)
		}
	}
	return len(p), nil
}

func (w *streamWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lineBuf.Len() > 0 && w.onLine != nil {
		line := strings.TrimRight(w.lineBuf.String(), "\r")
		w.lineBuf.Reset()
		w.onLine(w.stream, line)
	}
}

func (w *streamWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// runCommand executes a shell command with a timeout, capturing and optionally
// streaming stdout and stderr line-by-line.
func runCommand(ctx context.Context, command string, timeoutSec int, onLine lineCallback) (output string, detail string, err error) {
	if timeoutSec <= 0 {
		timeoutSec = 300
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
		cmd.Cancel = func() error {
			if cmd.Process != nil && cmd.Process.Pid > 0 {
				return exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
			}
			return nil
		}
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	stdout := &streamWriter{stream: "stdout", onLine: onLine}
	stderr := &streamWriter{stream: "stderr", onLine: onLine}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	runErr := cmd.Run()
	stdout.Flush()
	stderr.Flush()

	// Capture combined output preserving both streams
	outStr := stdout.String()
	errStr := stderr.String()
	if outStr != "" && errStr != "" {
		output = outStr + "\n" + errStr
	} else if outStr != "" {
		output = outStr
	} else {
		output = errStr
	}

	if runErr == nil {
		return output, "", nil
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		detail = "command execution cancelled"
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		detail = fmt.Sprintf("command timed out after %d seconds", timeoutSec)
	} else {
		detail = tail(errStr, 2000)
		if detail == "" {
			detail = tail(outStr, 2000)
		}
		if detail == "" {
			detail = runErr.Error()
		}
	}
	return output, detail, errors.New(detail)
}

// tail returns the last n runes of s, safe for multi-byte UTF-8 strings.
func tail(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[len(runes)-n:])
}
