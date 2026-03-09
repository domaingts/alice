package vless

import (
	"crypto/tls"
	"reflect"

	utls "github.com/refraction-networking/utls"
	"github.com/xtls/reality"
	"github.com/xtls/xray-core/proxy/vless/encryption"
)

var (
	EncryptOffset = NewConnOffset[encryption.CommonConn]()
	TLSOffset     = NewConnOffset[tls.Conn]()
	UtlsOffset    = NewConnOffset[utls.Conn]()
	RealityOffset = NewConnOffset[reality.Conn]()
)

type Offset interface {
	Input() uintptr
	RawInput() uintptr
}

type ConnOffset struct {
	input    uintptr
	rawInput uintptr
}

func NewConnOffset[T any]() Offset {
	t := reflect.TypeFor[T]()
	i, _ := t.FieldByName("input")
	r, _ := t.FieldByName("rawInput")
	return &ConnOffset{i.Offset, r.Offset}
}

func (c *ConnOffset) Input() uintptr {
	return c.input
}

func (c *ConnOffset) RawInput() uintptr {
	return c.rawInput
}
