package main

import (
	"flag"
	"octopus/build-tools/bin-maker/internal"
	"octopus/shared/ulimits"
	"strings"
)

var root = flag.String("root", "", "go binaries search root")
var excludes = flag.String("excludes", "", "targets to exclude, e.g.: arm-docker, space-separated")
var includes = flag.String("only", "", "targets to include, e.g.: arm-docker, space-separated")

func main() {
	flag.Parse()
	ulimits.SetupForHighLoad()

	var parsedExcludes map[string]struct{}
	if *excludes != "" {
		parsedExcludes = map[string]struct{}{}
		for _, target := range strings.Split(*excludes, " ") {
			parsedExcludes[target] = struct{}{}
		}
	}

	var parsedIncludes map[string]struct{}
	if *includes != "" {
		parsedIncludes = map[string]struct{}{}
		for _, target := range strings.Split(*includes, " ") {
			parsedIncludes[target] = struct{}{}
		}
	}

	internal.MakeBinaries(internal.BinMakerConfig{
		Root:     *root,
		Excludes: parsedExcludes,
		Includes: parsedIncludes,
	})
}
