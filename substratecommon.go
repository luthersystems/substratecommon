package substratecommon

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"

	hclog "github.com/hashicorp/go-hclog"
)

// ConcreteRequestOptions is a variant of RequestOptions that is
// "flattened" to pure data.
type ConcreteRequestOptions struct {
	Headers             map[string]string
	Endpoint            string
	ID                  string
	AuthToken           string
	Params              []byte
	Transient           map[string][]byte
	Timestamp           string
	MSPFilter           []string
	MinEndorsers        int
	Creator             string
	DependentTxID       string
	DisableWritePolling bool
	CCFetchURLDowngrade bool
	CCFetchURLProxy     string
}

// RequestOptions are operated on by the Config functions generated by
// the With* functions.
type RequestOptions struct {
	Log                 *logrus.Logger // not flat!
	LogFields           logrus.Fields  // not flat!
	Headers             map[string]string
	Endpoint            string
	ID                  string
	AuthToken           string
	Params              interface{} // not flat!
	Transient           map[string][]byte
	Target              *interface{}                 // not flat!
	TimestampGenerator  func(context.Context) string // not flat!
	MSPFilter           []string
	MinEndorsers        int
	Creator             string
	Ctx                 context.Context // not flat!
	DependentTxID       string
	DisableWritePolling bool
	CCFetchURLDowngrade bool
	CCFetchURLProxy     string
}

// Config is a type for a function that can mutate a requestOptions
// object.
type Config func(*RequestOptions)

// WithContext allows specifying the context to use.
func WithContext(ctx context.Context) Config {
	return func(r *RequestOptions) {
		r.Ctx = ctx
	}
}

// WithLog allows specifying the logger to use.
func WithLog(log *logrus.Logger) Config {
	return func(r *RequestOptions) {
		r.Log = log
	}
}

// WithLogField allows specifying a log field to be included.
func WithLogField(key string, value interface{}) Config {
	return func(r *RequestOptions) {
		r.LogFields[key] = value
	}
}

// WithLogrusFields allows specifying multiple log fields to be
// included.
func WithLogrusFields(fields logrus.Fields) Config {
	return func(r *RequestOptions) {
		for k, v := range fields {
			r.LogFields[k] = v
		}
	}
}

// WithHeader allows specifying an additional HTTP header.
func WithHeader(key string, value string) Config {
	return func(r *RequestOptions) {
		r.Headers[key] = value
	}
}

// WithEndpoint allows specifying the endpoint to target. The RPC
// implementation will not work if an endpoint is not specified.
func WithEndpoint(endpoint string) Config {
	return func(r *RequestOptions) {
		r.Endpoint = endpoint
	}
}

// WithID allows specifying the request ID. If the request ID is not
// specified, a randomly-generated UUID will be used.
func WithID(id string) Config {
	return func(r *RequestOptions) {
		r.ID = id
	}
}

// WithParams allows specifying the phylum "parameters" argument. This
// must be set to something that json.Marshal accepts.
func WithParams(params interface{}) Config {
	return func(r *RequestOptions) {
		r.Params = params
	}
}

// WithTransientData allows specifying a single "transient data"
// key-value pair.
func WithTransientData(key string, val []byte) Config {
	return func(r *RequestOptions) {
		r.Transient[key] = val
	}
}

// WithTransientDataMap allows specifying multiple "transient data"
// key-value pairs.
func WithTransientDataMap(data map[string][]byte) Config {
	return func(r *RequestOptions) {
		for key, val := range data {
			r.Transient[key] = val
		}
	}
}

// WithResponse allows capturing the RPC response for futher analysis.
func WithResponse(target *interface{}) Config {
	return func(r *RequestOptions) {
		r.Target = target
	}
}

// WithAuthToken passes authorization for the transaction issuer with a request
func WithAuthToken(token string) Config {
	return func(r *RequestOptions) {
		r.AuthToken = token
	}
}

