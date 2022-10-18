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

import "math"

func (p *LazyBuffer) AppendString(tag ProtoWireNumber, data string) {
	p.AppendBytes(tag, []byte(data))
}

func (p *LazyBuffer) AppendBytes(tag ProtoWireNumber, data []byte) {
	p.appendTag(tag, BytesType)
	p.appendVarInt(uint64(len(data)))
	p.buf = append(p.buf, data...)
}

func (p *LazyBuffer) AppendBool(tag ProtoWireNumber, data bool) {
	p.appendTag(tag, VarIntType)
	p.AppendBoolWithoutTag(data)
}

func (p *LazyBuffer) AppendBoolWithoutTag(data bool) {
	if data {
		p.appendVarInt(1)
	} else {
		p.appendVarInt(0)
	}
}

func (p *LazyBuffer) AppendInt32(tag ProtoWireNumber, data int32) {
	p.appendTag(tag, VarIntType)
	p.AppendInt32WithoutTag(data)
}

func (p *LazyBuffer) AppendInt32WithoutTag(data int32) {
	p.appendVarInt(uint64(data))
}

func (p *LazyBuffer) AppendInt64(tag ProtoWireNumber, data int64) {
	p.appendTag(tag, VarIntType)
	p.AppendInt64WithoutTag(data)
}

func (p *LazyBuffer) AppendInt64WithoutTag(data int64) {
	p.appendVarInt(uint64(data))
}

func (p *LazyBuffer) AppendUint32(tag ProtoWireNumber, data uint32) {
	p.appendTag(tag, VarIntType)
	p.AppendUint32WithoutTag(data)
}

func (p *LazyBuffer) AppendUint32WithoutTag(data uint32) {
	p.appendVarInt(uint64(data))
}

func (p *LazyBuffer) AppendUint64(tag ProtoWireNumber, data uint64) {
	p.appendTag(tag, VarIntType)
	p.AppendUint64WithoutTag(data)
}

func (p *LazyBuffer) AppendUint64WithoutTag(data uint64) {
	p.appendVarInt(data)
}

func (p *LazyBuffer) AppendSInt32(tag ProtoWireNumber, data int32) {
	p.appendTag(tag, VarIntType)
	p.AppendSInt32WithoutTag(data)
}

func (p *LazyBuffer) AppendSInt32WithoutTag(data int32) {
	p.appendSignedVarInt(int64(data))
}

func (p *LazyBuffer) AppendSInt64(tag ProtoWireNumber, data int64) {
	p.appendTag(tag, VarIntType)
	p.AppendSInt64WithoutTag(data)
}

func (p *LazyBuffer) AppendSInt64WithoutTag(data int64) {
	p.appendSignedVarInt(data)
}

func (p *LazyBuffer) AppendFixed32(tag ProtoWireNumber, data uint32) {
	p.appendTag(tag, Fixed32Type)
	p.AppendFixed32WithoutTag(data)
}

func (p *LazyBuffer) AppendFixed32WithoutTag(data uint32) {
	p.appendFixed32(data)
}

func (p *LazyBuffer) AppendFixed64(tag ProtoWireNumber, data uint64) {
	p.appendTag(tag, Fixed64Type)
	p.AppendFixed64WithoutTag(data)
}

func (p *LazyBuffer) AppendFixed64WithoutTag(data uint64) {
	p.appendFixed64(data)
}

func (p *LazyBuffer) AppendSFixed32(tag ProtoWireNumber, data int32) {
	p.appendTag(tag, Fixed32Type)
	p.AppendSFixed32WithoutTag(data)
}

func (p *LazyBuffer) AppendSFixed32WithoutTag(data int32) {
	p.appendFixed32(uint32(data))
}

func (p *LazyBuffer) AppendSFixed64(tag ProtoWireNumber, data int64) {
	p.appendTag(tag, Fixed64Type)
	p.AppendSFixed64WithoutTag(data)
}

func (p *LazyBuffer) AppendSFixed64WithoutTag(data int64) {
	p.appendFixed64(uint64(data))
}

func (p *LazyBuffer) AppendFloat32(tag ProtoWireNumber, data float32) {
	p.appendTag(tag, Fixed32Type)
	p.AppendFloat32WithoutTag(data)
}

func (p *LazyBuffer) AppendFloat32WithoutTag(data float32) {
	p.appendFixed32(math.Float32bits(data))
}

func (p *LazyBuffer) AppendFloat64(tag ProtoWireNumber, data float64) {
	p.appendTag(tag, Fixed64Type)
	p.AppendFloat64WithoutTag(data)
}

func (p *LazyBuffer) AppendFloat64WithoutTag(data float64) {
	p.appendFixed64(math.Float64bits(data))
}

func (p *LazyBuffer) appendTag(tag ProtoWireNumber, protoType ProtoWireType) {
	tagVarInt := uint64(tag)<<3 | uint64(protoType&7)
	p.appendVarInt(tagVarInt)
}

func (p *LazyBuffer) appendFixed32(v uint32) {
	p.buf = append(p.buf,
		byte(v>>0),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
	)
}

func (p *LazyBuffer) appendFixed64(v uint64) {
	p.buf = append(p.buf,
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

func (p *LazyBuffer) appendSignedVarInt(v int64) {
	value := uint64(v<<1) ^ uint64(v>>63)
	p.appendVarInt(value)
}

func (p *LazyBuffer) appendVarInt(v uint64) {
	switch {
	case v < 1<<7:
		p.buf = append(p.buf, byte(v))
	case v < 1<<14:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte(v>>7))
	case v < 1<<21:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte(v>>14))
	case v < 1<<28:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte(v>>21))
	case v < 1<<35:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte(v>>28))
	case v < 1<<42:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte(v>>35))
	case v < 1<<49:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte(v>>42))
	case v < 1<<56:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte((v>>42)&0x7f|0x80),
			byte(v>>49))
	case v < 1<<63:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte((v>>42)&0x7f|0x80),
			byte((v>>49)&0x7f|0x80),
			byte(v>>56))
	default:
		p.buf = append(p.buf,
			byte((v>>0)&0x7f|0x80),
			byte((v>>7)&0x7f|0x80),
			byte((v>>14)&0x7f|0x80),
			byte((v>>21)&0x7f|0x80),
			byte((v>>28)&0x7f|0x80),
			byte((v>>35)&0x7f|0x80),
			byte((v>>42)&0x7f|0x80),
			byte((v>>49)&0x7f|0x80),
			byte((v>>56)&0x7f|0x80),
			1)
	}
}
