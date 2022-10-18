package pathutils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func GetProjectRootByMakefile(currentPath string, maxDepth int) (string, error) {
	return searchProjectRootFromTopToBottom(currentPath, 0, maxDepth)
}

func searchProjectRootFromTopToBottom(currentPath string, depth int, maxDepth int) (string, error) {
	if depth > maxDepth {
		return "", fmt.Errorf("No root directory was found, exceeded hierarchy depth of : %d", depth)
	}
	files, err := os.ReadDir(currentPath)
	if err != nil {
		return "", err
	}

	for _, f := range files {
		if f.Name() == "Makefile" {
			return currentPath, nil
		}
	}

	path, err := filepath.Abs(currentPath + "/..")
	if err != nil {
		return "", err
	}

	depth++
	return searchProjectRootFromTopToBottom(path, depth, maxDepth)
}

func MkPath(path string, perm os.FileMode) error {
	if _, result := Exists(path); result {
		return nil
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create path: '%s', error: '%s'", path, err.Error())
	}

	return nil
}

func Exists(path string) (os.FileInfo, bool) {
	if info, err := os.Stat(path); err != nil {
		if os.IsPermission(err) {
			log.Printf("Permission error on file stat: %s, err = %v", path, err.Error())
		}

		return nil, false
	} else {
		return info, true
	}
}
