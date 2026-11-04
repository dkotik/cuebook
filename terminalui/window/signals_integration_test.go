//go:build integration && unix

package window_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/dkotik/cuebook/terminalui/window"
)

const windowSignalHelperEnv = "CUEBOOK_WINDOW_SIGNAL_HELPER"

func TestWindowDefaultSignalsIntegration(t *testing.T) {
	tests := []struct {
		name   string
		signal os.Signal
	}{
		{name: "SIGINT", signal: syscall.SIGINT},
		{name: "SIGTERM", signal: syscall.SIGTERM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			readyReader, readyWriter, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer readyReader.Close()
			defer readyWriter.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable,
				"-test.run=^TestWindowDefaultSignalsHelperProcess$", "-test.timeout=12s")
			cmd.Env = append(os.Environ(), windowSignalHelperEnv+"=1")
			cmd.ExtraFiles = []*os.File{readyWriter}
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.WaitDelay = time.Second
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}

			exited := make(chan struct{})
			var waitErr error
			go func() {
				waitErr = cmd.Wait()
				close(exited)
			}()
			t.Cleanup(func() {
				cancel()
				readyReader.Close()
				<-exited
				if t.Failed() {
					t.Logf("helper output:\n%s", output.String())
				}
			})
			if err := readyWriter.Close(); err != nil {
				t.Fatal(err)
			}

			ready := make(chan error, 1)
			go func() {
				line, err := bufio.NewReader(readyReader).ReadString('\n')
				if err == nil && line != "ready\n" {
					err = fmt.Errorf("unexpected readiness message %q", line)
				}
				ready <- err
			}()
			select {
			case err := <-ready:
				if err != nil {
					t.Fatalf("wait for helper readiness: %v", err)
				}
			case <-ctx.Done():
				t.Fatalf("wait for helper readiness: %v", ctx.Err())
			}

			// Event-loop readiness can precede asynchronous signal registration.
			// Retry only this child's PID; the guard keeps early signals harmless.
			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()
		waitForExit:
			for {
				if err := cmd.Process.Signal(tt.signal); err != nil && !errors.Is(err, os.ErrProcessDone) {
					t.Fatalf("send %s to helper: %v", tt.name, err)
				}
				select {
				case <-exited:
					break waitForExit
				case <-ctx.Done():
					t.Fatalf("wait for helper exit: %v", ctx.Err())
				case <-ticker.C:
				}
			}
			if waitErr != nil {
				t.Fatalf("helper did not exit cleanly: %v", waitErr)
			}
			if !cmd.ProcessState.Success() || cmd.ProcessState.ExitCode() != 0 {
				t.Fatalf("helper exit state: %s", cmd.ProcessState)
			}
			if !strings.Contains(output.String(), "Bubble Tea Run returned nil\n") {
				t.Fatal("helper did not confirm that Run returned nil")
			}
		})
	}
}

func TestWindowDefaultSignalsHelperProcess(t *testing.T) {
	if os.Getenv(windowSignalHelperEnv) != "1" {
		t.Skip("only runs in an isolated helper process")
	}
	readyWriter := os.NewFile(3, "window-signal-readiness")
	if readyWriter == nil {
		t.Fatal("missing readiness pipe")
	}
	defer readyWriter.Close()

	// Prevent default termination before Bubble Tea registers its handler.
	// Never drain this guard or use it to quit; keep it through process exit
	// so retries after Bubble Tea unregisters its handler are harmless too.
	guard := make(chan os.Signal, 1)
	signal.Notify(guard, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	model := &windowSignalModel{readyWriter: readyWriter}
	ui, err := window.New(window.WithCommandContext(ctx), window.WithInitialModels(model))
	if err != nil {
		t.Fatal(err)
	}
	p := tea.NewProgram(ui,
		tea.WithContext(ctx),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
	)
	if _, err := p.Run(); err != nil {
		t.Fatalf("Bubble Tea Run: %v", err)
	}
	if model.err != nil {
		t.Fatalf("helper readiness: %v", model.err)
	}
	if !model.ready {
		t.Fatal("Run returned before the event loop reported readiness")
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("helper context expired instead of signal-driven quit: %v", err)
	}
	fmt.Println("Bubble Tea Run returned nil")
}

type windowSignalReadyMsg struct{}

type windowSignalModel struct {
	readyWriter io.Writer
	ready       bool
	err         error
}

func (m *windowSignalModel) Init() (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		return windowSignalReadyMsg{}
	}
}

func (m *windowSignalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(windowSignalReadyMsg); ok {
		_, m.err = io.WriteString(m.readyWriter, "ready\n")
		m.ready = m.err == nil
	}
	return m, nil
}

func (m *windowSignalModel) View() string { return "" }
