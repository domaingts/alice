package task

import "github.com/xtls/xray-core/common"

// Close returns a func() that closes v.
func Close(v any) func() error {
	return func() error {
		return common.Close(v)
	}
}
