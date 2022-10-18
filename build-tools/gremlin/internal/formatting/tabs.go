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

Created by ab, 13.10.2022
*/

package formatting

import "strings"

func AddTabs(s string, tabs string) string {
	lines := strings.Split(s, "\n")
	res := &strings.Builder{}
	for i, line := range lines {
		res.WriteString(tabs)
		res.WriteString(line)
		if i != len(lines)-1 {
			res.WriteString("\n")
		}
	}
	return res.String()
}
