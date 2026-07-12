package game

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

var ErrAlreadyRunning = errors.New("game command is already running")

type Action string

const (
	ActionStart   Action = "start"
	ActionStop    Action = "stop"
	ActionRestart Action = "restart"
	ActionStatus  Action = "status"
)

var validActions = map[Action]struct{}{
	ActionStart:   {},
	ActionStop:    {},
	ActionRestart: {},
	ActionStatus:  {},
}

func (a Action) Valid() bool {
	_, ok := validActions[a]
	return ok
}

type Result struct {
	Output   string
	ExitCode int
}

type GameUsecase struct {
	config SSHConfig
	mu     sync.Mutex
}

func NewGameUsecase(config SSHConfig) (*GameUsecase, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &GameUsecase{config: config}, nil
}

func (u *GameUsecase) Execute(ctx context.Context, action Action) (*Result, error) {
	if !action.Valid() {
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	if action != ActionStatus {
		if !u.mu.TryLock() {
			return nil, ErrAlreadyRunning
		}
		defer u.mu.Unlock()
	}

	clientConfig, err := u.config.clientConfig()
	if err != nil {
		return nil, err
	}

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", u.config.address())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ssh host: %w", err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, u.config.address(), clientConfig)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to establish ssh connection: %w", err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create ssh session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	command := fmt.Sprintf("sudo /usr/local/bin/game %s", action)

	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return nil, ctx.Err()
	case err := <-done:
		output := stdout.String()
		if stderr.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				return nil, fmt.Errorf("failed to run command over ssh: %w", err)
			}
		}

		return &Result{Output: output, ExitCode: exitCode}, nil
	}
}
