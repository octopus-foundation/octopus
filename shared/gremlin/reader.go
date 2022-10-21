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

Created by ab, 03.10.2022
*/

package gremlin

type ProtoWireType int8
type ProtoWireNumber int32

const (
	VarIntType     ProtoWireType = 0
	Fixed64Type    ProtoWireType = 1
	BytesType      ProtoWireType = 2
	StartGroupType ProtoWireType = 3
	EndGroupType   ProtoWireType = 4
	Fixed32Type    ProtoWireType = 5
)

type Reader struct {
	buf []byte
}

func NewReader(data []byte) *Reader {
	return &Reader{
		buf: data,
	}
}

func (p *Reader) Bytes() []byte {
	if p == nil {
		return nil
	}
	return p.buf
}
