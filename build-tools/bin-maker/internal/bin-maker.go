package internal

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/olekukonko/tablewriter"

	"octopus/shared/ospathlib"
)

const tag = "-build-me-for:"

const (
	targetGopherJs = "gopherjs"
	targetNative   = "native"
	targetLinuxArm = "arm"
	targetWin      = "windows"
	targetLinux    = "linux"
	targetOsx      = "osx"
)

// we need lock for gopherjs because parallel execution is not allowed for this
var gopherjsLock = sync.Mutex{}

type BuildError struct {
	Err      error
	Command  string
	SOut     []byte
	SErr     []byte
	Duration time.Duration
}

func mkBuildError(cmd string, err error, sout []byte, serr []byte, duration time.Duration) *BuildError {
	return &BuildError{
		Err:      err,
		Command:  cmd,
		SOut:     sout,
		SErr:     serr,
		Duration: duration,
	}
}

func getGoBuildPrefix(target string) (string, error) {
	switch target {
	case targetNative:
		return "", nil
	case targetLinuxArm:
		return "GOOS=linux GOARCH=arm64", nil
	case targetWin:
		return "GOOS=windows GOARCH=amd64", nil
	case targetLinux:
		return "GOOS=linux GOARCH=amd64", nil
	case targetOsx:
		return "GOOS=darwin GOARCH=amd64", nil
	case targetGopherJs:
		return "GOPHERJS_GOROOT=\"$(go1.17.11 env GOROOT)\" GOOS=darwin GOPHERJS_SKIP_VERSION_CHECK=true", nil
	}

	return "", fmt.Errorf("unknown binary target in %s tag", tag)
}

var whitespace = []string{" ", ",", "\n", "\t"}

func sanityFix(src string) string {
	for _, ws := range whitespace {
		src = strings.ReplaceAll(src, ws, "")
	}

	return src
}

type BinMakerConfig struct {
	Root     string
	Excludes map[string]struct{}
	Includes map[string]struct{}
}

func MakeBinaries(config BinMakerConfig) {
	tms := time.Now()
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	doneChannel := make(chan *BuildError)
	spawnedKids := 0

	err = filepath.Walk(filepath.Join(cwd, config.Root), func(path string, info fs.FileInfo, err error) error {
		relativePath, err := ospathlib.SubtractPath(path, cwd+"/")
		if err != nil {
			panic(err)
		}
		if relativePath == "" {
			relativePath = "."
		}
		if info == nil {
			log.Panicf("empty info for %v", path)
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			return fileWalker(relativePath, cwd, config, doneChannel, &spawnedKids)
		} else {
			return nil
		}
	})

	if err != nil {
		log.Println("Got error during fs walk: ", err)
		os.Exit(-1)
	}

	var errors []*BuildError
	var okTableContent [][]string

	for i := 0; i < spawnedKids; i++ {
		if err := <-doneChannel; err.Err != nil {
			errors = append(errors, err)
		} else {
			okTableContent = append(okTableContent, []string{
				err.Command,
				fmt.Sprintf("%v", aurora.Green(err.Duration)),
			})
		}
	}

	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetHeader([]string{
		"cmd",
		"duration",
	})
	tw.SetRowLine(true)
	tw.SetRowSeparator("-")
	tw.AppendBulk(okTableContent)
	tw.Render()

	if len(errors) != 0 {
		for _, err2 := range errors {
			fmt.Printf("%s %s\n\tstderr: %s\n\tstdout: %s\n",
				aurora.Red("ERR"),
				aurora.Yellow(err2.Command),
				string(err2.SErr),
				string(err2.SOut))
		}

		fmt.Printf("%s building binaries in %v\n", aurora.Red("Failed"), time.Since(tms))
		os.Exit(-1)
	} else {
		fmt.Printf("%s building binaries in %v\n", aurora.Green("Done"), time.Since(tms))
	}
}

func fileWalker(
	path string,
	cwd string,
	config BinMakerConfig,
	doneChannel chan *BuildError,
	spawnedKids *int,
) error {
	f, err := tryParseGoFile(path)
	if err != nil {
		log.Printf("error parsing file %s, with error %v\n", path, err)
		return err
	}

	if hasMainFunction(f) {
		targets := extractTargetsFromGoFile(f, config)

		if len(targets) > 0 {
			launchTheBuilds(targets, path, cwd, doneChannel, spawnedKids)
		}
	}

	return nil
}

