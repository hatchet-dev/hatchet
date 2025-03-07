package v1

type GRPCClientFactory struct {
	defaultFs []GRPCClientOpt
}

func NewGRPCClientFactory(fs ...GRPCClientOpt) *GRPCClientFactory {
	return &GRPCClientFactory{
		defaultFs: fs,
	}
}

func (f *GRPCClientFactory) NewGRPCClient(token string) (*GRPCClient, error) {
	return NewGRPCClient(append(f.defaultFs, WithToken(token))...)
}
