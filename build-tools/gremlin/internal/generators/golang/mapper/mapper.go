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

package mapper

import (
	"fmt"
	"octopus/build-tools/gremlin/internal/generators/golang/core"
	"octopus/build-tools/gremlin/internal/generators/golang/fields"
	gotypes "octopus/build-tools/gremlin/internal/generators/golang/types"
	"octopus/build-tools/gremlin/internal/types"
	"path/filepath"
	"strings"
)

func MapProtoFiles(root string, files []*types.ProtoFile) ([]*core.GoGeneratedFile, []error) {
	var result = make([]*core.GoGeneratedFile, len(files))
	var errors []error

	for i, target := range files {
		shortName, fullName := getGoPackage(root, target)
		if shortName == "" || fullName == "" {
			errors = append(errors, fmt.Errorf("unable to get package for file %v", target.RelativePath))
			continue
		}

		goFile := &core.GoGeneratedFile{
			ProtoFile:        files[i],
			FullPackageName:  fullName,
			ShortPackageName: shortName,
		}
		result[i] = goFile
	}

	errors = append(errors, mapImports(result)...)
	if len(errors) > 0 {
		return nil, errors
	}

	for i, target := range result {
		result[i].FullOutputPath = buildOutputPath(root, target)
	}

	// now we have package names, imports and aliases for imports
	var structs [][]*gotypes.GoStructType
	for i := range result {
		goFile := result[i]
		var fileStructs []*gotypes.GoStructType
		for j := range goFile.ProtoFile.Enums {
			enumDef := goFile.ProtoFile.Enums[j]
			goFile.AddEnum(gotypes.NewEnumType(enumDef))
		}
		for j := range goFile.ProtoFile.Messages {
			messageDef := goFile.ProtoFile.Messages[j]
			goStruct := gotypes.NewStructType(messageDef)
			goFile.AddStruct(goStruct)
			fileStructs = append(fileStructs, goStruct)
		}
		structs = append(structs, fileStructs)
	}

	// last step - map fields
	for i := range result {
		goFile := result[i]
		for j := range goFile.ProtoFile.Messages {
			goMessageDef := structs[i][j]
			for k := range goFile.ProtoFile.Messages[j].Fields {
				fieldDef := goFile.ProtoFile.Messages[j].Fields[k]
				fieldType, err := fields.ResolveType(goFile, fieldDef)
				if err != nil {
					errors = append(errors, fmt.Errorf("file: %v, %w", goFile.ProtoFile.RelativePath, err))
					continue
				}

				goMessageDef.AddField(fieldDef, fieldType)
			}
		}
	}

	return result, errors
}

func getGoPackage(outBase string, target *types.ProtoFile) (string, string) {
	if target.Package == nil || target.Package.Name.PlatformName(types.TargetPlatform_Go) == "" {
		var pkgName = filepath.Dir(target.RelativePath)
		var shortName = filepath.Base(pkgName)
		if target.Package != nil && target.Package.Name.ProtoName() != "" {
			pkgName = filepath.Join(pkgName, target.Package.Name.ProtoName())
			shortName = target.Package.Name.ProtoName()
		}
		pkgName = strings.TrimPrefix(pkgName, outBase)
		pkgName = filepath.Join(filepath.Base(outBase), types.TargetFolder, pkgName)

		return cleanupPackageName(shortName), pkgName
	} else {
		var pkg = target.Package.Name.PlatformName(types.TargetPlatform_Go)
		var shortName = filepath.Base(pkg)
		// actually - we should remove this for open-source
		if !strings.HasPrefix(pkg, "octopus/target/generated-sources/protobuf") && strings.HasPrefix(pkg, "octopus/target/generated-sources") {
			pkgName := strings.TrimPrefix(pkg, "octopus/target/generated-sources")
			pkgName = filepath.Join("octopus/target/generated-sources/protobuf", pkgName)
			return cleanupPackageName(shortName), pkgName
		}
	}
	return "", ""
}

func cleanupPackageName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}

func buildOutputPath(root string, pFile *core.GoGeneratedFile) string {
	fName := filepath.Base(pFile.ProtoFile.RelativePath)
	fName = strings.TrimSuffix(fName, types.ProtoExtension)
	fName = fmt.Sprintf("%v%v", fName, types.PbGoExtension)

	res := strings.TrimPrefix(pFile.FullPackageName, filepath.Base(root))
	res = filepath.Join(root, res, fName)

	return res
}
