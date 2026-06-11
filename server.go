package wire

import "github.com/ekhodzitsky/kimi-wire/server"

// Server-side types.
type Server = server.Server
type Agent = server.Agent
type Emitter = server.Emitter
type Turn = server.Turn

// Optional agent capabilities.
type Steerer = server.Steerer
type Replayer = server.Replayer
type PlanModeSwitcher = server.PlanModeSwitcher

// Server options and constructor.
type ServerOption = server.Option

var NewServer = server.New
var WithServerInfo = server.WithServerInfo
var WithSlashCommands = server.WithSlashCommands
var WithExternalToolValidator = server.WithExternalToolValidator
var WithSupportedHooks = server.WithSupportedHooks
var WithDefaultRequestTimeout = server.WithDefaultRequestTimeout
var WithLogger = server.WithLogger
