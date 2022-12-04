package ospathlib

import (
	"fmt"
	"testing"
)

func TestSubtractWorks(t *testing.T) {
	path, err := SubtractPath("/this/is/path", "/this/is")
	if err != nil {
		panic(err)
	}

	if path != "/path" {
		panic(fmt.Sprintf("got %s instead of %s", path, "/path"))
	}
}

func TestLeastCommonPath(t *testing.T) {
	lcp, err := leastCommonPath2("/usr/local/Cellar/pari/bin", "/usr/local/Cellar/pari", "/usr/local/")

	if err != nil {
		panic(err)
	}

	if lcp != "/usr/local/" {
		panic(fmt.Sprintf("got %s instead of /usr/local/", lcp))
	}
}
