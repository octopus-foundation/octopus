package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/logrusorgru/aurora"
	"github.com/yosuke-furukawa/json5/encoding/json5"
	"gopkg.in/yaml.v3"
)

const tag = "-generate-configs:"

var path = flag.String("path", "", "appsconfigs search custom base path")
var ansible = flag.Bool("ansible", false, "force ansible build with custom base path")

func main() {
	flag.Parse()

	tms := time.Now()

	cwd, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}

	if *path != "" {
		log.Printf("custom path provided: %s", *path)
	}
	if *ansible {
		log.Printf("forcing ansible gen")
	}

	configs := searchForGen(*path, *ansible)

	parsed, err := parseGoSources(configs)
	if err != nil {
		panic(err.Error())
	}

	wg := sync.WaitGroup{}
	wg.Add(len(configs))
	for _, config := range configs {
		go func(config *genConfig) {
			log.Printf("Processing %v as %v\n", Green(config.configPath), Yellow(config.targetType))
			if processErr := processConfigFile(cwd, parsed, config); processErr != nil {
				log.Panicf("Failed to process %v: %v", config.configPath, processErr.Error())
			}
			wg.Done()
		}(config)
	}
	wg.Wait()

	var groups = map[string][]*genConfig{}
	for _, config := range configs {
		if config.group != "" {
			k := config.group + ":" + config.goGenSrcPath
			groups[k] = append(groups[k], config)
		}
	}

	for _, genConfigs := range groups {
		if err := generateConfigsGroup(cwd, genConfigs); err != nil {
			panic(err.Error())
		}
	}

	log.Printf("Done building %v configs in %v\n", len(configs), time.Since(tms))
}

type genConfig struct {
	folder        string
	goGenSrcPath  string
	configPath    string
	packageName   string
	targetType    string
	buildComments []string
	group         string

	generatedVar              string
	generatedPath             string
	generatedType             types.Type
	generatedPackageShortName string
	generatedPackageFullName  string // edge cases? for imports
}

func searchForGen(customPath string, forceAnsible bool) []*genConfig {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var res []*genConfig

	directories := []string{"appsconfigs", "ansible"}
	if customPath != "" {
		directories = []string{customPath}
		if forceAnsible {
			directories = append(directories, "ansible")
		}
	}
	for _, dir := range directories {
		err = filepath.Walk(filepath.Join(cwd, dir), func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(path) == ".go" {
				res = append(res, checkGoSource(path)...)
			}
			return nil
		})
		if err != nil {
			log.Panicf("Got error during fs walk: %v", err)
		}
	}
	return res
}

func checkGoSource(path string) []*genConfig {
	goPath := path
	fileContent, err := os.ReadFile(path)
	var res []*genConfig
	if err != nil {
		panic(err.Error())
	}
	if strings.Contains(string(fileContent), tag) {
		fset := token.NewFileSet()

		parsed, err := parser.ParseFile(fset, path, fileContent, parser.ParseComments)
		if err != nil {
			panic(err.Error())
		}

		if len(parsed.Comments) == 0 {
			return nil
		}

		entryFound := false
		for _, comment := range parsed.Comments {
			if strings.Contains(comment.Text(), tag) {
				entryFound = true
				break
			}
		}

		if !entryFound {
			return nil
		}

		buildComments := extractBuildComments(parsed)

		for _, comment := range parsed.Comments {
			commentText := comment.Text()
			commentText = strings.TrimSpace(commentText)
			entries := strings.Split(commentText, "\n")

			for _, txt := range entries {
				if !strings.HasPrefix(txt, tag) {
					continue
				}
				txt = strings.TrimPrefix(txt, tag)
				txt = strings.TrimSpace(txt)

				parts := strings.Split(txt, " ")
				if len(parts) != 2 {
					log.Panicf("Invalid syntax: %v, cannot split", comment.Text())
				}

				res = extractGenConfigsFromGoComments(path, parts, res, goPath, buildComments)
			}
		}
	}

	return res
}

func extractGenConfigsFromGoComments(path string, parts []string, res []*genConfig, goPath string, buildComments []string) []*genConfig {
	configMask := parts[0]
	parseType := parts[1]
	baseFolder := filepath.Dir(path)

	if strings.Contains(configMask, "*") {
		res = findTargetConfigsByMasks(baseFolder, configMask, res, goPath, parseType, buildComments)
	} else {
		res = append(res, &genConfig{
			goGenSrcPath:  goPath,
			folder:        baseFolder,
			configPath:    filepath.Join(baseFolder, configMask),
			packageName:   filepath.Dir(parseType),
			targetType:    filepath.Base(parseType),
			buildComments: buildComments,
		})
	}
	return res
}

