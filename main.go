package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/jshufro/protoc-gen-evpcgo/test/pb"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var ethclient = protogen.GoIdent{
	GoName:       "Client",
	GoImportPath: "github.com/ethereum/go-ethereum/ethclient",
}

var callOpts = protogen.GoIdent{
	GoName:       "CallOpts",
	GoImportPath: "github.com/ethereum/go-ethereum/accounts/abi/bind",
}

var customTypes = map[string]protogen.GoIdent{
	"common.Address": protogen.GoIdent{
		GoName:       "Address",
		GoImportPath: "github.com/ethereum/go-ethereum/common",
	},
}

type abiGen struct {
	f    string
	args []string
	ret  []string
}

func (a *abiGen) ContractName() string {
	return strings.Split(a.f, ".")[0]
}

func (a *abiGen) FieldType(g *protogen.GeneratedFile) string {
	if len(a.ret) == 0 {
		return "interface{}"
	}
	custom, ok := customTypes[a.ret[0]]
	if !ok {
		return a.ret[0]
	}

	return g.QualifiedGoIdent(custom)
}

// Parse the function name, arguments (if any), and returned values from an abigen binding
func parseAbigen(a string) (*abiGen, error) {
	out := abiGen{}
	endFunc := strings.Index(a, ")")
	if endFunc == -1 {
		return nil, fmt.Errorf("invalid binding abigen '%s', could not determine function name", a)
	}
	if strings.Count(a[:endFunc], "(") != 1 {
		return nil, fmt.Errorf("invalid binding abigen '%s', could not determine function name", a)
	}
	startArgs := strings.Index(a, "(")
	out.f = a[:startArgs]

	args := a[startArgs+1 : endFunc]
	out.args = strings.Split(args, ",")

	remainder := strings.TrimSpace(a[endFunc+1:])
	out.ret = strings.Split(strings.Trim(remainder, "()"), ",")

	return &out, nil
}

func parseNativeName(m *protogen.Message) (string, error) {
	native := strings.TrimSuffix(m.GoIdent.GoName, "Message")
	if native == m.GoIdent.GoName {
		return "", fmt.Errorf("evpc protobuf messages must end in 'Message', got: '%s'", m.GoIdent.GoName)
	}
	return native, nil
}

func generateTypes(g *protogen.GeneratedFile, m *protogen.Message) error {
	nativeName, err := parseNativeName(m)
	if err != nil {
		return err
	}

	// Map of field's GoName to the abigen
	fieldMap := make(map[string]*abiGen)

	for _, f := range m.Fields {
		o := f.Desc.Options().(*descriptorpb.FieldOptions)
		binding := proto.GetExtension(o, pb.E_Binding).(*pb.Binding)
		a, err := parseAbigen(binding.Abigen)
		if err != nil {
			return err
		}

		fieldMap[f.GoName] = a
	}

	// Generate a type with our native golang field types
	g.P("type ", nativeName, " struct {")
	for fieldName, a := range fieldMap {
		// Create the field
		g.P(fieldName, " ", a.FieldType(g))
	}
	g.P("}")

	g.P()

	// Generate a type that stores details (mainly, addresses) for the contract dependencies
	contractSet := make(map[string]interface{})
	g.P("type ", nativeName, "_Details struct {")
	for _, a := range fieldMap {
		contractSet[a.ContractName()] = struct{}{}
	}
	for contract, _ := range contractSet {
		g.P(contract, "_Address common.Address")
	}
	g.P("}")

	// Generate a type that serves as a caller for all the contract dependencies
	g.P("type Bound_", nativeName, " struct {")
	for _, a := range fieldMap {
		contractSet[a.ContractName()] = struct{}{}
	}
	for contract, _ := range contractSet {
		g.P("*abi.", contract)
	}
	g.P("}")

	g.P()

	// Generate a function to create the caller
	g.P("func New", nativeName, "_Details(")
	for contract, _ := range contractSet {
		g.P(contract, "_Address common.Address,")
	}
	g.P(") (*", nativeName, "_Details) {")
	g.P("	out := &", nativeName, "_Details{}")
	for contract, _ := range contractSet {
		g.P("out.", contract, "_Address = ", contract, "_Address")
	}
	g.P("	return out")
	g.P("}")

	g.P()

	g.P("func (d *", nativeName, "_Details) Bind(backend bind.ContractBackend) (*Bound_", nativeName, ", error) {")
	g.P("   var err error")
	g.P("	out := &Bound_", nativeName, "{}")
	for contract, _ := range contractSet {
		g.P("out.", contract, ", err = abi.New", contract, "(d.", contract, "_Address, backend)")
		g.P("if err != nil { return nil, err }")
	}
	g.P("	return out, nil")
	g.P("}")

	g.P()

	return nil
}

