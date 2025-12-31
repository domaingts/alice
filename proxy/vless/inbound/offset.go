package inbound

import (
	"crypto/tls"
	"reflect"
	"sync"

	"github.com/xtls/reality"
	"github.com/xtls/xray-core/proxy/vless/encryption"
)

var (
	encryptionOffsets = sync.OnceValues(func() (uintptr, uintptr) {
		t := reflect.TypeFor[*encryption.CommonConn]().Elem()
		i, _ := t.FieldByName("input")
		r, _ := t.FieldByName("rawInput")
		return i.Offset, r.Offset
	})
	tlsOffsets = sync.OnceValues(func() (uintptr, uintptr) {
		t := reflect.TypeFor[*tls.Conn]().Elem()
		i, _ := t.FieldByName("input")
		r, _ := t.FieldByName("rawInput")
		return i.Offset, r.Offset
	})
	realityOffsets = sync.OnceValues(func() (uintptr, uintptr) {
		t := reflect.TypeFor[*reality.Conn]().Elem()
		i, _ := t.FieldByName("input")
		r, _ := t.FieldByName("rawInput")
		return i.Offset, r.Offset
	})
)
