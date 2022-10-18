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

package fields

import (
	"fmt"
	"octopus/build-tools/gremlin/internal/generators/golang/core"
	"octopus/build-tools/gremlin/internal/types"
)

type GoEntitiesProvider interface {
	AddImport(path string, alias string)
	FindEnum(enumType *types.EnumDefinition) core.GoType
	FindEnumInImports(file *types.ProtoFile, enumType *types.EnumDefinition) (string, core.GoType)
	FindStruct(msgType *types.MessageDefinition) core.GoType
	FindStructInImports(file *types.ProtoFile, msgType *types.MessageDefinition) (string, core.GoType)
}

func ResolveType(targetFile GoEntitiesProvider, field *types.MessageFieldDefinition) (core.GoFieldType, error) {
	if field.ScalarValueType != "" {
		return resolveScalarType(targetFile, field)
	} else if field.LocalEnumType != nil || field.ExternalEnumType != nil {
		return resolveEnumType(targetFile, field)
	} else if field.LocalMsgType != nil || field.ExternalMsgType != nil {
		return resolveMsgType(targetFile, field)
	}
	return nil, fmt.Errorf("unknown field type for %v", field.Name)
}

func resolveScalarType(targetFile GoEntitiesProvider, field *types.MessageFieldDefinition) (core.GoFieldType, error) {
	typeName := goBasicTypesMap[field.ScalarValueType]
	if typeName == "" {
		return nil, fmt.Errorf("unknown scalar type %q", field.ScalarValueType)
	}

	baseType := &goBasicValueType{
		Name:      typeName,
		ProtoType: field.ScalarValueType,
	}
	if field.Repeated {
		return &goRepeatedValueType{
			RepeatedType: baseType,
		}, nil
	} else if field.Map {
		basicKeyType, err := resolveMapKeyType(field)
		if err != nil {
			return nil, err
		}

		return &goMapValueType{
			KeyType:   basicKeyType,
			ValueType: baseType,
		}, nil
	} else {
		if err := baseType.parseDefaultValue(targetFile, field); err != nil {
			return nil, err
		}
		return baseType, nil
	}
}

func resolveMapKeyType(field *types.MessageFieldDefinition) (core.GoFieldType, error) {
	keyType := goBasicTypesMap[field.MapKeyType]
	if keyType == "" {
		return nil, fmt.Errorf("unknown map key type %q", field.MapKeyType)
	}

	basicKeyType := &goBasicValueType{
		Name:      keyType,
		ProtoType: field.MapKeyType,
	}
	return basicKeyType, nil
}

func resolveEnumType(targetFile GoEntitiesProvider, field *types.MessageFieldDefinition) (core.GoFieldType, error) {
	var enumPackage string
	var enumType core.GoType
	if field.ExternalTypeFile != nil {
		enumPackage, enumType = targetFile.FindEnumInImports(field.ExternalTypeFile, field.ExternalEnumType)
		if enumPackage != "" {
			enumPackage = enumPackage + "."
		}
	} else {
		enumType = targetFile.FindEnum(field.LocalEnumType)
	}

	if enumType == nil {
		return nil, fmt.Errorf("unknown enum type %q for field %v", field.ProtoDef.Type, field.ProtoDef.Name)
	}

	valueType := &goEnumValueType{
		EnumName: enumPackage + enumType.GetName(),
	}

	if field.Repeated {
		return &goRepeatedValueType{
			RepeatedType: valueType,
		}, nil
	} else if field.Map {
		basicKeyType, err := resolveMapKeyType(field)
		if err != nil {
			return nil, err
		}
		return &goMapValueType{
			KeyType:   basicKeyType,
			ValueType: valueType,
		}, nil
	} else {
		valueType.resolveDefaultValue(field, enumPackage, enumType)
		return valueType, nil
	}
}

func resolveMsgType(targetFile GoEntitiesProvider, field *types.MessageFieldDefinition) (core.GoFieldType, error) {
	var msgPackage string
	var msgType core.GoType
	if field.ExternalTypeFile != nil {
		msgPackage, msgType = targetFile.FindStructInImports(field.ExternalTypeFile, field.ExternalMsgType)
	} else {
		msgType = targetFile.FindStruct(field.LocalMsgType)
	}

	if msgType == nil {
		return nil, fmt.Errorf("unknown message type %q", field.ProtoDef.Type)
	}

	valueType := &goStructValueType{
		StructPackage: msgPackage,
		StructName:    msgType.GetName(),
	}

	if field.Repeated {
		return &goRepeatedValueType{
			RepeatedType: valueType,
		}, nil
	} else if field.Map {
		basicKeyType, err := resolveMapKeyType(field)
		if err != nil {
			return nil, err
		}
		return &goMapValueType{
			KeyType:   basicKeyType,
			ValueType: valueType,
		}, nil
	} else {
		return valueType, nil
	}
}
