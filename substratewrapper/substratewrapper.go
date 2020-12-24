package substratewrapper

import (
	"github.com/luthersystems/substratecommon"
)

type SubstrateWrapper struct {
	substrate substratecommon.Substrate
}

type SubstrateInstanceWrapperCommon struct {
	substrate substratecommon.Substrate
	tag       string
}

type SubstrateInstanceWrapperRPC struct {
	SubstrateInstanceWrapperCommon
}

type SubstrateInstanceWrapperMock struct {
	SubstrateInstanceWrapperCommon
}

func NewSubstrateWrapper(substrate substratecommon.Substrate) *SubstrateWrapper {
	return &SubstrateWrapper{substrate: substrate}
}

func (sw *SubstrateWrapper) NewRPC() (*SubstrateInstanceWrapperRPC, error) {
	tag, err := sw.substrate.NewRPC()
	if err != nil {
		return nil, err
	}
	return &SubstrateInstanceWrapperRPC{SubstrateInstanceWrapperCommon{substrate: sw.substrate, tag: tag}}, nil
}

func (siw *SubstrateInstanceWrapperRPC) Upcast() *SubstrateInstanceWrapperCommon {
	return &SubstrateInstanceWrapperCommon{substrate: siw.substrate, tag: siw.tag}
}

func (siw *SubstrateInstanceWrapperRPC) CloseRPC() error {
	return siw.substrate.CloseRPC(siw.tag)
}

func (sw *SubstrateWrapper) NewMockFrom(name string, phylumVersion string, blob []byte) (*SubstrateInstanceWrapperMock, error) {
	tag, err := sw.substrate.NewMockFrom(name, phylumVersion, blob)
	if err != nil {
		return nil, err
	}
	return &SubstrateInstanceWrapperMock{SubstrateInstanceWrapperCommon{substrate: sw.substrate, tag: tag}}, nil
}

func (siw *SubstrateInstanceWrapperMock) Upcast() *SubstrateInstanceWrapperCommon {
	return &SubstrateInstanceWrapperCommon{substrate: siw.substrate, tag: siw.tag}
}

func (siw *SubstrateInstanceWrapperMock) SetCreatorWithAttributesMock(creator string, attrs map[string]string) error {
	return siw.substrate.SetCreatorWithAttributesMock(siw.tag, creator, attrs)
}

func (siw *SubstrateInstanceWrapperMock) SnapshotMock() ([]byte, error) {
	return siw.substrate.SnapshotMock(siw.tag)
}

func (siw *SubstrateInstanceWrapperMock) CloseMock() error {
	return siw.substrate.CloseMock(siw.tag)
}

func (siw *SubstrateInstanceWrapperCommon) Init(phylum string, configs ...substratecommon.Config) error {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return err
	}
	return siw.substrate.Init(siw.tag, phylum, fo)
}

func (siw *SubstrateInstanceWrapperCommon) Call(method string, configs ...substratecommon.Config) (*substratecommon.Response, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return nil, err
	}
	return siw.substrate.Call(siw.tag, method, fo)
}

func (siw *SubstrateInstanceWrapperCommon) QueryInfo(configs ...substratecommon.Config) (uint64, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return 0, err
	}
	return siw.substrate.QueryInfo(siw.tag, fo)
}

func (siw *SubstrateInstanceWrapperCommon) QueryBlock(blockNumber uint64, configs ...substratecommon.Config) (*substratecommon.Block, error) {
	fo, err := substratecommon.FlattenOptions(configs...)
	if err != nil {
		return nil, err
	}
	return siw.substrate.QueryBlock(siw.tag, blockNumber, fo)
}

func (siw *SubstrateInstanceWrapperCommon) IsTimeoutError(err error) bool {
	return siw.substrate.IsTimeoutError(err)
}
