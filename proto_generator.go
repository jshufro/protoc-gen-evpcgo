package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/jshufro/protoc-gen-evpcgo/test/pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func parseProtoMessageField(p *protogen.Plugin, f *protogen.File, m *protogen.Message, field *protogen.Field) (*Field, error) {
	out := new(Field)
	out.Name = field.GoName

	options := field.Desc.Options().(*descriptorpb.FieldOptions)
	binding := proto.GetExtension(options, pb.E_Binding).(*pb.Binding)

	out.Contract = binding.Contract
	selector, err := abi.ParseSelector(binding.Selector)
	if err != nil {
		return nil, err
	}

	out.Selector = &selector
	if binding.GoType != "" {
		out.Type = binding.GoType
	} else {
		out.Type = field.Desc.Kind().String()
	}

	return out, nil
}

func parseProtoMessage(p *protogen.Plugin, f *protogen.File, m *protogen.Message) (*Struct, error) {
	out := new(Struct)

	contractMap := make(map[string]interface{})

	// Get top-level settings
	{
		normalized := strings.TrimSuffix(m.GoIdent.GoName, "Message")
		if normalized == m.GoIdent.GoName {
			return nil, fmt.Errorf("error generating %s, evpc messages must have Message suffix... rename to %s", m.GoIdent.GoName, fmt.Sprintf("%sMessage", m.GoIdent.GoName))
		}
		out.Name = normalized
	}

	// Parse individual fields
	{
		out.Fields = make([]*Field, 0, len(m.Fields))
		for _, field := range m.Fields {
			parsed, err := parseProtoMessageField(p, f, m, field)
			if err != nil {
				return nil, err
			}
			out.Fields = append(out.Fields, parsed)
			contractMap[parsed.Contract] = struct{}{}
		}
	}

	// Sort the deduplicate contract map and add it
	out.contracts = make([]string, 0, len(contractMap))
	for k, _ := range contractMap {
		out.contracts = append(out.contracts, k)
	}
	sort.Strings(out.contracts)

	return out, nil
}

func parseProto(p *protogen.Plugin, f *protogen.File) (*File, error) {
	out := new(File)

	// Get top-level options
	{
		options := f.Desc.Options().(*descriptorpb.FileOptions)
		out.AbiPackage = proto.GetExtension(options, pb.E_AbiPackage).(string)
		out.Version = proto.GetExtension(options, pb.E_Version).(string)
	}

	// Parse individual messages
	{
		out.Structs = make([]*Struct, 0, len(f.Messages))
		for _, m := range f.Messages {
			parsed, err := parseProtoMessage(p, f, m)
			if err != nil {
				return nil, err
			}
			out.Structs = append(out.Structs, parsed)
		}
	}

	return out, nil
}
