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

Created by ab, 19.10.2022
*/

package gremlin

import "testing"

func TestSizeBool(t *testing.T) {
	buf := NewWriter(2)
	buf.AppendBool(1, true)
	if len(buf.Bytes()) != 2 {
		t.Errorf("Expected size 2, got %d", buf.Bytes())
	}
}