// WithTimestampGenerator allows specifying a function that will be
// invoked at every Upgrade, Init, and Call whose output is used to
// set the substrate "now" timestamp in mock mode. Has no effect
// outside of mock mode.
func WithTimestampGenerator(timestampGenerator func(context.Context) string) Config {
	return func(r *RequestOptions) {
		r.TimestampGenerator = timestampGenerator
	}
}

// WithMSPFilter allows specifying the MSP filter. Has no effect in
// mock mode.
func WithMSPFilter(mspFilter []string) Config {
	clonedMSPFilter := append([]string(nil), mspFilter...)
	return (func(r *RequestOptions) {
		r.MSPFilter = clonedMSPFilter
	})
}

// WithMinEndorsers allows specifying the minimum number of endorsing
// peers. Has no effect in mock mode.
func WithMinEndorsers(minEndorsers int) Config {
	return (func(r *RequestOptions) {
		r.MinEndorsers = minEndorsers
	})
}

// WithCreator allows specifying the creator. Only has effect in mock
// mode. Also works in gateway mock mode.
func WithCreator(creator string) Config {
	return (func(r *RequestOptions) {
		r.Creator = creator
	})
}

// WithDependentTxID allows specifying a dependency on a transaction ID.  If
// set, the client will poll for the presence of that transaction before
// simulating the request on the peer with the transaction.
func WithDependentTxID(txID string) Config {
	return (func(r *RequestOptions) {
		r.DependentTxID = txID
	})
}

// WithConditionalDependentTxID allows specifying a conditional dependency on a
// transaction ID when polling is disabled or transaction dependencies are
// already enabled.  This is intended for use with chained sequential calls that
// have a critical dependency.
func WithConditionalDependentTxID(txID string) Config {
	return (func(r *RequestOptions) {
		if r.DisableWritePolling || r.DependentTxID != "" {
			r.DependentTxID = txID
		}
	})
}

// WithDisableWritePolling allows disabling polling for full consensus after a
// write is committed.
func WithDisableWritePolling(disable bool) Config {
	return (func(r *RequestOptions) {
		r.DisableWritePolling = disable
	})
}

// WithCCFetchURLDowngrade allows controlling https -> http downgrade,
// typically useful before proxying for ccfetchurl library.
func WithCCFetchURLDowngrade(downgrade bool) Config {
	return (func(r *RequestOptions) {
		r.CCFetchURLDowngrade = downgrade
	})
}

// WithCCFetchURLProxy sets the proxy for ccfetchurl library.
func WithCCFetchURLProxy(proxy string) Config {
	return (func(r *RequestOptions) {
		r.CCFetchURLProxy = proxy
	})
}

func tsg(context context.Context, tg func(context.Context) string) string {
	if tg != nil {
		return tg(context)
	} else {
		return time.Now().UTC().Format(time.RFC3339)
	}
}

// FlattenOptions will flatten a list of config options.
func FlattenOptions(configs ...Config) (*ConcreteRequestOptions, error) {
	opt := &RequestOptions{
		LogFields: logrus.Fields{},
		Headers:   map[string]string{},
		Transient: map[string][]byte{},
		Params:    []interface{}{},
	}

	for _, config := range configs {
		config(opt)
	}

	params, err := json.Marshal(opt.Params)
	if err != nil {
		return nil, err
	}

	return &ConcreteRequestOptions{
		Headers:             opt.Headers,
		Endpoint:            opt.Endpoint,
		ID:                  opt.ID,
		AuthToken:           opt.AuthToken,
		Params:              params,
		Transient:           opt.Transient,
		Timestamp:           tsg(opt.Ctx, opt.TimestampGenerator),
		MSPFilter:           opt.MSPFilter,
		MinEndorsers:        opt.MinEndorsers,
		Creator:             opt.Creator,
		DependentTxID:       opt.DependentTxID,
		DisableWritePolling: opt.DisableWritePolling,
		CCFetchURLDowngrade: opt.CCFetchURLDowngrade,
		CCFetchURLProxy:     opt.CCFetchURLProxy,
	}, nil
}

