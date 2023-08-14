package main

import "github.com/ethereum/go-ethereum/accounts/abi"

// In-memory representation of a single field
type Field struct {
	Name     string // Must be a valid golang field name (alphanumeric plus underscore)
	Contract string // Must be a valid ethereum contract name, expected to be in the abigen format
	Selector *abi.SelectorMarshaling
	Type     string
}

// In-memory representation of a single struct
type Struct struct {
	Name   string
	Fields []*Field

	// For internal use, contracts, deduplicated and sorted.
	contracts []string
}

// In-memory representation of a single file defining types to generate
// Supports json or yaml
type File struct {
	AbiPackage string // Package of the artifacts of abigen, if not the same as the output package
	Version    string // Must be valid golang.org/x/mod/semver
	Structs    []*Struct
}
