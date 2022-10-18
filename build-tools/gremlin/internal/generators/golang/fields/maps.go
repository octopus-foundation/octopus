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

type goMapValueType struct {
	KeyType   core.GoFieldType
	ValueType core.GoFieldType
}

func (t *goMapValueType) ReaderTypeName() string {
	return fmt.Sprintf("map[%v]%v", t.KeyType.ReaderTypeName(), t.ValueType.ReaderTypeName())
}

func (t *goMapValueType) WriterTypeName() string {
	return fmt.Sprintf("map[%v]%v", t.KeyType.WriterTypeName(), t.ValueType.WriterTypeName())
}

func (t *goMapValueType) OffsetsType() string {
	return "[]int"
}

func (t *goMapValueType) WireTypeType() string {
	return ""
}

func (t *goMapValueType) CanBePacked() bool {
	return false
}

func (t *goMapValueType) EntryUnmarshalSaveOffsets(tabs string, fieldName string) string {
	return formatting.AddTabs(
		fmt.Sprintf(`m.offset%v = append(m.offset%v, offset)`, fieldName, fieldName),
		tabs,
	)
}

func (t *goMapValueType) EntryReader(tabs string, localVarName string) string {
	res := fmt.Sprintf(`
var %v = map[%v]%v{}
for i := range wOffset {
	wOffset := wOffset[i]

	entrySize, entrySizeSize := m.buf.SizedReadVarInt(wOffset)
	endOffset := wOffset + entrySizeSize + int(entrySize)
	wOffset += entrySizeSize
	
	var keyData %v
	var valueData %v
	for wOffset < endOffset {
		tag, wireType, tagSize, _ := m.buf.ReadTagAt(wOffset)
		wOffset += tagSize
		if tag == 1 {
%v
			wOffset += keyEntrySize
			keyData = keyEntry
		} else if tag == 2 {
%v
			wOffset += valueEntrySize
			valueData = valueEntry
		} else {
			wOffset, _ = m.buf.SkipData(wOffset, wireType)
		}
	}
	%v[keyData] = valueData
}
`, localVarName, t.KeyType.ReaderTypeName(), t.ValueType.ReaderTypeName(),
		t.KeyType.ReaderTypeName(), t.ValueType.ReaderTypeName(),
		t.KeyType.EntrySizedReader("\t\t\t", "keyEntry"),
		t.ValueType.EntrySizedReader("\t\t\t", "valueEntry"),
		localVarName)

	return formatting.AddTabs(res, tabs)
}

func (t *goMapValueType) EntrySizedReader(string, string) string {
	return ""
}

func (t *goMapValueType) ToStruct(tabs string, targetVar string, readerField string) string {
	res := fmt.Sprintf(`if len(%v) > 0 {
	%v = make(%v, len(%v))
	for k,v := range %v {
%v
	}
}`,
		readerField, targetVar, t.WriterTypeName(), readerField,
		readerField,
		t.ValueType.ToStruct("\t\t", targetVar+"[k]", "v"),
	)
	return formatting.AddTabs(res, tabs)
}

func (t *goMapValueType) EntryIsNotEmpty(localVarName string) string {
	return fmt.Sprintf(`len(%v) > 0`, localVarName)
}

func (t *goMapValueType) EntryWriter(tabs string, targetBuffer string, tag string, varName string) string {
	return formatting.AddTabs(fmt.Sprintf(`for k, v := range %v {
	var mapEntry = gremlin.NewLazyBuffer(nil)
%v
%v
	%v.AppendBytes(%v, mapEntry.Bytes())
}`,
		varName,
		t.KeyType.EntryWriter("\t", "mapEntry", "1", "k"),
		t.ValueType.EntryWriter("\t", "mapEntry", "2", "v"),
		targetBuffer, tag,
	), tabs)
}

func (t *goMapValueType) PackedEntryWriter(string, string, string) string {
	log.Panicf("PackedEntryWriter should not be called on a map value type")
	return ""
}

func (t *goMapValueType) DefaultReturn() string {
	return "nil"
}

func (t *goMapValueType) JsonStructCanBeUsedDirectly() bool {
	return t.KeyType.JsonStructCanBeUsedDirectly() && t.ValueType.JsonStructCanBeUsedDirectly()
}

func (t *goMapValueType) EntryCopy(tabs string, targetVar string, srcVar string) string {
	return formatting.AddTabs(fmt.Sprintf(`%v = make(map[%v]%v, len(%v))
for k, v := range %v {
%v
}`, targetVar, t.KeyType.WriterTypeName(), t.ValueType.WriterTypeName(), srcVar, srcVar, t.ValueType.EntryCopy("\t", targetVar+"[k]", "v")), tabs)
}