// FlattenContext will return the context selected by a list of config
// options.
func FlattenContext(configs ...Config) (context.Context, error) {
	opt := &RequestOptions{
		LogFields: logrus.Fields{},
		Headers:   map[string]string{},
		Transient: map[string][]byte{},
		Params:    []interface{}{},
	}

	for _, config := range configs {
		config(opt)
	}

	if opt.Ctx == nil {
		return nil, fmt.Errorf("expected context")
	}

	return opt.Ctx, nil
}

// Error represents a possible error. IsTimeoutError indicates whether
// the error was a timeout error.
type Error struct {
	IsTimeoutError bool
	Diagnostic     string
}

func (e Error) Error() string {
	return e.Diagnostic
}

// Response represents a shiroclient response.
type Response struct {
	ResultJSON    []byte
	HasError      bool
	ErrorCode     int
	ErrorMessage  string
	ErrorJSON     []byte
	TransactionID string
}

// UnmarshalTo unmarshals the response's result to dst.
func (s *Response) UnmarshalTo(dst interface{}) error {
	message, ok := dst.(proto.Message)
	if ok {
		return jsonpb.Unmarshal(bytes.NewReader([]byte(s.ResultJSON)), message)
	}
	return json.Unmarshal([]byte(s.ResultJSON), dst)
}

// Transaction represents summary information about a transaction.
type Transaction struct {
	ID          string
	Reason      string
	Event       []byte
	ChaincodeID string
}

// Block represents summary information about a block.
type Block struct {
	Hash         string
	Transactions []*Transaction
}

// Substrate is the interface that we're exposing as a plugin.
type Substrate interface {
	NewRPC() (string, error)
	CloseRPC(string) error

	NewMockFrom(string, string, []byte) (string, error)
	SetCreatorWithAttributesMock(string, string, map[string]string) error
	SnapshotMock(string) ([]byte, error)
	CloseMock(string) error

	Init(string, string, *ConcreteRequestOptions) error
	Call(string, string, *ConcreteRequestOptions) (*Response, error)
	QueryInfo(string, *ConcreteRequestOptions) (uint64, error)
	QueryBlock(string, uint64, *ConcreteRequestOptions) (*Block, error)

	// IsTimeoutError doesn't use RPC
	IsTimeoutError(err error) bool
}

// ArgsNewRPC encodes the arguments to NewRPC
type ArgsNewRPC struct {
}

// RespNewRPC encodes the response from NewRPC
type RespNewRPC struct {
	Tag string
	Err *Error
}

// ArgsCloseRPC encodes the arguments to CloseRPC
type ArgsCloseRPC struct {
	Tag string
}

// RespCloseRPC encodes the response from CloseRPC
type RespCloseRPC struct {
	Err *Error
}

// ArgsNewMockFrom encodes the arguments to NewMockFrom
type ArgsNewMockFrom struct {
	Name     string
	Version  string
	Snapshot []byte
}

// RespNewMockFrom encodes the response from NewMockFrom
type RespNewMockFrom struct {
	Tag string
	Err *Error
}

// ArgsSetCreatorWithAttributesMock encodes the arguments to SetCreatorWithAttributesMock
type ArgsSetCreatorWithAttributesMock struct {
	Tag     string
	Creator string
	Attrs   map[string]string
}

// RespSetCreatorWithAttributesMock encodes the response from SetCreatorWithAttributesMock
type RespSetCreatorWithAttributesMock struct {
	Err *Error
}

// ArgsSnapshotMock encodes the arguments to SnapshotMock
type ArgsSnapshotMock struct {
	Tag string
}

// RespSnapshotMock encodes the response from SnapshotMock
type RespSnapshotMock struct {
	Snapshot []byte
	Err      *Error
}

// ArgsCloseMock encodes the arguments to CloseMock
type ArgsCloseMock struct {
	Tag string
}

// RespCloseMock encodes the response from CloseMock
type RespCloseMock struct {
	Err *Error
}

// ArgsInit encodes the arguments to Init
type ArgsInit struct {
	Tag     string
	Phylum  string
	Options *ConcreteRequestOptions
}

// RespInit encodes the response from Init
type RespInit struct {
	Err *Error
}

