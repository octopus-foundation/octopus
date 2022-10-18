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

package main

import (
	"flag"
	"fmt"
	"github.com/logrusorgru/aurora"
	"log"
	"octopus/build-tools/gremlin/internal"
	"octopus/build-tools/gremlin/internal/generators/golang"
	pathutils "octopus/shared/path-utils"
	"os"
	"time"
)

var rootPath = flag.String("root", "", "root path for protobuf generation")

func main() {
	t := time.Now()
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var rootDir = *rootPath
	if rootDir == "" {
		rootDir, err = pathutils.GetProjectRootByMakefile(cwd, 5)
		if err != nil {
			log.Panicf("Failed to find project root dir by Makefile: %v", err.Error())
		}
	}

	if err = internal.CreateTargetFolder(rootDir); err != nil {
		panic(err.Error())
	}

	targets, err := internal.FindAllProtobufFiles(rootDir)
	if err != nil {
		panic(err.Error())
	}

	if err = internal.ParseProtoFiles(targets); err != nil {
		panic(err.Error())
	}

	errors := internal.ParseStruct(targets)
	if len(errors) > 0 {
		for _, err = range errors {
			fmt.Printf("%v: %v\n", aurora.Red("ERR"), err.Error())
		}
		os.Exit(-1)
	}

	errors = internal.ResolveImportsAndReferences(targets)
	if len(errors) > 0 {
		for _, err = range errors {
			fmt.Printf("%v: %v\n", aurora.Red("ERR"), err.Error())
		}
		os.Exit(-1)
	}

	fmt.Printf("All files parsed and analyzed in %v\n", aurora.Yellow(time.Since(t)))
	fmt.Printf("Generating golang files...\n")

	errors = golang.Generate(cwd, targets)
	if len(errors) > 0 {
		for _, err = range errors {
			fmt.Printf("%v: %v\n", aurora.Red("ERR"), err.Error())
		}
		os.Exit(-1)
	}

	fmt.Printf("Done in %v\n", aurora.Yellow(time.Since(t)))
}
