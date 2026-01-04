package vless

import (
	"crypto/tls"
	"reflect"
	"sync"

	utls "github.com/refraction-networking/utls"
	"github.com/xtls/reality"
	"github.com/xtls/xray-core/proxy/vless/encryption"
)

var (
	EncryptionOffsets = sync.OnceValues(func() (uintptr, uintptr) {
		t := reflect.TypeFor[*encryption.CommonConn]().Elem()
		i, _ := t.FieldByName("input")
		r, _ := t.FieldByName("rawInput")
		return i.Offset, r.Offset
	})
	TLSOffsets = sync.OnceValues(func() (uintptr, uintptr) {
		t := reflect.TypeFor[*tls.Conn]().Elem()
		i, _ := t.FieldByName("input")
		r, _ := t.FieldByName("rawInput")
		return i.Offset, r.Offset
	})
	UtlsOffsets = sync.OnceValues(func() (uintptr, uintptr) {
		t := reflect.TypeFor[*utls.Conn]().Elem()
		i, _ := t.FieldByName("input")
		r, _ := t.FieldByName("rawInput")
		return i.Offset, r.Offset
	})
	RealityOffsets = sync.OnceValues(func() (uintptr, uintptr) {
		t := reflect.TypeFor[*reality.Conn]().Elem()
		i, _ := t.FieldByName("input")
		r, _ := t.FieldByName("rawInput")
		return i.Offset, r.Offset
	})
)