// ArgsCall encodes the arguments to Call
type ArgsCall struct {
	Tag     string
	Command string
	Options *ConcreteRequestOptions
}

// RespCall encodes the response from Call
type RespCall struct {
	Response *Response
	Err      *Error
}

// ArgsQueryInfo encodes the arguments to QueryInfo
type ArgsQueryInfo struct {
	Tag     string
	Options *ConcreteRequestOptions
}

// RespQueryInfo encodes the response from QueryInfo
type RespQueryInfo struct {
	Height uint64
	Err    *Error
}

// ArgsQueryBlock encodes the arguments to QueryBlock
type ArgsQueryBlock struct {
	Tag     string
	Height  uint64
	Options *ConcreteRequestOptions
}

// RespQueryBlock encodes the response from QueryBlock
type RespQueryBlock struct {
	Block *Block
	Err   *Error
}

// PluginRPC is an implementation that talks over RPC
type PluginRPC struct{ client *rpc.Client }

var errRPC = fmt.Errorf("RPC failure")

// NewRPC forwards the call
func (g *PluginRPC) NewRPC() (string, error) {
	var resp RespNewRPC
	err := g.client.Call("Plugin.NewRPC", &ArgsNewRPC{}, &resp)
	if err != nil {
		return "", errRPC
	}
	if resp.Err != nil {
		return "", resp.Err
	}
	return resp.Tag, nil
}

// CloseRPC forwards the call
func (g *PluginRPC) CloseRPC(tag string) error {
	var resp RespCloseRPC
	err := g.client.Call("Plugin.CloseRPC", &ArgsCloseRPC{Tag: tag}, &resp)
	if err != nil {
		return errRPC
	}
	if resp.Err != nil {
		return resp.Err
	}
	return nil
}

// NewMockFrom forwards the call
func (g *PluginRPC) NewMockFrom(name string, version string, snapshot []byte) (string, error) {
	var resp RespNewMockFrom
	err := g.client.Call("Plugin.NewMockFrom", &ArgsNewMockFrom{Name: name, Version: version, Snapshot: snapshot}, &resp)
	if err != nil {
		return "", errRPC
	}
	if resp.Err != nil {
		return "", resp.Err
	}
	return resp.Tag, nil
}

// SetCreatorWithAttributesMock forwards the call
func (g *PluginRPC) SetCreatorWithAttributesMock(tag string, creator string, attrs map[string]string) error {
	var resp RespSetCreatorWithAttributesMock
	err := g.client.Call("Plugin.SetCreatorWithAttributesMock", &ArgsSetCreatorWithAttributesMock{Tag: tag, Creator: creator, Attrs: attrs}, &resp)
	if err != nil {
		return errRPC
	}
	if resp.Err != nil {
		return resp.Err
	}
	return nil
}

// SnapshotMock forwards the call
func (g *PluginRPC) SnapshotMock(tag string) ([]byte, error) {
	var resp RespSnapshotMock
	err := g.client.Call("Plugin.SnapshotMock", &ArgsSnapshotMock{Tag: tag}, &resp)
	if err != nil {
		return nil, errRPC
	}
	if resp.Err != nil {
		return nil, resp.Err
	}
	return resp.Snapshot, nil
}

// CloseMock forwards the call
func (g *PluginRPC) CloseMock(tag string) error {
	var resp RespCloseMock
	err := g.client.Call("Plugin.CloseMock", &ArgsCloseMock{Tag: tag}, &resp)
	if err != nil {
		return errRPC
	}
	if resp.Err != nil {
		return resp.Err
	}
	return nil
}

// Init forwards the call
func (g *PluginRPC) Init(tag string, phylum string, options *ConcreteRequestOptions) error {
	var resp RespInit
	err := g.client.Call("Plugin.Init", &ArgsInit{Tag: tag, Phylum: phylum, Options: options}, &resp)
	if err != nil {
		return errRPC
	}
	if resp.Err != nil {
		return resp.Err
	}
	return nil
}

