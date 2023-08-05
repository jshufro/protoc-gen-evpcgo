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

func generateType(g *protogen.GeneratedFile, m *protogen.Message) error {
	nativeName, err := parseNativeName(m)
	if err != nil {
		return err
	}

	// Generate a type with our native golang field types
	g.P("type ", nativeName, " struct {")
	for _, f := range m.Fields {
		o := f.Desc.Options().(*descriptorpb.FieldOptions)
		binding := proto.GetExtension(o, pb.E_Binding).(*pb.Binding)
		a, err := parseAbigen(binding.Abigen)
		if err != nil {
			return err
		}

		// Create the field
		g.P(f.GoName, " ", a.FieldType(g))
	}
	g.P("}")
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

func generateContractRegistry(g *protogen.GeneratedFile, f *protogen.File, abiPath string) error {
	o := f.Desc.Options().(*descriptorpb.FileOptions)
	contracts := proto.GetExtension(o, pb.E_Contracts).(*pb.Contracts)
	if contracts == nil {
		return fmt.Errorf("contracts must be defined")
	}

	for k, v := range contracts.Map {
		abiType := abiPath + "." + k
		g.P("var ", k, "Abi", " *", abiType)
		g.P("func Get", k, "Abi(client *", ethclient, ", addr string) (*", abiType, ", error) {")
		g.P("	if ", k, "Abi != nil {")
		g.P("		return ", k, "Abi, nil")
		g.P("	}")
		// Parse contract address to common.Address
		g.P("	address:= common.HexToAddress(\"", v, "\")")
		g.P("	out, err := ", abiPath, ".New", k, "(address, client)")
		g.P("	if err != nil {")
		g.P("		return nil, err")
		g.P("	}")
		g.P("	", k, "Abi = out")
		g.P("	return out, nil")
		g.P("}")
	}

	g.P("var contractAddrMap = map[string]string {")
	for k, v := range contracts.Map {
		g.P("	\"", k, "\": \"", v, "\",")
	}
	g.P("}")

	return nil
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
	g.P("// Populate", nativeName, " populates an instance of ", nativeName, " from the provided client.")
	g.P("func Populate", nativeName, " (m *", nativeName, ", client *", g.QualifiedGoIdent(ethclient), ", opts *", g.QualifiedGoIdent(callOpts), ") error {")
	if len(m.Fields) > 0 {
		g.P("var err error")
	}

	for k, _ := range fields.contractGetters {
		g.P(k)
		g.P("if err != nil {")
		g.P("	return err")
		g.P("}")
	}

	for k, v := range fields.fieldMap {
		g.P("m.", k, ", err = contract", strings.Split(v, ".")[0], ".", strings.Split(v, ".")[1], "(opts)")
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
	abiPath, err := importAbi(g, f)
	if err != nil {
		return err
	}

	generateContractRegistry(g, f, abiPath)

	for _, m := range f.Messages {
		err := generateType(g, m)
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
