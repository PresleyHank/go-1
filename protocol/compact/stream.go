package compact

import (
	"io"
	"fmt"
	"math"
	"github.com/thrift-iterator/go/protocol"
)

type Stream struct {
	writer io.Writer
	buf    []byte
	err    error
}

func NewStream(writer io.Writer, buf []byte) *Stream {
	return &Stream{
		writer: writer,
		buf:    buf,
	}
}

func (stream *Stream) Error() error {
	return stream.err
}

func (stream *Stream) ReportError(operation string, err string) {
	if stream.err == nil {
		stream.err = fmt.Errorf("%s: %s", operation, err)
	}
}

func (stream *Stream) Buffer() []byte {
	return stream.buf
}

func (stream *Stream) Reset(writer io.Writer) {
	stream.writer = writer
	stream.err = nil
	stream.buf = stream.buf[:0]
}

func (stream *Stream) Flush() {
	if stream.writer == nil {
		return
	}
	_, err := stream.writer.Write(stream.buf)
	if err != nil {
		stream.ReportError("Flush", err.Error())
		return
	}
	stream.buf = stream.buf[:0]
}

func (stream *Stream) WriteMessageHeader(header protocol.MessageHeader) {
	panic("not implemented")
}

func (stream *Stream) WriteMessage(message protocol.Message) {
	stream.WriteMessageHeader(message.MessageHeader)
	stream.WriteStruct(message.Arguments)
}

