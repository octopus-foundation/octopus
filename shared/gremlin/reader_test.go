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

Created by ab, 05.10.2022
*/

package gremlin

import (
	"testing"
)

func TestDecodeInt64(t *testing.T) {
	var data = []byte{212, 1}

	buf := NewReader(data)
	t.Logf("decoded: %d", buf.ReadSInt64(0))

	if buf.ReadSInt64(0) != 106 {
		t.Fail()
	}
}

func TestVarInt(t *testing.T) {
	size := SizeInt64(-32) + SizeTag(1)
	buf := NewWriter(size)
	buf.AppendInt64(1, -32)

	// expected value: 18446744073709551584
	// expected buf: [8 224 255 255 255 255 255 255 255 255 1]

	t.Logf("buf: %v, size %v", buf.Bytes(), size)
}