// Call forwards the call
func (g *PluginRPC) Call(tag string, command string, options *ConcreteRequestOptions) (*Response, error) {
	var resp RespCall
	err := g.client.Call("Plugin.Call", &ArgsCall{Tag: tag, Command: command, Options: options}, &resp)
	if err != nil {
		return nil, errRPC
	}
	if resp.Err != nil {
		return nil, resp.Err
	}
	return resp.Response, nil
}

// QueryInfo forwards the call
func (g *PluginRPC) QueryInfo(tag string, options *ConcreteRequestOptions) (uint64, error) {
	var resp RespQueryInfo
	err := g.client.Call("Plugin.QueryInfo", &ArgsQueryInfo{Tag: tag, Options: options}, &resp)
	if err != nil {
		return 0, errRPC
	}
	if resp.Err != nil {
		return 0, resp.Err
	}
	return resp.Height, nil
}

// QueryBlock forwards the call
func (g *PluginRPC) QueryBlock(tag string, height uint64, options *ConcreteRequestOptions) (*Block, error) {
	var resp RespQueryBlock
	err := g.client.Call("Plugin.QueryInfo", &ArgsQueryBlock{Tag: tag, Height: height, Options: options}, &resp)
	if err != nil {
		return nil, errRPC
	}
	if resp.Err != nil {
		return nil, resp.Err
	}
	return resp.Block, nil
}

// IsTimeoutError checks if the error is a timeout error. This is done locally.
func (g *PluginRPC) IsTimeoutError(err error) bool {
	if e, ok := err.(Error); ok {
		return e.IsTimeoutError
	}
	return false
}

// PluginRPCServer is the RPC server that PluginRPC talks to,
// conforming to the requirements of net/rpc
type PluginRPCServer struct {
	// This is the real implementation
	Impl Substrate
}

func (s *PluginRPCServer) newError(err error) *Error {
	b := s.Impl.IsTimeoutError(err)
	return &Error{IsTimeoutError: b, Diagnostic: err.Error()}
}

// NewRPC forwards the call
func (s *PluginRPCServer) NewRPC(args *ArgsNewRPC, resp *RespNewRPC) error {
	tag, err := s.Impl.NewRPC()
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	resp.Tag = tag
	return nil
}

// CloseRPC forwards the call
func (s *PluginRPCServer) CloseRPC(args *ArgsCloseRPC, resp *RespCloseRPC) error {
	err := s.Impl.CloseRPC(args.Tag)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	return nil
}

// NewMockFrom forwards the call
func (s *PluginRPCServer) NewMockFrom(args *ArgsNewMockFrom, resp *RespNewMockFrom) error {
	tag, err := s.Impl.NewMockFrom(args.Name, args.Version, args.Snapshot)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	resp.Tag = tag
	return nil
}

// SetCreatorWithAttributesMock forwards the call
func (s *PluginRPCServer) SetCreatorWithAttributesMock(args *ArgsSetCreatorWithAttributesMock, resp *RespSetCreatorWithAttributesMock) error {
	err := s.Impl.SetCreatorWithAttributesMock(args.Tag, args.Creator, args.Attrs)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	return nil
}

// SnapshotMock forwards the call
func (s *PluginRPCServer) SnapshotMock(args *ArgsSnapshotMock, resp *RespSnapshotMock) error {
	dat, err := s.Impl.SnapshotMock(args.Tag)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	resp.Snapshot = dat
	return nil
}

// CloseMock forwards the call
func (s *PluginRPCServer) CloseMock(args *ArgsCloseMock, resp *RespCloseMock) error {
	err := s.Impl.CloseMock(args.Tag)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	return nil
}

// Init forwards the call
func (s *PluginRPCServer) Init(args *ArgsInit, resp *RespInit) error {
	err := s.Impl.Init(args.Tag, args.Phylum, args.Options)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	return nil
}

// Call forwards the call
func (s *PluginRPCServer) Call(args *ArgsCall, resp *RespCall) error {
	res, err := s.Impl.Call(args.Tag, args.Command, args.Options)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	resp.Response = res
	return nil
}

