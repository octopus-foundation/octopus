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
	"fmt"
	"math"
)

func (p *LazyBuffer) ReadTagAt(offset int) (ProtoWireNumber, ProtoWireType, int, error) {
	tagData, tagSize := p.readVarIntAt(offset)
	if tagSize < 0 {
		return 0, 0, 0, fmt.Errorf("invalid tag")
	}

	if tagData>>3 > uint64(math.MaxInt32) {
		return 0, 0, 0, fmt.Errorf("tag number out of range")
	}
	return ProtoWireNumber(tagData >> 3), ProtoWireType(tagData & 7), tagSize, nil
}

func (p *LazyBuffer) GetAllFields(handler func(ProtoWireNumber, ProtoWireType, int) error) error {
	offset := 0
	for p.hasNext(offset, 0) {
		tag, protoType, tagSize, err := p.ReadTagAt(offset)
		if err != nil {
			return err
		}

		offset += tagSize
		if err = handler(tag, protoType, offset); err != nil {
			return err
		}

		offset, err = p.SkipData(offset, protoType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *LazyBuffer) SkipData(offset int, protoType ProtoWireType) (int, error) {
	switch protoType {
	case VarIntType:
		varIntSize, err := p.getVarIntSize(offset)
		if err != nil {
			return 0, err
		}
		return offset + varIntSize, nil
	case Fixed32Type:
		return offset + 4, nil
	case Fixed64Type:
		return offset + 8, nil
	case BytesType:
		size, sizeSize := p.readVarIntAt(offset)
		if sizeSize < 0 {
			return 0, fmt.Errorf("invalid varint")
		}
		return offset + sizeSize + int(size), nil
	case StartGroupType: // deprecated, but need to skip
		for {
			_, cType, tagSize, err := p.ReadTagAt(offset)
			if err != nil {
				return 0, err
			}
			offset += tagSize
			if cType == EndGroupType {
				return offset, nil
			}
			offset, err = p.SkipData(offset, cType)
			if err != nil {
				return 0, err
			}
		}
	}

	return 0, fmt.Errorf("invalid wire type while skipping data: %v", protoType)
}

func (p *LazyBuffer) hasNext(offset int, size int) bool {
	return offset+size < len(p.buf)
}

func (p *LazyBuffer) getVarIntSize(offset int) (int, error) {
	for i := 0; i < 10; i++ {
		if !p.hasNext(offset, i) {
			return 0, fmt.Errorf("invalid varint, not enough bytes")
		}
		if p.buf[offset+i] < 0x80 {
			return i + 1, nil
		}
	}
	return 10, nil
}

func (p *LazyBuffer) readFixed32At(offset int) uint64 {
	return uint64(p.buf[offset]) | uint64(p.buf[offset+1])<<8 | uint64(p.buf[offset+2])<<16 | uint64(p.buf[offset+3])<<24
}

func (p *LazyBuffer) readFixed64At(offset int) uint64 {
	return uint64(p.buf[offset]) | uint64(p.buf[offset+1])<<8 | uint64(p.buf[offset+2])<<16 | uint64(p.buf[offset+3])<<24 |
		uint64(p.buf[offset+4])<<32 | uint64(p.buf[offset+5])<<40 | uint64(p.buf[offset+6])<<48 | uint64(p.buf[offset+7])<<56
}

func (p *LazyBuffer) ReadBytes(offset int) []byte {
	size, sizeSize := p.readVarIntAt(offset)
	return p.buf[offset+sizeSize : offset+sizeSize+int(size)]
}

func (p *LazyBuffer) SizedReadBytes(offset int) ([]byte, int) {
	size, sizeSize := p.readVarIntAt(offset)
	return p.buf[offset+sizeSize : offset+sizeSize+int(size)], sizeSize + int(size)
}

func (p *LazyBuffer) ReadString(offset int) string {
	return bytesToString(p.ReadBytes(offset))
}

func (p *LazyBuffer) SizedReadString(offset int) (string, int) {
	v, size := p.SizedReadBytes(offset)
	return bytesToString(v), size
}

func (p *LazyBuffer) ReadVarInt(offset int) uint64 {
	v, _ := p.readVarIntAt(offset)
	return v
}

func (p *LazyBuffer) SizedReadVarInt(offset int) (uint64, int) {
	return p.readVarIntAt(offset)
}

func (p *LazyBuffer) ReadUint64(offset int) uint64 {
	return p.ReadVarInt(offset)
}

func (p *LazyBuffer) SizedReadUint64(offset int) (uint64, int) {
	return p.SizedReadVarInt(offset)
}

func (p *LazyBuffer) ReadUint32(offset int) uint32 {
	return uint32(p.ReadVarInt(offset))
}

func (p *LazyBuffer) SizedReadUint32(offset int) (uint32, int) {
	res, size := p.SizedReadVarInt(offset)
	return uint32(res), size
}

func (p *LazyBuffer) ReadInt64(offset int) int64 {
	return int64(p.ReadVarInt(offset))
}

func (p *LazyBuffer) SizedReadInt64(offset int) (int64, int) {
	res, size := p.readVarIntAt(offset)
	return int64(res), size
}

func (p *LazyBuffer) ReadSInt64(offset int) int64 {
	res, _ := p.readSignedVarIntAt(offset)
	return res
}

func (p *LazyBuffer) SizedReadSInt64(offset int) (int64, int) {
	return p.readSignedVarIntAt(offset)
}

func (p *LazyBuffer) ReadInt32(offset int) int32 {
	return int32(p.ReadVarInt(offset))
}

func (p *LazyBuffer) SizedReadInt32(offset int) (int32, int) {
	res, size := p.readVarIntAt(offset)
	return int32(res), size
}

func (p *LazyBuffer) ReadSInt32(offset int) int32 {
	res, _ := p.readSignedVarIntAt(offset)
	return int32(res)
}

func (p *LazyBuffer) SizedReadSInt32(offset int) (int32, int) {
	res, n := p.readSignedVarIntAt(offset)
	return int32(res), n
}

func (p *LazyBuffer) ReadBool(offset int) bool {
	return p.ReadVarInt(offset) != 0
}

func (p *LazyBuffer) SizedReadBool(offset int) (bool, int) {
	res, size := p.SizedReadVarInt(offset)
	return res != 0, size
}

func (p *LazyBuffer) ReadFloat32(offset int) float32 {
	v := p.readFixed32At(offset)
	return math.Float32frombits(uint32(v))
}

func (p *LazyBuffer) SizedReadFloat32(offset int) (float32, int) {
	v := p.readFixed32At(offset)
	return math.Float32frombits(uint32(v)), 4
}

func (p *LazyBuffer) ReadFloat64(offset int) float64 {
	v := p.readFixed64At(offset)
	return math.Float64frombits(v)
}

func (p *LazyBuffer) SizedReadFloat64(offset int) (float64, int) {
	v := p.readFixed64At(offset)
	return math.Float64frombits(v), 8
}

func (p *LazyBuffer) ReadFixed32(offset int) uint32 {
	return uint32(p.readFixed32At(offset))
}

func (p *LazyBuffer) SizedReadFixed32(offset int) (uint32, int) {
	return uint32(p.readFixed32At(offset)), 4
}

func (p *LazyBuffer) ReadFixed64(offset int) uint64 {
	return p.readFixed64At(offset)
}

func (p *LazyBuffer) SizedReadFixed64(offset int) (uint64, int) {
	return p.readFixed64At(offset), 8
}

func (p *LazyBuffer) ReadSFixed32(offset int) int32 {
	return int32(p.readFixed32At(offset))
}

func (p *LazyBuffer) SizedReadSFixed32(offset int) (int32, int) {
	return int32(p.readFixed32At(offset)), 4
}

func (p *LazyBuffer) ReadSFixed64(offset int) int64 {
	return int64(p.readFixed64At(offset))
}

func (p *LazyBuffer) SizedReadSFixed64(offset int) (int64, int) {
	return int64(p.readFixed64At(offset)), 8
}

func (p *LazyBuffer) readSignedVarIntAt(offset int) (int64, int) {
	v, n := p.readVarIntAt(offset)
	if n < 0 {
		return 0, n
	}

	return int64(v>>1) ^ int64(v)<<63>>63, n
}

func (p *LazyBuffer) readVarIntAt(offset int) (v uint64, n int) {
	var y uint64
	if !p.hasNext(offset, 0) {
		return 0, -1
	}
	v = uint64(p.buf[offset+0])
	if v < 0x80 {
		return v, 1
	}
	v -= 0x80

	if !p.hasNext(offset, 1) {
		return 0, -1
	}
	y = uint64(p.buf[offset+1])
	v += y << 7
	if y < 0x80 {
		return v, 2
	}
	v -= 0x80 << 7

	if !p.hasNext(offset, 2) {
		return 0, -1
	}
	y = uint64(p.buf[offset+2])
	v += y << 14
	if y < 0x80 {
		return v, 3
	}
	v -= 0x80 << 14

	if !p.hasNext(offset, 3) {
		return 0, -1
	}
	y = uint64(p.buf[offset+3])
	v += y << 21
	if y < 0x80 {
		return v, 4
	}
	v -= 0x80 << 21

	if !p.hasNext(offset, 4) {
		return 0, -1
	}
	y = uint64(p.buf[offset+4])
	v += y << 28
	if y < 0x80 {
		return v, 5
	}
	v -= 0x80 << 28

	if !p.hasNext(offset, 5) {
		return 0, -1
	}
	y = uint64(p.buf[offset+5])
	v += y << 35
	if y < 0x80 {
		return v, 6
	}
	v -= 0x80 << 35

	if !p.hasNext(offset, 6) {
		return 0, -1
	}
	y = uint64(p.buf[offset+6])
	v += y << 42
	if y < 0x80 {
		return v, 7
	}
	v -= 0x80 << 42

	if !p.hasNext(offset, 7) {
		return 0, -1
	}
	y = uint64(p.buf[offset+7])
	v += y << 49
	if y < 0x80 {
		return v, 8
	}
	v -= 0x80 << 49

	if !p.hasNext(offset, 8) {
		return 0, -1
	}
	y = uint64(p.buf[offset+8])
	v += y << 56
	if y < 0x80 {
		return v, 9
	}
	v -= 0x80 << 56

	if !p.hasNext(offset, 9) {
		return 0, -1
	}
	y = uint64(p.buf[offset+9])
	v += y << 63
	if y < 2 {
		return v, 10
	}
	return 0, -1
}
