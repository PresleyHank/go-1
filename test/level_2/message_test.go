package test

import (
	"testing"
	"github.com/stretchr/testify/require"
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/thrift-iterator/go"
	"github.com/thrift-iterator/go/protocol"
)

func Test_decode_message_by_iterator(t *testing.T) {
	should := require.New(t)
	buf := thrift.NewTMemoryBuffer()
	proto := thrift.NewTBinaryProtocol(buf, true, true)
	proto.WriteMessageBegin("hello", thrift.CALL, 17)
	proto.WriteStructBegin("args")
	proto.WriteFieldBegin("field1", thrift.I64, 1)
	proto.WriteI64(1)
	proto.WriteFieldBegin("field2", thrift.I64, 2)
	proto.WriteI64(2)
	proto.WriteFieldEnd()
	proto.WriteFieldStop()
	proto.WriteStructEnd()
	proto.WriteMessageEnd()
	iter := thrifter.NewBufferedIterator(buf.Bytes())
	msg := iter.ReadMessageHeader()
	should.Equal(protocol.VERSION_1, msg.Version)
	should.Equal("hello", msg.MessageName)
	should.Equal(protocol.CALL, msg.MessageType)
	should.Equal(protocol.SeqId(17), msg.SeqId)
}

func Test_decode_message_as_object(t *testing.T) {
	should := require.New(t)
	buf := thrift.NewTMemoryBuffer()
	proto := thrift.NewTBinaryProtocol(buf, true, true)
	proto.WriteMessageBegin("hello", thrift.CALL, 17)
	proto.WriteStructBegin("args")
	proto.WriteFieldBegin("field1", thrift.I64, 1)
	proto.WriteI64(1)
	proto.WriteFieldBegin("field2", thrift.I64, 2)
	proto.WriteI64(2)
	proto.WriteFieldEnd()
	proto.WriteFieldStop()
	proto.WriteStructEnd()
	proto.WriteMessageEnd()
	iter := thrifter.NewBufferedIterator(buf.Bytes())
	msg := iter.ReadMessage()
	should.Equal(protocol.VERSION_1, msg.Version)
	should.Equal("hello", msg.MessageName)
	should.Equal(protocol.CALL, msg.MessageType)
	should.Equal(protocol.SeqId(17), msg.SeqId)
	should.Equal(int64(1), msg.Arguments[protocol.FieldId(1)])
	should.Equal(int64(2), msg.Arguments[protocol.FieldId(2)])
}

func Test_encode_message_as_object(t *testing.T) {
	should := require.New(t)
	stream := thrifter.NewBufferedStream(nil)
	stream.WriteMessage(protocol.Message{
		MessageHeader: protocol.MessageHeader{
			Version: protocol.VERSION_1,
			MessageType: protocol.CALL,
			MessageName: "hello",
			SeqId: protocol.SeqId(17),
		},
		Arguments: map[protocol.FieldId]interface{} {
			protocol.FieldId(1): int64(1),
			protocol.FieldId(2): int64(2),
		},
	})
	iter := thrifter.NewBufferedIterator(stream.Buffer())
	msg := iter.ReadMessage()
	should.Equal(protocol.VERSION_1, msg.Version)
	should.Equal("hello", msg.MessageName)
	should.Equal(protocol.CALL, msg.MessageType)
	should.Equal(protocol.SeqId(17), msg.SeqId)
	should.Equal(int64(1), msg.Arguments[protocol.FieldId(1)])
	should.Equal(int64(2), msg.Arguments[protocol.FieldId(2)])
}