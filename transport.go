package wire

import "github.com/ekhodzitsky/kimi-wire/transport"

type Transport = transport.Transport
type ChannelTransport = transport.ChannelTransport
type InMemoryTransport = transport.InMemoryTransport
type SpawnOptions = transport.SpawnOptions
type ChildProcessTransport = transport.ChildProcessTransport

var NewChannelTransportPair = transport.NewChannelTransportPair
var NewInMemoryTransport = transport.NewInMemoryTransport
var SpawnChildProcessTransport = transport.SpawnChildProcessTransport
