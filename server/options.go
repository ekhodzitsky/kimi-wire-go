package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ekhodzitsky/kimi-wire/protocol"
)

// Option configures a Server.
type Option func(*Server)

// WithServerInfo sets the server name and version returned by initialize.
func WithServerInfo(name, version string) Option {
	return func(s *Server) { s.info = protocol.ServerInfo{Name: name, Version: version} }
}

// WithSlashCommands sets the slash command list returned by initialize.
func WithSlashCommands(cmds []protocol.SlashCommandInfo) Option {
	return func(s *Server) {
		s.slashCmds = append([]protocol.SlashCommandInfo(nil), cmds...)
	}
}

// WithExternalToolValidator decides whether an external tool is accepted.
func WithExternalToolValidator(fn func(protocol.ExternalTool) error) Option {
	if fn == nil {
		panic("server: nil tool validator")
	}
	return func(s *Server) { s.toolValidator = fn }
}

// WithSupportedHooks declares hook event types the server supports.
func WithSupportedHooks(events []string) Option {
	return func(s *Server) { s.supportedHooks = events }
}

// WithDefaultRequestTimeout sets the default timeout for CallExternalTool / AskQuestion.
func WithDefaultRequestTimeout(d time.Duration) Option {
	return func(s *Server) { s.defaultTimeout = d }
}

// WithLogger sets a structured logger for server diagnostics.
func WithLogger(l *slog.Logger) Option {
	if l == nil {
		panic("server: nil logger")
	}
	return func(s *Server) {
		s.logf = func(format string, args ...any) { l.Info(fmt.Sprintf(format, args...)) }
	}
}
