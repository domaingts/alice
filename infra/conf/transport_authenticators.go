package conf

import (
	"github.com/xtls/xray-core/transport/internet/headers/noop"
	"google.golang.org/protobuf/proto"
)

type NoOpConnectionAuthenticator struct{}

func (NoOpConnectionAuthenticator) Build() (proto.Message, error) {
	return new(noop.ConnectionConfig), nil
}
