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
	"log"
	"octopus/build-tools/gremlin/internal/formatting"
	"octopus/build-tools/gremlin/internal/generators/golang/core"
)

type goRepeatedValueType struct {
	RepeatedType core.GoFieldType
}

func (t *goRepeatedValueType) ReaderTypeName() string {
	return fmt.Sprintf("[]%v", t.RepeatedType.ReaderTypeName())
}

func (t *goRepeatedValueType) WriterTypeName() string {
	return fmt.Sprintf("[]%v", t.RepeatedType.WriterTypeName())
}

func (t *goRepeatedValueType) OffsetsType() string {
	return "[]int"
}

func (t *goRepeatedValueType) WireTypeType() string {
	if t.RepeatedType.CanBePacked() {
		return "[]gremlin.ProtoWireType"
	} else {
		return ""
	}
}

func (t *goRepeatedValueType) CanBePacked() bool {
	return false
}

func (t *goRepeatedValueType) EntrySizedReader(string, string) string {
	return ""
}

func (t *goRepeatedValueType) EntryUnmarshalSaveOffsets(tabs string, fieldName string) string {
	var res string
	if t.RepeatedType.CanBePacked() {
		res = fmt.Sprintf(`m.offset%v = append(m.offset%v, offset)
m.wireType%v = append(m.wireType%v, wire)`, fieldName, fieldName, fieldName, fieldName)
	} else {
		res = fmt.Sprintf(`m.offset%v = append(m.offset%v, offset)`, fieldName, fieldName)
	}
	return formatting.AddTabs(res, tabs)
}

func (t *goRepeatedValueType) EntryReader(tabs string, localVarName string) string {
	var res string
	if t.RepeatedType.CanBePacked() {
		res = fmt.Sprintf(`
var %v %v
for i := 0; i < len(wOffset); i++ {
	wOffset := wOffset[i]
	wType := wType[i]
	if wType == gremlin.BytesType {
		size, sizeSize := m.buf.SizedReadVarInt(wOffset)
		offset := 0
		for offset < int(size) {
			wOffset := wOffset + sizeSize + offset
%v
			%v = append(%v, listEntry)
			offset += listEntrySize
		}
	} else {
%v
		%v = append(%v, listEntry)
	}
}
`, localVarName, t.ReaderTypeName(),
			t.RepeatedType.EntrySizedReader("\t\t\t", "listEntry"),
			localVarName, localVarName,
			t.RepeatedType.EntryReader("\t\t", "listEntry"),
			localVarName, localVarName)
	} else {
		res = fmt.Sprintf(`
var %v %v
for i := 0; i < len(wOffset); i++ {
	wOffset := wOffset[i]
%v
	%v = append(%v, listEntry)
}
`, localVarName, t.ReaderTypeName(),
			t.RepeatedType.EntryReader("\t", "listEntry"),
			localVarName, localVarName)
	}

	return formatting.AddTabs(res, tabs)
}

func (t *goRepeatedValueType) ToStruct(tabs string, targetVar string, readerField string) string {
	res := fmt.Sprintf(`if len(%v) > 0 {
	%v = make(%v, len(%v))
	for i := range %v {
%v
	}
}`, readerField, targetVar, t.WriterTypeName(), readerField, readerField,
		t.RepeatedType.ToStruct("\t\t", targetVar+"[i]", readerField+"[i]"),
	)

	return formatting.AddTabs(res, tabs)
}

func (t *goRepeatedValueType) EntryIsNotEmpty(localVarName string) string {
	return fmt.Sprintf(`len(%v) > 0`, localVarName)
}

func (t *goRepeatedValueType) EntryWriter(tabs string, targetBuffer string, tag string, varName string) string {
	if t.RepeatedType.CanBePacked() {
		return formatting.AddTabs(t.packedEntryWriter(targetBuffer, tag, varName), tabs)
	} else {
		return formatting.AddTabs(fmt.Sprintf(`for _, entry := range %v {
%v
}`, varName, t.RepeatedType.EntryWriter("\t", targetBuffer, tag, "entry")), tabs)
	}
}

func (t *goRepeatedValueType) packedEntryWriter(targetBuffer string, tag string, name string) string {
	return fmt.Sprintf(`if len(%v) > 1 {
	var packed = gremlin.NewLazyBuffer(nil)
	for _, entry := range %v {
%v
	}
	%v.AppendBytes(%v, packed.Bytes())
} else if len(%v) == 1 {
%v
}`, name, name, t.RepeatedType.PackedEntryWriter("\t\t", "packed", "entry"),
		targetBuffer, tag,
		name, t.RepeatedType.EntryWriter("\t", targetBuffer, tag, name+"[0]"))
}

func (t *goRepeatedValueType) PackedEntryWriter(string, string, string) string {
	log.Panicf("PackedEntryWriter should not be called on a repeated value type")
	return ""
}

func (t *goRepeatedValueType) DefaultReturn() string {
	return "nil"
}

func (t *goRepeatedValueType) JsonStructCanBeUsedDirectly() bool {
	return t.RepeatedType.JsonStructCanBeUsedDirectly()
}

func (t *goRepeatedValueType) EntryCopy(tabs string, targetVar string, srcVar string) string {
	if t.RepeatedType.JsonStructCanBeUsedDirectly() {
		return formatting.AddTabs(fmt.Sprintf(`%v = %v`, targetVar, srcVar), tabs)
	} else {
		return formatting.AddTabs(fmt.Sprintf(`%v = make(%v, len(%v))
for i := range %v {
%v
}`, targetVar, t.WriterTypeName(), srcVar, srcVar, t.RepeatedType.EntryCopy("\t", targetVar+"[i]", srcVar+"[i]")), tabs)
	}
}
