package caddyexec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyevents"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler implements an event handler that runs a command/program.
// By default, commands are run in the background so as to not
// block the Caddy goroutine.
type Handler struct {
	// The command to execute.
	Command string `json:"command,omitempty"`

	// Arguments to the command. Placeholders are expanded
	// in arguments, so use caution to not introduce any
	// security vulnerabilities with the command.
	Args []string `json:"args,omitempty"`

	// The directory in which to run the command.
	Dir string `json:"dir,omitempty"`

	// How long to wait for the command to terminate
	// before forcefully closing it. Default: 30s
	Timeout caddy.Duration `json:"timeout,omitempty"`

	// If true, runs the command in the foreground,
	// which will block and wait for output. Only
	// do this if you know the command will finish
	// quickly! Required if you want to abort the
	// event.
	Foreground bool `json:"foreground,omitempty"`

	// If the command exits with any of these codes, the
	// event will be signaled to abort with the error.
	// Must be running in the foreground to apply.
	AbortCodes []int `json:"abort_codes,omitempty"`

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "events.handlers.exec",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up the module.
func (eh *Handler) Provision(ctx caddy.Context) error {
	eh.logger = ctx.Logger(eh)

	if eh.Timeout <= 0 {
		eh.Timeout = caddy.Duration(30 * time.Second)
	}

	if len(eh.AbortCodes) > 0 && !eh.Foreground {
		return fmt.Errorf("must run commands in foreground to apply abort codes")
	}

	return nil
}

// Handle handles the event.
func (eh *Handler) Handle(ctx context.Context, e caddyevents.Event) error {
	repl := ctx.Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	// expand placeholders in command args;
	// notably, we do not expand placeholders
	// in the command itself for safety reasons
	expandedArgs := make([]string, len(eh.Args))
	for i := range eh.Args {
		expandedArgs[i] = repl.ReplaceAll(eh.Args[i], "")
	}

	if eh.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(eh.Timeout))
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, eh.Command, expandedArgs...)
	cmd.Dir = eh.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if eh.Foreground {
		err := cmd.Run()

		exitCode := cmd.ProcessState.ExitCode()
		for _, abortCode := range eh.AbortCodes {
			if exitCode == abortCode {
				return fmt.Errorf("%w: %v", caddyevents.ErrAborted, err)
			}
		}

		return err
	}

	go func() {
		if err := cmd.Run(); err != nil {
			eh.logger.Error("background command failed", zap.Error(err))
		}
	}()
	return nil
}

// UnmarshalCaddyfile parses the module's Caddyfile config. Syntax:
//
//	exec <command> <args...>
func (eh *Handler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.NextArg() {
			return d.ArgErr()
		}
		eh.Command = d.Val()
		eh.Args = d.RemainingArgs()
	}
	return nil
}

// Interface guards
var (
	_ caddyfile.Unmarshaler = (*Handler)(nil)
	_ caddy.Provisioner     = (*Handler)(nil)
)