func findTargetConfigsByMasks(baseFolder string, configMask string, res []*genConfig, goPath string, parseType string, buildComments []string) []*genConfig {
	if err := filepath.Walk(baseFolder, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(baseFolder, path)
		if err != nil {
			return err
		}

		matched, err := filepath.Match(configMask, relPath)
		if err != nil {
			return err
		}

		if matched {
			res = append(res, &genConfig{
				goGenSrcPath:  goPath,
				folder:        filepath.Dir(path),
				configPath:    path,
				packageName:   filepath.Dir(parseType),
				targetType:    filepath.Base(parseType),
				buildComments: buildComments,
				group:         configMask,
			})
		}

		return nil
	}); err != nil {
		log.Panicf("failed to search configs: %v", err.Error())
	}
	return res
}

func extractBuildComments(parsed *ast.File) []string {
	var buildComments []string
	for _, comment := range parsed.Comments {
		txt := comment.Text()
		if strings.Contains(txt, "+build") {
			parts := strings.Split(txt, "\n")
			for _, part := range parts {
				if strings.Contains(part, "+build") {
					buildComments = append(buildComments, part)
				}
			}
		}
	}
	return buildComments
}

type goSourcesState struct {
	packages map[string]*types.Package
	fset     *token.FileSet
}

func parseGoSources(configs []*genConfig) (*goSourcesState, error) {
	res := &goSourcesState{
		fset:     token.NewFileSet(),
		packages: map[string]*types.Package{},
	}

	var dirs = map[string]bool{}
	for _, config := range configs {
		dirs[config.packageName] = true
	}

	var srcFiles = map[string][]*ast.File{}

	for dirName := range dirs {
		parsed, err := parser.ParseDir(res.fset, dirName, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		for _, a := range parsed {
			for _, file := range a.Files {
				srcFiles[dirName] = append(srcFiles[dirName], file)
			}
		}
	}

	var conf = types.Config{
		Importer: importer.ForCompiler(res.fset, "source", nil),
	}
	var info = types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	for pkgName := range dirs {
		pkg, err := conf.Check(pkgName, res.fset, srcFiles[pkgName], &info)
		if err != nil {
			return nil, err
		}
		res.packages[pkgName] = pkg
	}

	return res, nil
}

func processConfigFile(baseDir string, goParsed *goSourcesState, config *genConfig) error {
	parsedStruct, err := parseConfigFile(config.configPath)
	if err != nil {
		return err
	}

	res, err := types.Eval(goParsed.fset, goParsed.packages[config.packageName], token.NoPos, config.targetType)
	if err != nil {
		return err
	}
	var state = &generatorState{
		imports: map[string]struct{}{},
		buf:     bytes.NewBufferString(""),
	}
	if err := generateGoSource(res.Type, parsedStruct, state); err != nil {
		return err
	}

	targetName := filepath.Base(config.configPath)
	resName := genVariableName(targetName)
	extraTags := genExtraTags(targetName)

	var generated = "// Code generated by json-gogen. DO NOT EDIT.\n\n"
	for _, comment := range config.buildComments {
		generated += "// " + comment + "\n"
	}
	for _, extraTag := range extraTags {
		generated += "// +build " + extraTag + "\n"
	}

	genPackage := filepath.Base(filepath.Dir(config.configPath))
	genPackage = strings.ReplaceAll(genPackage, "-", "_")
	generated += fmt.Sprintf("\npackage %v\n\n", genPackage)
	generated += generateImports(state.imports)
	generated += "\n"

	generatedType, err := generateTypeInstanceString(res.Type, state, false)
	if err != nil {
		return err
	}

	generated += fmt.Sprintf("var %v %v", resName, generatedType)
	if parsedStruct != nil {
		generated += " = " + state.buf.String()
	}

	cleanedTargetPath := filepath.Base(config.configPath)
	cleanedTargetPath = strings.ReplaceAll(cleanedTargetPath, "+", "")

	relPath, err := filepath.Rel(baseDir, filepath.Dir(config.configPath))
	if err != nil {
		return err
	}

	targetDir := filepath.Join("target", "generated-sources", relPath)
	targetFile := filepath.Join(targetDir, cleanedTargetPath+".go")

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	parsedFile, err := parser.ParseFile(goParsed.fset, targetFile, []byte(generated), parser.ParseComments)
	if err != nil {
		_ = os.WriteFile(targetFile, []byte(generated), 0755)
		return err
	}

	log.Printf("generating %s", Green(targetFile))

	f, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	if err := printer.Fprint(w, goParsed.fset, parsedFile); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	if err := goFmt(targetFile); err != nil {
		return err
	}

	config.generatedPath = targetFile
	config.generatedVar = resName
	config.generatedType = res.Type
	config.generatedPackageShortName = genPackage
	config.generatedPackageFullName = targetDir

	return nil
}

func goFmt(fname string) error {
	b, err := os.ReadFile(fname)
	if err != nil {
		return err
	}

	formatted, err := format.Source(b)
	if err != nil {
		return err
	}

	return os.WriteFile(fname, formatted, 0644)
}

func genVariableName(targetName string) string {
	if strings.Contains(targetName, ".+") {
		targetName = strings.Split(targetName, ".+")[0]
	}
	targetNameShouldUpper := true
	resName := ""
	for _, c := range targetName {
		switch c {
		case '-', '_', '.', ' ':
			targetNameShouldUpper = true
		default:
			if targetNameShouldUpper {
				resName += strings.ToUpper(string(c))
				targetNameShouldUpper = false
			} else {
				resName += string(c)
			}
		}
	}
	return resName
}

func genExtraTags(targetName string) []string {
	targetName = strings.TrimSuffix(targetName, filepath.Ext(targetName))
	if strings.Contains(targetName, ".+") {
		return strings.Split(targetName, ".+")[1:]
	} else {
		return nil
	}
}

type generatorState struct {
	imports map[string]struct{}
	buf     *bytes.Buffer
}

func generateGoSource(goType types.Type, val interface{}, state *generatorState) error {
	if val == nil {
		state.buf.WriteString("nil")
	}
	namedType, isNamed := goType.(*types.Named)
	if isNamed {
		return generateGoNamedType(namedType, val, state)
	}
	mapType, isMap := goType.(*types.Map)
	if isMap {
		return generateGoMap(mapType, val, state)
	}

	sliceType, isSlice := goType.(*types.Slice)
	if isSlice {
		return generateGoSlice(val, sliceType, state)
	}

	structType, isStruct := goType.(*types.Struct)
	if isStruct {
		return generateGoStruct(val, structType, state)
	}

	basicType, isBasic := goType.(*types.Basic)
	if isBasic {
		if basicType.Kind() == types.String {
			state.buf.WriteString(fmt.Sprintf("%q", val))
		} else {
			state.buf.WriteString(fmt.Sprintf("%v", val))
		}
		return nil
	}

	pointerType, isPointer := goType.(*types.Pointer)
	if isPointer {
		state.buf.WriteString("&")
		return generateGoSource(pointerType.Elem(), val, state)
	}

	if _, isInterface := goType.(*types.Interface); isInterface {
		if s, ok := val.(string); ok {
			state.buf.WriteString(fmt.Sprintf("%s", strconv.Quote(s)))
			return nil
		}
		if i, ok := val.(int); ok {
			state.buf.WriteString(fmt.Sprintf("%d", i))
			return nil
		}
		state.buf.WriteString(strconv.Quote(fmt.Sprintf("%v", val)))
		return nil
	}

	return fmt.Errorf("unknown type: %v %v", basicType, val)
}

func generateGoStruct(val interface{}, structType *types.Struct, state *generatorState) error {
	if val == nil {
		state.buf.WriteString("nil")
		return nil
	}
	casted, ok := val.(map[string]interface{})
	if !ok {
		res := make(map[string]interface{})
		for k, v := range val.(map[interface{}]interface{}) {
			res[k.(string)] = v
		}
		casted = res
	}
	structDef, err := generateTypeInstanceString(structType, state, false)
	if err != nil {
		return err
	}
	state.buf.WriteString(structDef)
	state.buf.WriteString("{\n")
	for i := 0; i < structType.NumFields(); i++ {
		tag := structType.Tag(i)
		if tag == "" {
			continue
		}
		castedTag := reflect.StructTag(tag)
		targetKey, _ := castedTag.Lookup("json")
		if targetKey == "" {
			targetKey, _ = castedTag.Lookup("yaml")
		}
		if targetKey == "" {
			continue
		}
		if strings.Contains(targetKey, ",") {
			targetKey = strings.Split(targetKey, ",")[0]
		}

		value, found := casted[targetKey]
		if found && value != nil {
			field := structType.Field(i)
			state.buf.WriteString(field.Name() + ":")
			if err := generateGoSource(field.Type(), value, state); err != nil {
				return err
			}
			state.buf.WriteString(",\n")
		}
	}
	state.buf.WriteString("}")
	return nil
}

func generateGoSlice(val interface{}, sliceType *types.Slice, state *generatorState) error {
	if val == nil {
		state.buf.WriteString("nil")
		return nil
	}
	var casted = val.([]interface{})
	typeDef, err := generateTypeInstanceString(sliceType, state, false)
	if err != nil {
		return err
	}
	state.buf.WriteString(typeDef)
	state.buf.WriteString("{\n")
	for _, entry := range casted {
		if err := generateGoSource(sliceType.Elem(), entry, state); err != nil {
			return err
		}
		state.buf.WriteString(",\n")
	}
	state.buf.WriteString("}")
	return nil
}

func generateGoNamedType(namedType *types.Named, val interface{}, state *generatorState) error {
	if namedType.Obj().Name() == "Duration" && namedType.Obj().Pkg().Path() == "time" {
		state.imports["go:"+namedType.Obj().Pkg().Path()] = struct{}{}
		var parsedValue time.Duration
		var err error
		if strVal, isStr := val.(string); isStr {
			parsedValue, err = time.ParseDuration(strVal)
			if err != nil {
				return err
			}
		} else if intVal, isInt := val.(int); isInt {
			parsedValue = time.Duration(intVal)
		}
		state.buf.WriteString("time.Duration(" + strconv.Itoa(int(parsedValue)) + ")")
		return nil
	} else if namedType.Obj().Name() == "Time" && namedType.Obj().Pkg().Path() == "time" {
		state.imports["go:"+namedType.Obj().Pkg().Path()] = struct{}{}
		var parsedValue time.Time
		var err error
		if strVal, isStr := val.(string); isStr {
			parsedValue, err = time.Parse(time.RFC3339, strVal)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("time.Time value is not string %v", val)
		}

		zoneName, zoneOffset := parsedValue.Zone()
		timeInitStr := fmt.Sprintf("time.Date(%d, time.Month(%d), %d, %d, %d, %d, %d, time.FixedZone(\"%s\", %d))",
			parsedValue.Year(),
			int(parsedValue.Month()),
			parsedValue.Day(),
			parsedValue.Hour(),
			parsedValue.Minute(),
			parsedValue.Second(),
			parsedValue.Nanosecond(),
			zoneName,
			zoneOffset)
		state.buf.WriteString(timeInitStr)
		return nil
	} else {
		under := namedType.Underlying()
		_, isBasic := under.(*types.Basic)
		if isBasic {
			return generateGoSource(under, val, state)
		} else {
			state.imports[namedType.Obj().Pkg().Path()] = struct{}{}
			if _, nestedIsStruct := namedType.Underlying().(*types.Struct); nestedIsStruct {
				typedef, err := generateTypeInstanceString(namedType, state, false)
				if err != nil {
					return err
				}
				state.buf.WriteString(typedef)
			}
			return generateGoSource(under, val, state)
		}
	}
}

func generateTypeInstanceString(goType types.Type, state *generatorState, full bool) (string, error) {
	if namedType, isNamed := goType.(*types.Named); isNamed {
		state.imports[namedType.Obj().Pkg().Path()] = struct{}{}
		return namedType.Obj().Pkg().Name() + "." + namedType.Obj().Name(), nil
	} else if basicType, isBasic := goType.(*types.Basic); isBasic {
		return basicType.Name(), nil
	} else if mapType, isMap := goType.(*types.Map); isMap {
		keyType, err := generateTypeInstanceString(mapType.Key(), state, false)
		if err != nil {
			return "", err
		}
		valType, err := generateTypeInstanceString(mapType.Elem(), state, true)
		if err != nil {
			return "", err
		}
		return "map[" + keyType + "]" + valType, nil
	} else if sliceType, isSlice := goType.(*types.Slice); isSlice {
		valType, err := generateTypeInstanceString(sliceType.Elem(), state, false)
		if err != nil {
			return "", err
		}
		return "[]" + valType, nil
	} else if _, isStruct := goType.(*types.Struct); isStruct {
		return "", nil
	} else if pointerType, isPointer := goType.(*types.Pointer); isPointer {
		valType, err := generateTypeInstanceString(pointerType.Elem(), state, false)
		if err != nil {
			return "", err
		}
		return "*" + valType, nil
	} else if _, isInterface := goType.(*types.Interface); isInterface {
		if full {
			return "interface{}", nil
		} else {
			return "", nil
		}
	} else {
		return "", fmt.Errorf("unknown type: %v", goType)
	}
}

func generateGoMap(mapType *types.Map, val interface{}, state *generatorState) error {
	namedKeyType, keyIsNamed := mapType.Key().(*types.Named)
	var keyType *types.Basic
	if keyIsNamed {
		keyType = namedKeyType.Underlying().(*types.Basic)
	} else {
		keyType = mapType.Key().(*types.Basic)
	}
	valType := mapType.Elem()

	typeDef, err := generateTypeInstanceString(mapType, state, false)
	if err != nil {
		return err
	}
	state.buf.WriteString(typeDef)
	state.buf.WriteString("{\n")

	if keyType.Kind() == types.String {
		castedVal, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed to cast json map: %v", val)
		}
		for k, i := range castedVal {
			state.buf.WriteString(fmt.Sprintf("%q:", k))
			if err := generateGoSource(valType, i, state); err != nil {
				return err
			}
			state.buf.WriteString(",\n")
		}
	} else {
		castedVal, ok := val.(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf("failed to cast string json map: %v", val)
		}
		for k, i := range castedVal {
			state.buf.WriteString(fmt.Sprintf("%v:", k))
			if err := generateGoSource(valType, i, state); err != nil {
				return err
			}
			state.buf.WriteString(",\n")
		}
	}

	state.buf.WriteString("}")
	return nil
}