func importAbi(g *protogen.GeneratedFile, f *protogen.File) (string, error) {
	o := f.Desc.Options().(*descriptorpb.FileOptions)
	abi_package := proto.GetExtension(o, pb.E_AbiPackage).(string)
	if abi_package == "" {
		return "", fmt.Errorf("abi_package must be defined")
	}

	g.P("import \"", abi_package, "\"")
	dirs := strings.Split(abi_package, "/")
	return dirs[len(dirs)-1], nil
}

type fields struct {
	fieldMap        map[string]string      // Map of field names to the contract namess whence their data comes
	contractGetters map[string]interface{} // Map of deduplicated contract names
}

func populateParseFields(fs []*protogen.Field) (*fields, error) {
	out := &fields{}
	out.fieldMap = make(map[string]string)
	out.contractGetters = make(map[string]interface{})

	// Grab the bound abi
	for _, f := range fs {
		fieldName := f.GoName
		o := f.Desc.Options().(*descriptorpb.FieldOptions)
		binding := proto.GetExtension(o, pb.E_Binding).(*pb.Binding)
		a, err := parseAbigen(binding.Abigen)
		if err != nil {
			return nil, err
		}

		contractName := strings.Split(a.f, ".")[0]
		out.fieldMap[fieldName] = a.f
		out.contractGetters[fmt.Sprintf("contract%s, err := Get%sAbi(client, contractAddrMap[\"%s\"])", contractName, contractName, contractName)] = struct{}{}
	}

	return out, nil
}

func generatePopulate(g *protogen.GeneratedFile, m *protogen.Message) error {
	nativeName, err := parseNativeName(m)
	if err != nil {
		return err
	}

	fields, err := populateParseFields(m.Fields)
	if err != nil {
		return err
	}

	// Generate a function which accepts an eth client and bind.CallOpts, and produces the message
	g.P("func (c *Bound_", nativeName, ") Populate (dst *", nativeName, ", opts *", g.QualifiedGoIdent(callOpts), ") error {")
	if len(m.Fields) > 0 {
		g.P("var err error")
	}

	for k, v := range fields.fieldMap {
		g.P("dst.", k, ", err = c.", strings.Split(v, ".")[1], "(opts)")
		g.P("if err != nil {")
		g.P("	return err")
		g.P("}")
	}
	g.P("return nil")
	g.P("}")

	return nil
}

func generateFile(p *protogen.Plugin, f *protogen.File) error {
	if len(f.Messages) == 0 {
		return nil
	}

	filename := f.GeneratedFilenamePrefix + "_evpc.pb.go"
	g := p.NewGeneratedFile(filename, f.GoImportPath)
	g.P("// Code generated by protoc-gen-evpcgo. DO NOT EDIT.")
	g.P()
	g.P("package ", f.GoPackageName)
	g.P()
	_, err := importAbi(g, f)
	if err != nil {
		return err
	}

	for _, m := range f.Messages {
		err := generateTypes(g, m)
		if err != nil {
			return err
		}
		err = generatePopulate(g, m)
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

			if err := generateFile(plugin, file); err != nil {
				return err
			}
		}

		return nil
	})
}
