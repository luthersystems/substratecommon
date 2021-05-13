package substratewrapper

import (
	"context"
	"io"

	"github.com/luthersystems/substratecommon"
)

type SubstrateWrapper interface {
	NewRPC() (SubstrateInstanceWrapperRPC, error)
	NewMockFrom(name string, phylumVersion string, blob []byte) (SubstrateInstanceWrapperMock, error)
}

type SubstrateInstanceWrapperCommon interface {
	io.Closer
	HealthCheck(x int) (int, error)
	NewCoherent() SubstrateInstanceWrapperCommon
	NewContextCoherent() SubstrateInstanceWrapperCommon
	IsTimeoutError(err error) bool
	Init(phylum string, configs ...substratecommon.Config) error
	Call(method string, configs ...substratecommon.Config) (*substratecommon.Response, error)
	QueryInfo(configs ...substratecommon.Config) (uint64, error)
	QueryBlock(blockNumber uint64, configs ...substratecommon.Config) (*substratecommon.Block, error)
	GetLastTransactionID() string
}

type SubstrateInstanceWrapperRPC interface {
	SubstrateInstanceWrapperCommon
}

type SubstrateInstanceWrapperMock interface {
	SubstrateInstanceWrapperCommon
	SetCreatorWithAttributes(creator string, attrs map[string]string) error
	Snapshot() ([]byte, error)
}

type substrateWrapper struct {
	substrate substratecommon.Substrate
}

func NewSubstrateWrapper(substrate substratecommon.Substrate) SubstrateWrapper {
	return &substrateWrapper{substrate: substrate}
}

type substrateInstanceWrapperRPC struct {
	substrate substratecommon.Substrate
	tag       string
}

func (sw *substrateWrapper) NewRPC() (SubstrateInstanceWrapperRPC, error) {
	tag, err := sw.substrate.NewRPC()
	if err != nil {
		return nil, err
	}
	return &substrateInstanceWrapperRPC{substrate: sw.substrate, tag: tag}, nil
}

type substrateInstanceWrapperMock struct {
	substrate substratecommon.Substrate
	tag       string
}

func (sw *substrateWrapper) NewMockFrom(name string, phylumVersion string, blob []byte) (SubstrateInstanceWrapperMock, error) {
	tag, err := sw.substrate.NewMockFrom(name, phylumVersion, blob)
	if err != nil {
		return nil, err
	}
	return &substrateInstanceWrapperMock{substrate: sw.substrate, tag: tag}, nil
}

func (siwr *substrateInstanceWrapperRPC) Close() error {
	return siwr.substrate.CloseRPC(siwr.tag)
}

func (siwr *substrateInstanceWrapperRPC) HealthCheck(x int) (int, error) {
	return siwr.substrate.HealthCheck(x)
}

func (siwr *substrateInstanceWrapperRPC) NewCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperCoherent(siwr)
}

func (siwr *substrateInstanceWrapperRPC) NewContextCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperContextCoherent(siwr)
}

func (siwr *substrateInstanceWrapperRPC) IsTimeoutError(err error) bool {
	return siwr.substrate.IsTimeoutError(err)
}

func (siwr *substrateInstanceWrapperRPC) Init(phylum string, configs ...substratecommon.Config) error {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return err
	}
	return siwr.substrate.Init(siwr.tag, phylum, fo)
}

func (siwr *substrateInstanceWrapperRPC) Call(method string, configs ...substratecommon.Config) (*substratecommon.Response, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return nil, err
	}
	return siwr.substrate.Call(siwr.tag, method, fo)
}

func (siwr *substrateInstanceWrapperRPC) QueryInfo(configs ...substratecommon.Config) (uint64, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return 0, err
	}
	return siwr.substrate.QueryInfo(siwr.tag, fo)
}

func (siwr *substrateInstanceWrapperRPC) QueryBlock(blockNumber uint64, configs ...substratecommon.Config) (*substratecommon.Block, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return nil, err
	}
	return siwr.substrate.QueryBlock(siwr.tag, blockNumber, fo)
}

func (siwr *substrateInstanceWrapperRPC) GetLastTransactionID() string {
	return ""
}

func (siwm *substrateInstanceWrapperMock) Close() error {
	return siwm.substrate.CloseMock(siwm.tag)
}

func (siwm *substrateInstanceWrapperMock) HealthCheck(x int) (int, error) {
	return siwm.substrate.HealthCheck(x)
}

func (siwm *substrateInstanceWrapperMock) NewCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperCoherent(siwm)
}

func (siwm *substrateInstanceWrapperMock) NewContextCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperContextCoherent(siwm)
}

func (siwm *substrateInstanceWrapperMock) IsTimeoutError(err error) bool {
	return siwm.substrate.IsTimeoutError(err)
}

func (siwm *substrateInstanceWrapperMock) SetCreatorWithAttributes(creator string, attrs map[string]string) error {
	return siwm.substrate.SetCreatorWithAttributesMock(siwm.tag, creator, attrs)
}

func (siwm *substrateInstanceWrapperMock) Snapshot() ([]byte, error) {
	return siwm.substrate.SnapshotMock(siwm.tag)
}

func (siwm *substrateInstanceWrapperMock) Init(phylum string, configs ...substratecommon.Config) error {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return err
	}
	return siwm.substrate.Init(siwm.tag, phylum, fo)
}

func (siwm *substrateInstanceWrapperMock) Call(method string, configs ...substratecommon.Config) (*substratecommon.Response, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return nil, err
	}
	return siwm.substrate.Call(siwm.tag, method, fo)
}

func (siwm *substrateInstanceWrapperMock) QueryInfo(configs ...substratecommon.Config) (uint64, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return 0, err
	}
	return siwm.substrate.QueryInfo(siwm.tag, fo)
}

