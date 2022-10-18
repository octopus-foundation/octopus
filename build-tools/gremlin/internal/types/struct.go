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

package types

import "github.com/emicklei/proto"

type TargetPlatform string

const (
	TargetPlatform_Go TargetPlatform = "go"
)

var ProtobufScalarTypes = map[string]struct{}{
	"string":   {},
	"bytes":    {},
	"bool":     {},
	"double":   {},
	"float":    {},
	"int32":    {},
	"int64":    {},
	"uint32":   {},
	"uint64":   {},
	"sint32":   {},
	"sint64":   {},
	"fixed32":  {},
	"fixed64":  {},
	"sfixed32": {},
	"sfixed64": {},
}

type EnumDefinition struct {
	Name ScopedName

	Values []*EnumValueDefinition

	ProtoDef *proto.Enum
}

type EnumValueDefinition struct {
	Name ScopedName

	Value int

	ProtoDef *proto.EnumField

	Options map[string]*EnumValueOptionDefinition
}

type EnumValueOptionDefinition struct {
	Name string

	Repeated bool

	ScalarValueType string
	LocalEnumType   *EnumDefinition

	ProtoDef *proto.Option

	Values []string
	Value  string
}

type MessageDefinition struct {
	Name ScopedName

	Parent *MessageDefinition
	Fields []*MessageFieldDefinition

	ProtoDef *proto.Message
}

type MessageFieldDefinition struct {
	Name       ScopedName
	Repeated   bool
	Map        bool
	OneOfGroup string
	MapKeyType string

	ScalarValueType string

	ExternalTypeFile *ProtoFile
	ExternalEnumType *EnumDefinition
	ExternalMsgType  *MessageDefinition

	LocalEnumType *EnumDefinition
	LocalMsgType  *MessageDefinition

	ProtoDef *proto.Field
	// because of extensions - we should look first from out type and after - of parent
	ExtraScopes []ScopedName

	DefaultValue *proto.Option
}

func (m *MessageFieldDefinition) Copy() *MessageFieldDefinition {
	res := &MessageFieldDefinition{
		Name:             m.Name,
		Repeated:         m.Repeated,
		Map:              m.Map,
		OneOfGroup:       m.OneOfGroup,
		MapKeyType:       m.MapKeyType,
		ScalarValueType:  m.ScalarValueType,
		ExternalTypeFile: m.ExternalTypeFile,
		ExternalEnumType: m.ExternalEnumType,
		ExternalMsgType:  m.ExternalMsgType,
		LocalEnumType:    m.LocalEnumType,
		LocalMsgType:     m.LocalMsgType,
		ProtoDef:         m.ProtoDef,
		DefaultValue:     m.DefaultValue,
	}

	if m.ExtraScopes != nil {
		res.ExtraScopes = make([]ScopedName, len(m.ExtraScopes))
		copy(res.ExtraScopes, m.ExtraScopes)
	}

	return res
}
