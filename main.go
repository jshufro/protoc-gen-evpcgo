package main

import (
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/protobuf/compiler/protogen"
)

var ethclient = protogen.GoIdent{
	GoName:       "Client",
	GoImportPath: "github.com/ethereum/go-ethereum/ethclient",
}

var callOpts = protogen.GoIdent{
	GoName:       "CallOpts",
	GoImportPath: "github.com/ethereum/go-ethereum/accounts/abi/bind",
}

var errorf = protogen.GoIdent{
	GoName:       "Errorf",
	GoImportPath: "fmt",
}

var call = protogen.GoIdent{
	GoName:       "Call",
	GoImportPath: "github.com/jshufro/protoc-gen-evpcgo/lib",
}

var customTypes = map[string]protogen.GoIdent{
	"common.Address": protogen.GoIdent{
		GoName:       "Address",
		GoImportPath: "github.com/ethereum/go-ethereum/common",
	},
}

func generateTypes(g *protogen.GeneratedFile, s *Struct, abiPrefix string) error {

	// Generate a type with our native golang field types
	g.P("type ", s.Name, " struct {")
	for _, f := range s.Fields {
		// Create the field
		if ct, ok := customTypes[f.Type]; ok {
			g.P(f.Name, " ", ct)
		} else {
			g.P(f.Name, " ", f.Type)
		}
	}
	g.P("}")

	g.P()

	// Generate a type that stores details (mainly, addresses) for the contract dependencies
	g.P("type ", s.Name, "_Details struct {")
	for _, contract := range s.contracts {
		g.P(contract, "_Address common.Address")
	}
	g.P("}")

	g.P()

	// Generate a type that serves as a caller for all the contract dependencies
	g.P("type Bound_", s.Name, " struct {")
	for _, contract := range s.contracts {
		g.P("*", abiPrefix, contract)
	}

	g.P("	details *", s.Name, "_Details")
	g.P("}")

	g.P()

	// Generate a function to create the caller
	g.P("func New", s.Name, "_Details(")
	for _, contract := range s.contracts {
		g.P(contract, "_Address common.Address,")
	}
	g.P(") (*", s.Name, "_Details) {")
	g.P("	out := &", s.Name, "_Details{}")
	for _, contract := range s.contracts {
		g.P("out.", contract, "_Address = ", contract, "_Address")
	}
	g.P("	return out")
	g.P("}")

	g.P()

	g.P("func (d *", s.Name, "_Details) Bind(backend bind.ContractBackend) (*Bound_", s.Name, ", error) {")
	g.P("   var err error")
	g.P("	out := &Bound_", s.Name, "{")
	g.P("		details: d,")
	g.P("}")
	for _, contract := range s.contracts {
		g.P("out.", contract, ", err = ", abiPrefix, "New", contract, "(d.", contract, "_Address, backend)")
		g.P("if err != nil { return nil, ", errorf, "(\"failed to bind contract ", contract, " to address %s: %v\", d.", contract, "_Address, err) }")
	}
	g.P("	return out, nil")
	g.P("}")

	g.P()

	return nil
}

func importAbi(g *protogen.GeneratedFile, spec *File) (string, error) {
	if spec.AbiPackage == "" {
		return "", nil
	}
	g.P("import _abi \"", spec.AbiPackage, "\"")
	return "_abi.", nil
}

type fields struct {
	fieldMap map[string]string // Map of field names to the contract namess whence their data comes
}