func (siwm *substrateInstanceWrapperMock) QueryBlock(blockNumber uint64, configs ...substratecommon.Config) (*substratecommon.Block, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return nil, err
	}
	return siwm.substrate.QueryBlock(siwm.tag, blockNumber, fo)
}

func (siwm *substrateInstanceWrapperMock) GetLastTransactionID() string {
	return ""
}

type substrateInstanceWrapperCoherent struct {
	underlying SubstrateInstanceWrapperCommon
	dependent  string
}

func (siwc *substrateInstanceWrapperCoherent) Close() error {
	return siwc.underlying.Close()
}

func (siwc *substrateInstanceWrapperCoherent) HealthCheck(x int) (int, error) {
	return siwc.underlying.HealthCheck(x)
}

func (siwc *substrateInstanceWrapperCoherent) NewCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperCoherent(siwc)
}

func (siwc *substrateInstanceWrapperCoherent) NewContextCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperContextCoherent(siwc)
}

func (siwc *substrateInstanceWrapperCoherent) IsTimeoutError(err error) bool {
	return siwc.underlying.IsTimeoutError(err)
}

func (siwc *substrateInstanceWrapperCoherent) Init(phylum string, configs ...substratecommon.Config) error {
	return siwc.underlying.Init(phylum, configs...)
}

func (siwc *substrateInstanceWrapperCoherent) Call(method string, configs ...substratecommon.Config) (*substratecommon.Response, error) {
	configs2 := configs
	if siwc.dependent != "" {
		configs2 = append(configs2, substratecommon.WithConditionalDependentTxID(siwc.dependent))
	}
	resp, err := siwc.underlying.Call(method, configs2...)
	if err != nil {
		return nil, err
	}
	siwc.dependent = resp.TransactionID
	return resp, nil
}

func (siwc *substrateInstanceWrapperCoherent) QueryInfo(configs ...substratecommon.Config) (uint64, error) {
	return siwc.underlying.QueryInfo(configs...)
}

func (siwc *substrateInstanceWrapperCoherent) QueryBlock(blockNumber uint64, configs ...substratecommon.Config) (*substratecommon.Block, error) {
	return siwc.underlying.QueryBlock(blockNumber, configs...)
}

func (siwc *substrateInstanceWrapperCoherent) GetLastTransactionID() string {
	return siwc.dependent
}

func NewSubstrateInstanceWrapperCoherent(siwc SubstrateInstanceWrapperCommon) SubstrateInstanceWrapperCommon {
	return &substrateInstanceWrapperCoherent{underlying: siwc}
}

type key int

const (
	// DependentTransactionKey is the key for the context value that stores the DependentWrapper struct
	dependentKey key = iota
)

type dependentWrapper struct {
	dependent string
}

func ContextWithTransactionID(ctx context.Context) context.Context {
	return context.WithValue(ctx, dependentKey, &dependentWrapper{})
}

func GetContextTransactionID(ctx context.Context) string {
	dw, ok := ctx.Value(dependentKey).(*dependentWrapper)
	if ok {
		return dw.dependent
	}
	return ""
}

type substrateInstanceWrapperContextCoherent struct {
	underlying SubstrateInstanceWrapperCommon
}

func (siwc *substrateInstanceWrapperContextCoherent) Close() error {
	return siwc.underlying.Close()
}

func (siwc *substrateInstanceWrapperContextCoherent) HealthCheck(x int) (int, error) {
	return siwc.underlying.HealthCheck(x)
}

func (siwc *substrateInstanceWrapperContextCoherent) NewCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperCoherent(siwc)
}

func (siwc *substrateInstanceWrapperContextCoherent) NewContextCoherent() SubstrateInstanceWrapperCommon {
	return NewSubstrateInstanceWrapperContextCoherent(siwc)
}

func (siwc *substrateInstanceWrapperContextCoherent) IsTimeoutError(err error) bool {
	return siwc.underlying.IsTimeoutError(err)
}

func (siwc *substrateInstanceWrapperContextCoherent) Init(phylum string, configs ...substratecommon.Config) error {
	return siwc.underlying.Init(phylum, configs...)
}

func (siwc *substrateInstanceWrapperContextCoherent) Call(method string, configs ...substratecommon.Config) (*substratecommon.Response, error) {
	configs2 := configs
	ctx, err := substratecommon.FlattenContext(configs...)
	if err != nil {
		ctx = context.Background()
	}
	dw, ok := ctx.Value(dependentKey).(*dependentWrapper)
	if ok && dw.dependent != "" {
		configs2 = append(configs2, substratecommon.WithDependentTxID(dw.dependent))
	}
	resp, err := siwc.underlying.Call(method, configs2...)
	if err != nil {
		return nil, err
	}
	if ok {
		dw.dependent = resp.TransactionID
	}
	return resp, nil
}

func (siwc *substrateInstanceWrapperContextCoherent) QueryInfo(configs ...substratecommon.Config) (uint64, error) {
	return siwc.underlying.QueryInfo(configs...)
}

func (siwc *substrateInstanceWrapperContextCoherent) QueryBlock(blockNumber uint64, configs ...substratecommon.Config) (*substratecommon.Block, error) {
	return siwc.underlying.QueryBlock(blockNumber, configs...)
}

func (siwc *substrateInstanceWrapperContextCoherent) GetLastTransactionID() string {
	return ""
}

func NewSubstrateInstanceWrapperContextCoherent(siwc SubstrateInstanceWrapperCommon) SubstrateInstanceWrapperCommon {
	return &substrateInstanceWrapperContextCoherent{underlying: siwc}
}
