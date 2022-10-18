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

	buf := NewLazyBuffer(data)
	t.Logf("decoded: %d", buf.ReadSInt64(0))

	if buf.ReadSInt64(0) != 106 {
		t.Fail()
	}
}

func TestVarInt(t *testing.T) {
	buf := NewLazyBuffer(nil)
	buf.AppendInt64(1, -32)

	// expected value: 18446744073709551584
	// expected buf: [8 224 255 255 255 255 255 255 255 255 1]

	t.Logf("buf: %v", buf.buf)

	_, _, size, _ := buf.ReadTagAt(0)
	value, _ := buf.readVarIntAt(size)

	if int64(value) != -32 {
		t.Fatalf("expected value: %v, got: %v", -32, value)
	}
}

func TestDecodeString(t *testing.T) {
	buf := NewLazyBuffer(nil)
	buf.AppendString(1, "hello")
	buf.AppendString(2, "world")

	t.Logf("buf: %v", buf.buf)
	_, _, size, _ := buf.ReadTagAt(0)
	str1, str1size := buf.SizedReadString(size)
	t.Logf("str1: %s, size: %d", str1, str1size)
	_, _, size2, _ := buf.ReadTagAt(size + str1size)
	str2 := buf.ReadString(size + str1size + size2)
	t.Logf("str2: %s", str2)
}