// QueryInfo forwards the call
func (s *PluginRPCServer) QueryInfo(args *ArgsQueryInfo, resp *RespQueryInfo) error {
	height, err := s.Impl.QueryInfo(args.Tag, args.Options)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	resp.Height = height
	return nil
}

// QueryBlock forwards the call
func (s *PluginRPCServer) QueryBlock(args *ArgsQueryBlock, resp *RespQueryBlock) error {
	block, err := s.Impl.QueryBlock(args.Tag, args.Height, args.Options)
	if err != nil {
		resp.Err = s.newError(err)
		return nil
	}
	resp.Block = block
	return nil
}

// Plugin is the implementation of plugin.Plugin so we can
// serve/consume this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type Plugin struct {
	// Impl Injection
	Impl Substrate
}

// Server returns an RPC server for this plugin type. We construct a
// PluginRPCServer for this.
func (p *Plugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: p.Impl}, nil
}

// Client returns an implementation of our interface that communicates
// over an RPC client. We return PluginRPC for this.
func (Plugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPC{client: c}, nil
}

// EncodePhylumBytes encodes a phylum in the manner expected by
// mock substrate.
func EncodePhylumBytes(phylum string) string {
	return base64.StdEncoding.EncodeToString([]byte(phylum))
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SUBSTRATEHCP1",
	MagicCookieValue: "substratehcp1",
}

// pluginMap is the map of plugins we can dispense.
var pluginMap = map[string]plugin.Plugin{
	"substrate": &Plugin{},
}

type connectOption struct {
	level        hclog.Level
	command      string
	attachStdamp io.Writer
}

// ConnectOption represents the type of a builder action for connectOption
type ConnectOption func(co *connectOption) error

// ConnectWithLogLevel specifies the log level to use (the default is Debug)
func ConnectWithLogLevel(level hclog.Level) func(co *connectOption) error {
	return (func(co *connectOption) error {
		co.level = level
		return nil
	})
}

// ConnectWithCommand specifies the path to the plugin (the default is "")
func ConnectWithCommand(command string) func(co *connectOption) error {
	return (func(co *connectOption) error {
		co.command = command
		return nil
	})
}

// ConnectWithAttachStdamp specifies an io.Writer to receive stdio output from the plugin
func ConnectWithAttachStdamp(attachStdamp io.Writer) func(co *connectOption) error {
	return (func(co *connectOption) error {
		co.attachStdamp = attachStdamp
		return nil
	})
}

type SubstrateConnection struct {
	client    *plugin.Client
	substrate Substrate
}

// NewSubstrateConnection connects to a plugin in the background.
func NewSubstrateConnection(opts ...ConnectOption) (*SubstrateConnection, error) {
	co := &connectOption{level: hclog.Debug, attachStdamp: nil}

	for _, opt := range opts {
		if err := opt(co); err != nil {
			panic(err)
		}
	}

	// Create an hclog.Logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugin",
		Output: os.Stdout,
		Level:  co.level,
	})

	// We're a host! Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
		Cmd:             exec.Command(co.command),
		Logger:          logger,
		Stderr:          co.attachStdamp,
		SyncStdout:      co.attachStdamp,
		SyncStderr:      co.attachStdamp,
	})

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		log.Fatal(err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("substrate")
	if err != nil {
		log.Fatal(err)
	}

	// This feels like a normal interface implementation but is in
	// fact over an RPC connection.
	substrate := raw.(Substrate)

	return &SubstrateConnection{client: client, substrate: substrate}, nil
}

// GetSubstrate returns the Substrate interface associated with a
// connection.
func (s *SubstrateConnection) GetSubstrate() Substrate {
	return s.substrate
}

// Close closes a connection.
func (s *SubstrateConnection) Close() error {
	s.client.Kill()
	return nil
}

// Connect connects to a plugin synchronously; all operations on the
// Substrate interface must be performed from within the passed
// closure.
func Connect(user func(Substrate) error, opts ...ConnectOption) error {
	conn, err := NewSubstrateConnection(opts...)
	if err != nil {
		return err
	}

	err = user(conn.GetSubstrate())
	if err != nil {
		return err
	}

	return conn.Close()
}
