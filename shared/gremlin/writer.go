/*
               .'\   /`.
             .'.-.`-'.-.`.
        ..._:   .-. .-.   :_...
      .'    '-.(o ) (o ).-'    `.
     :  _    _ _`~(_)~`_ _    _  :
    :  /:   ' .-=_   _=-. `   ;\  :
    :   :|-.._  '     `  _..-|:   :
     :   `:| |`:-:-.-:-:'| |:'   :
      `.   `.| | | | | | |.'   .'
        `.   `-:_| | |_:-'   .'
          `-._   ````    _.-'
              ``-------''

Created by ab, 06.10.2022
*/

package gremlin

import (
	"math"
)

type Writer struct {
	buf []byte // contents are the bytes buf[off : len(buf)]
}

func NewWriter(size int) *Writer {
	return &Writer{
		buf: make([]byte, 0, size),
	}
}

func (p *Writer) AppendString(tag ProtoWireNumber, data string) {
	bytesLen := len(data)
	p.appendTag(tag, BytesType)
	p.appendVarInt(uint64(bytesLen))

	m := len(p.buf)
	p.buf = p.buf[:m+bytesLen]
	copy(p.buf[m:], data)
}

func SizeString(data string) int {
	return len(data)
}

func (p *Writer) AppendBytesTag(tag ProtoWireNumber, len int) {
	p.appendTag(tag, BytesType)
	p.appendVarInt(uint64(len))
}

func (p *Writer) AppendBytes(tag ProtoWireNumber, data []byte) {
	p.appendTag(tag, BytesType)
	p.appendVarInt(uint64(len(data)))
	p.writeBytes(data)
}

func SizeBytes(data []byte) int {
	return len(data)
}

func (p *Writer) AppendBool(tag ProtoWireNumber, data bool) {
	p.appendTag(tag, VarIntType)
	p.AppendBoolWithoutTag(data)
}

func SizeBool(data bool) int {
	return 1
}

func (p *Writer) AppendBoolWithoutTag(data bool) {
	if data {
		p.appendVarInt(1)
	} else {
		p.appendVarInt(0)
	}
}

func (p *Writer) AppendInt32(tag ProtoWireNumber, data int32) {
	p.appendTag(tag, VarIntType)
	p.AppendInt32WithoutTag(data)
}

func SizeInt32(data int32) int {
	return SizeVarInt(uint64(data))
}

func (p *Writer) AppendInt32WithoutTag(data int32) {
	p.appendVarInt(uint64(data))
}

func (p *Writer) AppendInt64(tag ProtoWireNumber, data int64) {
	p.appendTag(tag, VarIntType)
	p.AppendInt64WithoutTag(data)
}

func SizeInt64(data int64) int {
	return SizeVarInt(uint64(data))
}

func (p *Writer) AppendInt64WithoutTag(data int64) {
	p.appendVarInt(uint64(data))
}

func (p *Writer) AppendUint32(tag ProtoWireNumber, data uint32) {
	p.appendTag(tag, VarIntType)
	p.AppendUint32WithoutTag(data)
}

func SizeUint32(data uint32) int {
	return SizeVarInt(uint64(data))
}

func (p *Writer) AppendUint32WithoutTag(data uint32) {
	p.appendVarInt(uint64(data))
}

func (p *Writer) AppendUint64(tag ProtoWireNumber, data uint64) {
	p.appendTag(tag, VarIntType)
	p.AppendUint64WithoutTag(data)
}

func SizeUint64(data uint64) int {
	return SizeVarInt(data)
}

func (p *Writer) AppendUint64WithoutTag(data uint64) {
	p.appendVarInt(data)
}

func (p *Writer) AppendSInt32(tag ProtoWireNumber, data int32) {
	p.appendTag(tag, VarIntType)
	p.AppendSInt32WithoutTag(data)
}

func SizeSInt32(data int32) int {
	return sizeSignedVarInt(int64(data))
}

func (p *Writer) AppendSInt32WithoutTag(data int32) {
	p.appendSignedVarInt(int64(data))
}

func (p *Writer) AppendSInt64(tag ProtoWireNumber, data int64) {
	p.appendTag(tag, VarIntType)
	p.AppendSInt64WithoutTag(data)
}

func SizeSInt64(data int64) int {
	return sizeSignedVarInt(data)
}

func (p *Writer) AppendSInt64WithoutTag(data int64) {
	p.appendSignedVarInt(data)
}

func (p *Writer) AppendFixed32(tag ProtoWireNumber, data uint32) {
	p.appendTag(tag, Fixed32Type)
	p.AppendFixed32WithoutTag(data)
}

func SizeFixed32(data uint32) int {
	return sizeFixed32()
}

func (p *Writer) AppendFixed32WithoutTag(data uint32) {
	p.appendFixed32(data)
}