func parseConfigFile(path string) (interface{}, error) {
	configBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ext = filepath.Ext(path)
	ext = strings.TrimPrefix(ext, ".")
	ext = strings.ToLower(ext)

	var res interface{}
	switch ext {
	case "json":
		if err := json.Unmarshal(configBytes, &res); err != nil {
			return nil, err
		}
	case "json5":
		if err := json5.Unmarshal(configBytes, &res); err != nil {
			return nil, err
		}
	case "yaml", "yml":
		if err := yaml.Unmarshal(configBytes, &res); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func generateConfigsGroup(cwd string, configs []*genConfig) error {
	groupGoSrc := configs[0].goGenSrcPath
	relPath, err := filepath.Rel(cwd, groupGoSrc)
	if err != nil {
		return err
	}

	goPath := filepath.Join("target", "generated-sources", relPath)

	goPkg := filepath.Base(filepath.Dir(goPath))
	goPkg = strings.ReplaceAll(goPkg, "-", "_")
	goPkg = strings.ReplaceAll(goPkg, ".", "_")
	goPkg = strings.ReplaceAll(goPkg, "-", "_")

	var generated = "// Code generated by json-gogen. DO NOT EDIT.\n\n"
	for _, comment := range configs[0].buildComments {
		generated += "// " + comment + "\n"
	}

	generated += "\npackage " + goPkg + "\n\n"
	// probably - edge cases like **/*, etc
	groupName := genVariableName(filepath.Base(filepath.Dir(configs[0].group)))

	typeState := &generatorState{
		imports: map[string]struct{}{},
		buf:     bytes.NewBufferString(""),
	}
	typeDef, err := generateTypeInstanceString(configs[0].generatedType, typeState, false)

	var generatedPackages = map[string]struct{}{}
	for _, config := range configs {
		generatedPackages[config.generatedPackageFullName] = struct{}{}
	}

	generated += generateImports(generatedPackages)
	generated += generateImports(typeState.imports)

	generated += "\nvar " + groupName + " = map[string]" + typeDef + "{\n"
	for _, config := range configs {
		relConfigPath, err := filepath.Rel(filepath.Dir(config.goGenSrcPath), config.configPath)
		if err != nil {
			return err
		}
		generated += fmt.Sprintf("  %q: %v.%v,\n", relConfigPath, config.generatedPackageShortName, config.generatedVar)
	}
	generated += "\n}"

	log.Printf("Generating general config at %s for packages %v",
		Green(goPath),
		Yellow(sliceOfKeys(generatedPackages)))
	if err := os.WriteFile(goPath, []byte(generated), 0755); err != nil {
		return err
	}

	return goFmt(goPath)
}

func sliceOfKeys(m map[string]struct{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func generateImports(imports map[string]struct{}) string {
	res := ""
	for i := range imports {
		if strings.HasPrefix(i, "go:") {
			i = strings.TrimPrefix(i, "go:")
		} else if !strings.HasPrefix(i, "octopus/") {
			i = "octopus/" + i
		}
		res += fmt.Sprintf("import %q\n", i)
	}
	return res
}
