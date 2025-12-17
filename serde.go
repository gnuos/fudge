package fudge

import (
	"bytes"
	"encoding/binary"
	"log"

	"github.com/fxamacker/cbor/v2"
)

var enc cbor.EncMode

var opt = cbor.CanonicalEncOptions()

func getEncMode() (cbor.EncMode, error) {
	if enc != nil {
		return enc, nil
	}

	return opt.EncMode()
}

func marshal(v any) ([]byte, error) {
	enc, err := getEncMode()
	if err != nil {
		log.Fatal(err)
	}

	return enc.Marshal(v)
}

// KeyToBinary return key in bytes
func KeyToBinary(v any) ([]byte, error) {
	var err error

	switch v := v.(type) {
	case []byte:
		return v, nil
	case bool, float32, float64, complex64, complex128, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.BigEndian, v)
		return buf.Bytes(), err
	case int:
		val := uint64(v)
		p := make([]byte, 8)
		p[0] = byte(val >> 56)
		p[1] = byte(val >> 48)
		p[2] = byte(val >> 40)
		p[3] = byte(val >> 32)
		p[4] = byte(val >> 24)
		p[5] = byte(val >> 16)
		p[6] = byte(val >> 8)
		p[7] = byte(val)
		return p, err
	case string:
		return []byte(v), nil
	default:
		return marshal(v)
	}
}

// ValToBinary return value in bytes
func ValToBinary(v any) ([]byte, error) {
	switch v := v.(type) {
	case []byte:
		return v, nil
	default:
		return marshal(v)
	}
}
