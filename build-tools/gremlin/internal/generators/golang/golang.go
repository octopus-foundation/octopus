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

Created by ab, 27.09.2022
*/

package golang

import (
	"octopus/build-tools/gremlin/internal/generators/golang/mapper"
	"octopus/build-tools/gremlin/internal/types"
	"os"
	"path/filepath"
)

func Generate(root string, targets []*types.ProtoFile) []error {
	mapped, errors := mapper.MapProtoFiles(root, targets)
	if len(errors) > 0 {
		return errors
	}

	for _, target := range mapped {
		content := target.GenerateCode()
		_ = os.MkdirAll(filepath.Dir(target.FullOutputPath), 0755)
		if err := os.WriteFile(target.FullOutputPath, []byte(content), 0644); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
