package file

// This is based on work in the go source code, but modified
// to be outside of a command. It is covered by a BSD-style
// license.

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"debug/gosym"
	"github.com/vsoch/gosmeagle/pkg/debug/dwarf"
)

// An opened File - can be multiple types
type File struct {
	handle  *os.File
	Entries []*Entry
}

// A generic Entry in a file has a name and data
type Entry struct {
	Name string
	data rawFile
}

// Symbols and other data associated with the file
// Each of these functions needs to be defined for the specific file
type rawFile interface {
	Dwarf() (*dwarf.Data, error)
	ParseDwarf() map[string]map[string]DwarfEntry
	GoArch() string

	// renamed from pcln
	PCLineTable() (textStart uint64, symtab, pclntab []byte, err error)
	loadAddress() (uint64, error)
	DynamicSymbols() (syms []Symbol, err error)
	Symbols() (syms []Symbol, err error)
	text() (textStart uint64, text []byte, err error)
}

// LoadAddress returns the EXPECTED (not actual) address of the file.
func (e *Entry) LoadAddress() (uint64, error) {
	return e.data.loadAddress()
}

// Return Dwarf debug information (if it exists)
func (e *Entry) Dwarf() (*dwarf.Data, error) {
	return e.data.Dwarf()
}

// A raw symbol provides extra functions for interaction
type Symbol interface {
	GetName() string    // symbol name
	GetAddress() uint64 // virtual address of symbol
	GetSize() int64     // size in bytes
	GetCode() rune      // nm code (T for text, D for data, and so on)
	GetType() string    // string of type calculated from s.Info
	GetLibrary() string
	GetVersion() string
	GetBinding() string // binding calculated from s.Info
	GetRelocations() []Relocation
	GetOriginal() interface{}
	GetDirection() string // import or export based on symbol definition
	GetIntArch() int
}

type Relocation struct {
	Address  uint64 // Address of first byte that reloc applies to.
	Size     uint64 // Number of bytes
	Stringer RelocStringer
}

type RelocStringer interface {
	// insnOffset is the offset of the instruction containing the relocation
	// from the start of the symbol containing the relocation.
	String(insnOffset uint64) string
}

// We need to have multiple openers to handle different kinds of files
var openers = []func(io.ReaderAt) (rawFile, error){
	OpenElf,
	//	openMacho,
	//	openPE,
	//	openPlan9,
	//	openXcoff,
}

// Open the named file (please close f.Close after finishing)
func Open(name string) (*File, error) {
	handle, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	// eventually parse go files
	// requires cmd/objfile/goobj.go to be public
	//if f, err := openGoFile(handle); err == nil {
	//	return f, nil
	//}
	for _, function := range openers {
		if data, err := function(handle); err == nil {
			return &File{handle, []*Entry{{data: data}}}, nil
		}
	}
	handle.Close()
	return nil, fmt.Errorf("cannot open %s: unrecognized file type", name)
}

// Close the file handle
func (f *File) Close() error {
	return f.handle.Close()
}

// The architecture has to be consistent throughout the file
func (f *File) GoArch() string {
	return f.Entries[0].GOARCH()
}

func (f *File) DynamicSymbols() ([]Symbol, error) {
	return f.Entries[0].DynamicSymbols()
}

func (f *File) Symbols() ([]Symbol, error) {
	return f.Entries[0].Symbols()
}

func (f *File) PCLineTable() (Liner, error) {
	return f.Entries[0].PCLineTable()
}

func (f *File) Text() (uint64, []byte, error) {
	return f.Entries[0].Text()
}

func (f *File) LoadAddress() (uint64, error) {
	return f.Entries[0].LoadAddress()
}

// Added back to support getting assembly to parse call sites
func (f *File) Disasm() (*Disasm, error) {
	return f.Entries[0].Disasm()
}

// Since this returns the top node (root), it returns all the dwarf
func (f *File) DWARF() (*dwarf.Data, error) {
	return f.Entries[0].Dwarf()
}

func (f *File) ParseDwarf() map[string]map[string]DwarfEntry {
	dwf, err := f.Entries[0].Dwarf()
	if err != nil {
		log.Fatalf("Error parsing dwarf %v", err)
	}
	return ParseDwarf(dwf)
}

func (e *Entry) Symbols() ([]Symbol, error) {
	syms, err := e.data.Symbols()
	if err != nil {
		return nil, err
	}
	sort.Sort(SortByAddress(syms))
	return syms, nil
}

func (e *Entry) DynamicSymbols() ([]Symbol, error) {
	syms, err := e.data.DynamicSymbols()
	if err != nil {
		return nil, err
	}
	sort.Sort(SortByAddress(syms))
	return syms, nil
}

func (e *Entry) PCLineTable() (Liner, error) {
	// If the raw file implements Liner directly, use that.
	// Currently, only Go intermediate objects and archives (goobj) use this path.
	if pcln, ok := e.data.(Liner); ok {
		return pcln, nil
	}
	// Otherwise, read the pcln tables and build a Liner out of that.
	textStart, symtab, pclntab, err := e.data.PCLineTable()
	if err != nil {
		return nil, err
	}
	return gosym.NewTable(symtab, gosym.NewLineTable(pclntab, textStart))
}

// Text returns the text assocaited with the entry
func (e *Entry) Text() (uint64, []byte, error) {
	return e.data.text()
}

// GOARCH returns the GOARCH assocaited with the entry
func (e *Entry) GOARCH() string {
	return e.data.GoArch()
}