func (stream *Stream) WriteListHeader(elemType protocol.TType, length int) {
	stream.buf = append(stream.buf, byte(elemType),
		byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
}

func (stream *Stream) WriteList(val []interface{}) {
	if len(val) == 0 {
		stream.ReportError("WriteList", "input is empty slice, can not tell element type")
		return
	}
	elemType, elemWriter := stream.WriterOf(val[0])
	stream.WriteListHeader(elemType, len(val))
	for _, elem := range val {
		elemWriter(elem)
	}
}

func (stream *Stream) WriteStructField(fieldType protocol.TType, fieldId protocol.FieldId) {
	stream.buf = append(stream.buf, byte(fieldType), byte(fieldId>>8), byte(fieldId))
}

func (stream *Stream) WriteStructFieldStop() {
	stream.buf = append(stream.buf, byte(protocol.TypeStop))
}

func (stream *Stream) WriteStruct(val map[protocol.FieldId]interface{}) {
	for key, elem := range val {
		switch typedElem := elem.(type) {
		case bool:
			stream.WriteStructField(protocol.TypeBool, key)
			stream.WriteBool(typedElem)
		case int8:
			stream.WriteStructField(protocol.TypeI08, key)
			stream.WriteInt8(typedElem)
		case uint8:
			stream.WriteStructField(protocol.TypeI08, key)
			stream.WriteUInt8(typedElem)
		case int16:
			stream.WriteStructField(protocol.TypeI16, key)
			stream.WriteInt16(typedElem)
		case uint16:
			stream.WriteStructField(protocol.TypeI16, key)
			stream.WriteUInt16(typedElem)
		case int32:
			stream.WriteStructField(protocol.TypeI32, key)
			stream.WriteInt32(typedElem)
		case uint32:
			stream.WriteStructField(protocol.TypeI32, key)
			stream.WriteUInt32(typedElem)
		case int64:
			stream.WriteStructField(protocol.TypeI64, key)
			stream.WriteInt64(typedElem)
		case uint64:
			stream.WriteStructField(protocol.TypeI64, key)
			stream.WriteUInt64(typedElem)
		case float64:
			stream.WriteStructField(protocol.TypeDouble, key)
			stream.WriteFloat64(typedElem)
		case string:
			stream.WriteStructField(protocol.TypeString, key)
			stream.WriteString(typedElem)
		case []interface{}:
			stream.WriteStructField(protocol.TypeList, key)
			stream.WriteList(typedElem)
		case map[interface{}]interface{}:
			stream.WriteStructField(protocol.TypeMap, key)
			stream.WriteMap(typedElem)
		case map[protocol.FieldId]interface{}:
			stream.WriteStructField(protocol.TypeStruct, key)
			stream.WriteStruct(typedElem)
		default:
			panic("unsupported type")
		}
	}
	stream.WriteStructFieldStop()
	stream.Flush()
}

func (stream *Stream) WriteMapHeader(keyType protocol.TType, elemType protocol.TType, length int) {
	stream.buf = append(stream.buf, byte(keyType), byte(elemType),
		byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
}

func (stream *Stream) WriteMap(val map[interface{}]interface{}) {
	hasSample, sampleKey, sampleElem := takeSampleFromMap(val)
	if !hasSample {
		stream.ReportError("WriteMap", "input is empty map, can not tell element type")
		return
	}
	keyType, keyWriter := stream.WriterOf(sampleKey)
	elemType, elemWriter := stream.WriterOf(sampleElem)
	stream.WriteMapHeader(keyType, elemType, len(val))
	for key, elem := range val {
		keyWriter(key)
		elemWriter(elem)
	}
}

func takeSampleFromMap(val map[interface{}]interface{}) (bool, interface{}, interface{}) {
	for key, elem := range val {
		return true, key, elem
	}
	return false, nil, nil
}

func (stream *Stream) WriteBool(val bool) {
	if val {
		stream.WriteUInt8(1)
	} else {
		stream.WriteUInt8(0)
	}
}

func (stream *Stream) WriteInt8(val int8) {
	stream.WriteUInt8(uint8(val))
}

func (stream *Stream) WriteUInt8(val uint8) {
	stream.buf = append(stream.buf, byte(val))
}

func (stream *Stream) WriteInt16(val int16) {
	stream.WriteInt32(int32(val))
}

func (stream *Stream) WriteUInt16(val uint16) {
	stream.WriteInt32(int32(val))
}

func (stream *Stream) WriteInt32(val int32) {
	stream.writeVarInt32((val << 1) ^ (val >> 31))
}

func (stream *Stream) WriteUInt32(val uint32) {
	stream.WriteInt32(int32(val))
}

// Write an i32 as a varint. Results in 1-5 bytes on the wire.
func (stream *Stream) writeVarInt32(n int32) {
	for {
		if (n & ^0x7F) == 0 {
			stream.buf = append(stream.buf, byte(n))
			break
		} else {
			stream.buf = append(stream.buf, byte((n & 0x7F) | 0x80))
			u := uint64(n)
			n = int32(u >> 7)
		}
	}
}

func (stream *Stream) WriteInt64(val int64) {
	stream.writeVarInt64((val << 1) ^ (val >> 63))
}

// Write an i64 as a varint. Results in 1-10 bytes on the wire.
func (stream *Stream) writeVarInt64(n int64) {
	for {
		if (n & ^0x7F) == 0 {
			stream.buf = append(stream.buf, byte(n))
			break
		} else {
			stream.buf = append(stream.buf, byte((n & 0x7F) | 0x80))
			u := uint64(n)
			n = int64(u >> 7)
		}
	}
}

func (stream *Stream) WriteUInt64(val uint64) {
	stream.WriteInt64(int64(val))
}

func (stream *Stream) WriteInt(val int) {
	stream.WriteInt64(int64(val))
}

func (stream *Stream) WriteUInt(val uint) {
	stream.WriteUInt64(uint64(val))
}

func (stream *Stream) WriteFloat64(val float64) {
	bits := math.Float64bits(val)
	stream.buf = append(stream.buf,
		byte(bits),
		byte(bits >> 8),
		byte(bits >> 16),
		byte(bits >> 24),
		byte(bits >> 32),
		byte(bits >> 40),
		byte(bits >> 48),
		byte(bits >> 56),
	)
}

func (stream *Stream) WriteBinary(val []byte) {
	stream.writeVarInt32(int32(len(val)))
	stream.buf = append(stream.buf, val...)
}

func (stream *Stream) WriteString(val string) {
	stream.writeVarInt32(int32(len(val)))
	stream.buf = append(stream.buf, val...)
}

func (stream *Stream) WriterOf(sample interface{}) (protocol.TType, func(interface{})) {
	switch sample.(type) {
	case bool:
		return protocol.TypeBool, func(val interface{}) {
			stream.WriteBool(val.(bool))
		}
	case int8:
		return protocol.TypeI08, func(val interface{}) {
			stream.WriteInt8(val.(int8))
		}
	case uint8:
		return protocol.TypeI08, func(val interface{}) {
			stream.WriteUInt8(val.(uint8))
		}
	case int16:
		return protocol.TypeI16, func(val interface{}) {
			stream.WriteInt16(val.(int16))
		}
	case uint16:
		return protocol.TypeI16, func(val interface{}) {
			stream.WriteUInt16(val.(uint16))
		}
	case int32:
		return protocol.TypeI32, func(val interface{}) {
			stream.WriteInt32(val.(int32))
		}
	case uint32:
		return protocol.TypeI32, func(val interface{}) {
			stream.WriteUInt32(val.(uint32))
		}
	case int64:
		return protocol.TypeI64, func(val interface{}) {
			stream.WriteInt64(val.(int64))
		}
	case uint64:
		return protocol.TypeI64, func(val interface{}) {
			stream.WriteUInt64(val.(uint64))
		}
	case float64:
		return protocol.TypeDouble, func(val interface{}) {
			stream.WriteFloat64(val.(float64))
		}
	case string:
		return protocol.TypeString, func(val interface{}) {
			stream.WriteString(val.(string))
		}
	case []interface{}:
		return protocol.TypeList, func(val interface{}) {
			stream.WriteList(val.([]interface{}))
		}
	case map[interface{}]interface{}:
		return protocol.TypeMap, func(val interface{}) {
			stream.WriteMap(val.(map[interface{}]interface{}))
		}
	case map[protocol.FieldId]interface{}:
		return protocol.TypeStruct, func(val interface{}) {
			stream.WriteStruct(val.(map[protocol.FieldId]interface{}))
		}
	default:
		panic("unsupported type")
	}
}

