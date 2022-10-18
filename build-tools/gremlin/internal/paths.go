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

package internal

import (
	"io/fs"
	"log"
	"octopus/build-tools/gremlin/internal/types"
	"os"
	"path/filepath"
	"strings"
)

var protoIgnore = map[string]struct{}{
	"cpp-legacy":   {},
	"node_modules": {},
	"vendor":       {},
}

func FindAllProtobufFiles(cwd string) ([]*types.ProtoFile, error) {
	var files []*types.ProtoFile
	if err := filepath.Walk(cwd, func(filePath string, info fs.FileInfo, err error) error {
		if info.IsDir() || !strings.HasSuffix(filePath, types.ProtoExtension) {
			return nil
		}
		pathParts := strings.Split(filePath, string(os.PathSeparator))
		for _, part := range pathParts {
			if _, ignored := protoIgnore[part]; ignored {
				return nil
			}
		}

		pFile := &types.ProtoFile{Path: filePath}
		pFile.RelativePath = buildRelativePath(cwd, pFile)

		files = append(files, pFile)

		return err
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func buildRelativePath(root string, pFile *types.ProtoFile) string {
	if !strings.HasPrefix(pFile.Path, root) {
		log.Panicf("non-root relative path: %v", pFile.Path)
	}

	relative := strings.TrimPrefix(pFile.Path, root)
	relative = strings.TrimPrefix(relative, "/protobufs/")
	return relative
}

func CreateTargetFolder(cwd string) error {
	return os.MkdirAll(filepath.Join(cwd, types.TargetFolder), 0755)
}
