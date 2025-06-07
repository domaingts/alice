package internet_test

import (
	"testing"

	"github.com/xtls/xray-core/common"
	. "github.com/xtls/xray-core/transport/internet"
	"github.com/xtls/xray-core/transport/internet/headers/noop"
)

func TestAllHeadersLoadable(t *testing.T) {
	testCases := []struct {
		Input any
		Size  int32
	}{
		{
			Input: new(noop.Config),
			Size:  0,
		},
	}

	for _, testCase := range testCases {
		header, err := CreatePacketHeader(testCase.Input)
		common.Must(err)
		if header.Size() != testCase.Size {
			t.Error("expected size ", testCase.Size, " but got ", header.Size())
		}
	}
}
