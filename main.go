package main

import (
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"google.golang.org/protobuf/compiler/protogen"
)

var abiABI = protogen.GoIdent{
	GoName:       "ABI",
	GoImportPath: "github.com/ethereum/go-ethereum/accounts/abi",
}

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

	// Generate a type that defines the expected way in which contract addresses for
	// a given struct will be provided to the generated code
	g.P("type ", s.Name, "AddressProvider interface {")
	for _, contract := range s.contracts {
		g.P(contract, "Address() (*common.Address, error)")
	}
	g.P("}")

	g.P()

	// Generate a type that serves as a writer for all the contract dependencies
	g.P("type ", s.Name, "Writer struct {")
	g.P()
	for _, contract := range s.contracts {
		g.P(firstToLower(contract), "ABI *", abiABI)
	}

	g.P("}")

	g.P()

	// Generate a type that serves as a caller for all the contract dependencies
	g.P("type Bound", s.Name, "Writer struct {")
	g.P("	*", s.Name, "Writer")
	g.P()
	for _, contract := range s.contracts {
		g.P(firstToLower(contract), " *", abiPrefix, contract)
	}

	g.P("}")

	g.P()

	// Generate a type that serves as a raw caller for all the contract dependencies
	g.P("type Raw", s.Name, "Writer struct {")
	g.P("	*", s.Name, "Writer")
	g.P()
	for _, contract := range s.contracts {
		g.P(firstToLower(contract), "Address *common.Address")
	}

	g.P("}")

	g.P()

	g.P("func New", s.Name, "Writer() (*", s.Name, "Writer, error) {")
	g.P("   var err error")
	g.P("	out := &", s.Name, "Writer{}")
	for _, contract := range s.contracts {
		g.P("out.", firstToLower(contract), "ABI, err = ", abiPrefix, contract, "MetaData.GetAbi()")
		g.P("if err != nil { return nil, ", errorf, "(\"failed to parse contract ", contract, " abi: %v\", err) }")
	}
	g.P("	return out, nil")
	g.P("}")

	g.P()

	g.P("func (w *", s.Name, "Writer) Bind(backend bind.ContractBackend, addressProvider ", s.Name, "AddressProvider) (*Bound", s.Name, "Writer, error) {")
	g.P("   var err error")
	g.P("   var address *common.Address")
	g.P("	out := &Bound", s.Name, "Writer{")
	g.P("		", s.Name, "Writer: w,")
	g.P("	}")
	for _, contract := range s.contracts {
		// Get the address
		g.P("address, err = addressProvider.", contract, "Address()")
		g.P("if err != nil { return nil, ", errorf, "(\"error getting contract ", contract, " address: %v\", err) }")
		g.P("out.", firstToLower(contract), ", err = ", abiPrefix, "New", contract, "(*address, backend)")
		g.P("if err != nil { return nil, ", errorf, "(\"failed to bind contract ", contract, " abi: %v\", err) }")
		g.P()
	}
	g.P("	return out, nil")
	g.P("}")

	g.P()

	g.P("func (w *", s.Name, "Writer) Raw(addressProvider ", s.Name, "AddressProvider) (*Raw", s.Name, "Writer, error) {")
	g.P("   var err error")
	g.P("	out := &Raw", s.Name, "Writer{")
	g.P("		", s.Name, "Writer: w,")
	g.P("	}")
	for _, contract := range s.contracts {
		// Get the address
		g.P("out.", firstToLower(contract), "Address, err = addressProvider.", contract, "Address()")
		g.P("if err != nil { return nil, ", errorf, "(\"error getting contract ", contract, " address: %v\", err) }")
		g.P()
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
		g.P("func (c *Raw", s.Name, "Writer) ", field.Name, "(dst *", s.Name, ") *", call, " {")
		g.P("	out := new(", call, ")")
		g.P("	out.Abi = c.", s.Name, "Writer.", firstToLower(field.Contract), "ABI")
		g.P("	out.Address = c.", firstToLower(field.Contract), "Address")
		g.P("	out.CallData = func() ([]byte, error) { return out.Abi.Pack(\"", field.Selector.Name, "\")}")
		g.P("	out.Method = \"", field.Selector.Name, "\"")
		g.P("	out.Destination = &dst.", field.Name)
		g.P("	return out")
		g.P("}")
		g.P()
	}

	// Generate a function which accepts a bind.CallOpts, and produces all the calls
	g.P("func (c *Raw", s.Name, "Writer) AllCalls (dst *", s.Name, ") []*", call, " {")
	if len(s.Fields) > 0 {
		g.P("var call *", call)
	}
	g.P("	out := make([]*", call, ",0,", len(s.Fields), ")")
	g.P()

	for _, field := range s.Fields {
		g.P("call = c.", field.Name, "(dst)")
		g.P("out = append(out, call)")
	}
	g.P("	return out")
	g.P("}")

	return nil
}

func generatePopulate(g *protogen.GeneratedFile, s *Struct) error {

	// Generate functions for each field
	for _, field := range s.Fields {
		g.P("func (c *", s.Name, "Writer) Populate", field.Name, "(dst *", s.Name, ", backend bind.ContractBackend, addressProvider ", s.Name, "AddressProvider, opts *", g.QualifiedGoIdent(callOpts), ") error {")
		g.P("	var err error")
		g.P("	address, err := addressProvider.", field.Contract, "Address()")
		g.P("	if err != nil { return ", errorf, "(\"error getting contract ", field.Contract, " address: %v\", err) }")
		g.P("	bound, err := New", field.Contract, "(*address, backend)")
		g.P("	if err != nil { return ", errorf, "(\"error binding contract ", field.Contract, "\") }")
		g.P("	dst.", field.Name, ", err = bound.", abi.ToCamelCase(field.Selector.Name), "(opts)")
		g.P("	return err")
		g.P("}")
		g.P()
	}

	for _, field := range s.Fields {
		g.P("func (c *Bound", s.Name, "Writer) Populate", field.Name, "(dst *", s.Name, ", opts *", g.QualifiedGoIdent(callOpts), ") error {")
		g.P("	var err error")
		g.P("	dst.", field.Name, ", err = c.", firstToLower(field.Contract), ".", abi.ToCamelCase(field.Selector.Name), "(opts)")
		g.P("	return err")
		g.P("}")
		g.P()
	}

	// Generate a function which accepts an eth client and bind.CallOpts, and produces the message
	g.P("func (c *", s.Name, "Writer) Populate (dst *", s.Name, ", backend bind.ContractBackend, addressProvider ", s.Name, "AddressProvider, opts *", g.QualifiedGoIdent(callOpts), ") error {")
	if len(s.Fields) > 0 {
		g.P("var err error")
	}

	// First, create a temporary binding
	g.P("	bound, err := c.Bind(backend, addressProvider)")
	g.P("	if err != nil { return ", errorf, "(\"failed to bind ", s.Name, ": %v\", err) }")
	g.P("	bound.Populate(dst, opts)")
	g.P("	return nil")
	g.P("}")

	// Generate a function which accepts a bind.CallOpts, and produces the message
	g.P("func (c *Bound", s.Name, "Writer) Populate (dst *", s.Name, ", opts *", g.QualifiedGoIdent(callOpts), ") error {")
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

	g.P()

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