func extractTargetsFromGoFile(f *ast.File, config BinMakerConfig) []string {
	var targets = parseBuildInstructions(f)

	if len(config.Excludes) > 0 {
		var filteredTargets []string
		for _, src := range targets {
			if _, exists := config.Excludes[src]; !exists {
				filteredTargets = append(filteredTargets, src)
			}
		}
		targets = filteredTargets
	}

	if len(config.Includes) > 0 {
		var filteredTargets []string
		for _, src := range targets {
			if _, exists := config.Includes[src]; exists {
				filteredTargets = append(filteredTargets, src)
			}
		}
		targets = filteredTargets
	}
	return targets
}

func runTimedCommand(cmd string, doneChannel chan *BuildError) {
	if strings.Contains(cmd, targetGopherJs) {
		gopherjsLock.Lock()
		defer gopherjsLock.Unlock()
	}
	ts := time.Now()
	process := exec.Command("/bin/sh", "-c", cmd)
	process.Stdin = os.Stdin
	stdout, err := process.StdoutPipe()
	if err != nil {
		doneChannel <- mkBuildError(cmd, err, nil, nil, time.Since(ts))
		return
	}
	stderr, err := process.StderrPipe()
	if err != nil {
		doneChannel <- mkBuildError(cmd, err, nil, nil, time.Since(ts))
		return
	}

	err = process.Start()
	if err != nil {
		doneChannel <- mkBuildError(cmd, err, nil, nil, time.Since(ts))
		return
	}

	so, _ := io.ReadAll(stdout)
	se, _ := io.ReadAll(stderr)

	if err = process.Wait(); err != nil {
		doneChannel <- mkBuildError(cmd, err, so, se, time.Since(ts))
		return
	}

	doneChannel <- mkBuildError(cmd, err, so, se, time.Since(ts))
}

func tryParseGoFile(path string) (*ast.File, error) {
	fset := token.NewFileSet()
	src, err := os.ReadFile(path)
	if err != nil {
		log.Printf("error reading file %s, with error %v\n", path, err)
		return nil, err
	}
	return parser.ParseFile(fset, path, src, parser.ParseComments)
}

func launchTheBuilds(targets []string, path string, cwd string, doneChannel chan *BuildError, spawnedKids *int) {
	fmt.Printf("file: %s, packaging for %d platforms %v\n",
		aurora.Yellow(path),
		aurora.Blue(len(targets)),
		aurora.Green(targets))

	pathTokens := strings.Split(path, "/")
	fileName := pathTokens[len(pathTokens)-1]
	binaryName := strings.ReplaceAll(fileName, ".go", "")

	for _, target := range targets {
		goEnv, err := getGoBuildPrefix(target)
		extras := ""
		if err != nil {
			log.Panicf("error processing '%s' directive... for %v!\n", tag, path)
		} else {
			var outPath string
			var goBin = "go"
			var goMods = "-mod vendor"

			if goEnv == "" && target != targetGopherJs {
				outPath = fmt.Sprintf("target/%s", binaryName)
			} else {
				if target == targetGopherJs {
					goBin = targetGopherJs
					goMods = ""
					extras = "-m"
					outPath = fmt.Sprintf("target/%s/%s.js", binaryName, binaryName)
				} else {
					cleanedTarget := target
					if strings.HasSuffix(cleanedTarget, "-docker") {
						cleanedTarget = strings.TrimSuffix(cleanedTarget, "-docker")
					}
					outPath = fmt.Sprintf("target/%s.%s", binaryName, cleanedTarget)
				}
			}

			cmd := ""
			if goEnv == "" && target != targetGopherJs {
				cmd = fmt.Sprintf("%s build %s -o %s %s", goBin, goMods, outPath, path)
			} else {
				cmd = fmt.Sprintf("%s %s build %s %s -o %s %s", goEnv, goBin, goMods, extras, outPath, path)
			}

			*spawnedKids++

			go runTimedCommand(cmd, doneChannel)
		}
	}
}

func parseBuildInstructions(f *ast.File) []string {
	var targets []string
	for _, comment := range f.Comments {
		parts := strings.Split(comment.Text(), "\n")
		for _, c := range parts {
			if strings.HasPrefix(c, tag) {
				tokens := strings.SplitN(c, ":", 2)
				if len(tokens) == 2 {
					platforms := strings.Split(tokens[1], ",")
					for _, platform := range platforms {
						platform = sanityFix(platform)
						targets = append(targets, platform)
					}
				}
			}
		}
	}

	return targets
}

func hasMainFunction(f *ast.File) bool {
	if f.Name.Name == "main" {
		for _, d := range f.Decls {
			if gen, ok := d.(*ast.FuncDecl); ok {
				if gen.Name.Name == "main" {
					return true
				}
			}
		}
	}

	return false
}