func (p *Writer) AppendFixed64(tag ProtoWireNumber, data uint64) {
	p.appendTag(tag, Fixed64Type)
	p.AppendFixed64WithoutTag(data)
}

func SizeFixed64(data uint64) int {
	return sizeFixed64()
}

func (p *Writer) AppendFixed64WithoutTag(data uint64) {
	p.appendFixed64(data)
}

func (p *Writer) AppendSFixed32(tag ProtoWireNumber, data int32) {
	p.appendTag(tag, Fixed32Type)
	p.AppendSFixed32WithoutTag(data)
}

func SizeSFixed32(data int32) int {
	return sizeFixed32()
}

func (p *Writer) AppendSFixed32WithoutTag(data int32) {
	p.appendFixed32(uint32(data))
}

func (p *Writer) AppendSFixed64(tag ProtoWireNumber, data int64) {
	p.appendTag(tag, Fixed64Type)
	p.AppendSFixed64WithoutTag(data)
}

func SizeSFixed64(data int64) int {
	return sizeFixed64()
}

func (p *Writer) AppendSFixed64WithoutTag(data int64) {
	p.appendFixed64(uint64(data))
}

func (p *Writer) AppendFloat32(tag ProtoWireNumber, data float32) {
	p.appendTag(tag, Fixed32Type)
	p.AppendFloat32WithoutTag(data)
}

func SizeFloat32(data float32) int {
	return sizeFixed32()
}

func (p *Writer) AppendFloat32WithoutTag(data float32) {
	p.appendFixed32(math.Float32bits(data))
}

func (p *Writer) AppendFloat64(tag ProtoWireNumber, data float64) {
	p.appendTag(tag, Fixed64Type)
	p.AppendFloat64WithoutTag(data)
}

func SizeFloat64(data float64) int {
	return sizeFixed64()
}

func (p *Writer) AppendFloat64WithoutTag(data float64) {
	p.appendFixed64(math.Float64bits(data))
}

func (p *Writer) appendTag(tag ProtoWireNumber, protoType ProtoWireType) {
	tagVarInt := uint64(tag)<<3 | uint64(protoType&7)
	p.appendVarInt(tagVarInt)
}

func SizeTag(tag ProtoWireNumber) int {
	tagVarInt := uint64(tag)<<3 | uint64(0&7) //wire type not affect size
	return SizeVarInt(tagVarInt)
}

func (p *Writer) appendFixed32(v uint32) {
	p.writeByte(
		byte(v>>0),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
	)
}

func sizeFixed32() int {
	return 4
}

func (p *Writer) appendFixed64(v uint64) {
	p.writeByte(
		byte(v>>0),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
		byte(v>>32),
		byte(v>>40),
		byte(v>>48),
		byte(v>>56),
	)
}

func sizeFixed64() int {
	return 8
}

func (p *Writer) appendSignedVarInt(v int64) {
	value := uint64(v<<1) ^ uint64(v>>63)
	p.appendVarInt(value)
}

func sizeSignedVarInt(v int64) int {
	value := uint64(v<<1) ^ uint64(v>>63)
	return SizeVarInt(value)
}

func (p *Writer) appendVarInt(v uint64) {
	switch {
	case v < 1<<7:
		p.writeByte(byte(v))
	case v < 1<<14:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte(v>>7),
		)
	case v < 1<<21:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte(v>>14),
		)
	case v < 1<<28:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte(v>>21),
		)
	case v < 1<<35:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte(v>>28),
		)
	case v < 1<<42:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte(v>>35),
		)
	case v < 1<<49:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte(v>>42),
		)
	case v < 1<<56:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte((v>>42)&0x7f|0x80),
			byte(v>>49),
		)
	case v < 1<<63:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte((v>>42)&0x7f|0x80),
			byte((v>>49)&0x7f|0x80),
			byte(v>>56),
		)
	default:
		p.writeByte(
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte((v>>42)&0x7f|0x80),
			byte((v>>49)&0x7f|0x80),
			byte((v>>56)&0x7f|0x80),
			1,
		)
	}
}

func SizeVarInt(v uint64) int {
	switch {
	case v < 1<<7:
		return 1
	case v < 1<<14:
		return 2
	case v < 1<<21:
		return 3
	case v < 1<<28:
		return 4
	case v < 1<<35:
		return 5
	case v < 1<<42:
		return 6
	case v < 1<<49:
		return 7
	case v < 1<<56:
		return 8
	case v < 1<<63:
		return 9
	default:
		return 10
	}
}

func (p *Writer) writeBytes(data []byte) {
	m := len(p.buf)
	p.buf = p.buf[:m+len(data)]
	copy(p.buf[m:], data)
}

func (p *Writer) writeByte(data ...byte) {
	p.writeBytes(data)
}

func (p *Writer) Bytes() []byte {
	return p.buf[:len(p.buf)]
}
