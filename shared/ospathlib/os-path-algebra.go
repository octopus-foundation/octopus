package ospathlib

import (
	"fmt"
	"strings"
)

func leastCommonPath(p1, p2 string) (string, error) {
	if p1[0] != p2[0] || p1[0] != '/' {
		return "", fmt.Errorf("bot path should start from root")
	}

	for pos, _ := range p1 {
		if pos+1 > len(p1) || pos+1 > len(p2) || p1[pos] != p2[pos] {
			return p1[:pos], nil
		}
	}

	return "", nil
}

func leastCommonPath2(p1 string, p2 ...string) (string, error) {
	lcp, err := leastCommonPath(p1, p2[0])
	if err != nil {
		return "", err
	}

	if len(p2) == 1 {
		return lcp, nil
	}

	return leastCommonPath2(p1, p2[len(p2)-1:]...)
}

func SubtractPath(minuend, subtrahend string) (string, error) {
	if minuend[0] != subtrahend[0] || minuend[0] != '/' {
		return "", fmt.Errorf("bot path should start from root")
	}

	for pos, _ := range minuend {
		if pos+1 > len(subtrahend) || minuend[pos] != subtrahend[pos] {
			return minuend[pos:], nil
		}
	}

	return "", nil
}

func GetFileName(path string) string {
	tokens := strings.Split(path, "/")
	return tokens[len(tokens)-1]
}

func GetFileDirPath(path string) string {
	tokens := strings.Split(path, "/")
	fileDirPath := ""
	for _, token := range tokens[:len(tokens)-1] {
		if token == "" {
			continue
		}
		if fileDirPath == "" {
			fileDirPath = fmt.Sprintf("/%s", token)
		} else {
			fileDirPath = fmt.Sprintf("%s/%s", fileDirPath, token)
		}
	}

	return fileDirPath
}