func generateRaw(g *protogen.GeneratedFile, s *Struct) error {

	// Generate functions for each field
	for _, field := range s.Fields {
		g.P("func (c *Bound_", s.Name, ") Raw", field.Name, "(dst *", s.Name, ") (*", call, ", error) {")
		g.P("	var err error")
		g.P("	out := new(", call, ")")
		g.P("	out.Address = &c.details.", field.Contract, "_Address")
		g.P("	parsedAbi, err := ", field.Contract, "MetaData.GetAbi()")
		g.P("	if err != nil {")
		g.P("		return nil, fmt.Errorf(\"failed to parse ABI for ", field.Contract, ": %v\", err)")
		g.P("	}")
		g.P("	out.Abi = parsedAbi")
		g.P("	out.CallData, err = parsedAbi.Pack(\"", field.Selector.Name, "\")")
		g.P("	if err != nil {")
		g.P("		return nil, fmt.Errorf(\"failed to pack ABI for ", field.Contract, ": %v\", err)")
		g.P("	}")
		g.P("	out.Method = \"", field.Selector.Name, "\"")
		g.P("	out.Destination = &dst.", field.Name)
		g.P("	return out, err")
		g.P("}")
		g.P()
	}

	// Generate a function which accepts a bind.CallOpts, and produces all the calls
	g.P("func (c *Bound_", s.Name, ") Raw (dst *", s.Name, ") ([]*", call, ", error) {")
	if len(s.Fields) > 0 {
		g.P("var err error")
		g.P("var call *", call)
	}
	g.P("	out := make([]*", call, ",0,", len(s.Fields), ")")

	for _, field := range s.Fields {
		g.P("call, err = c.Raw", field.Name, "(dst)")
		g.P("if err != nil {")
		g.P("	return nil, ", errorf, "(\"failed to get raw data for field ", field.Name, ": %v\", err)")
		g.P("}")
		g.P("out = append(out, call)")
	}
	g.P("return out, nil")
	g.P("}")

	return nil
}

func generatePopulate(g *protogen.GeneratedFile, s *Struct) error {

	// Generate functions for each field
	for _, field := range s.Fields {
		g.P("func (c *Bound_", s.Name, ") Populate", field.Name, "(dst *", s.Name, ", opts *", g.QualifiedGoIdent(callOpts), ") error {")
		g.P("	var err error")
		g.P("	dst.", field.Name, ", err = c.", abi.ToCamelCase(field.Selector.Name), "(opts)")
		g.P("	return err")
		g.P("}")
		g.P()
	}

	// Generate a function which accepts an eth client and bind.CallOpts, and produces the message
	g.P("func (c *Bound_", s.Name, ") Populate (dst *", s.Name, ", opts *", g.QualifiedGoIdent(callOpts), ") error {")
	if len(s.Fields) > 0 {
		g.P("var err error")
	}

	for _, field := range s.Fields {
		g.P("err = c.Populate", field.Name, "(dst, opts)")
		g.P("if err != nil {")
		g.P("	return ", errorf, "(\"failed to populate field ", field.Name, ": %v\", err)")
		g.P("}")
	}
	g.P("return nil")
	g.P("}")

	return nil
}

func packageFromSpec(spec *File) string {
	s := strings.Split(spec.AbiPackage, "/")
	return s[len(s)-1]
}

func generateFile(p *protogen.Plugin, f *protogen.File, spec *File) error {
	if len(f.Messages) == 0 {
		return nil
	}

	filename := f.GeneratedFilenamePrefix + "_evpc.pb.go"
	// Replace the pb path with the abi path
	filename = strings.Replace(filename, string(f.GoPackageName), spec.AbiPackage, 1)
	g := p.NewGeneratedFile("abi/storage_evpc.pb.go", protogen.GoImportPath(spec.AbiPackage))
	g.P("// Code generated by protoc-gen-evpcgo. DO NOT EDIT.")
	g.P()
	g.P("package ", packageFromSpec(spec))
	g.P()
	/*abiPrefix, err := importAbi(g, spec)
	if err != nil {
		return err
	}*/
	abiPrefix := ""

	for _, s := range spec.Structs {
		err := generateTypes(g, s, abiPrefix)
		if err != nil {
			return err
		}
		err = generatePopulate(g, s)
		if err != nil {
			return err
		}
		err = generateRaw(g, s)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	log.Println("generating evpcgo")
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		for _, file := range plugin.Files {
			if !file.Generate {
				continue
			}

			spec, err := parseProto(plugin, file)
			if err != nil {
				return err
			}

			if err := generateFile(plugin, file, spec); err != nil {
				return err
			}
		}

		return nil
	})
}
