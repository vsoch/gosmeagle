package file

import (
	"encoding/binary"
	"fmt"
	"github.com/vsoch/gosmeagle/pkg/debug/dwarf"
	"github.com/vsoch/gosmeagle/pkg/debug/elf"
	"io"
	"log"
)

type ElfFile struct {
	elf *elf.File
}

// Parse dwarf into the file object
func (f *ElfFile) ParseDwarf() map[string]map[string]DwarfEntry {
	dwf, err := f.Dwarf()
	if err != nil {
		log.Fatalf("Error parsing dwarf %v", err)
	}
	return ParseDwarf(dwf)
}

func OpenElf(r io.ReaderAt) (rawFile, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	return &ElfFile{f}, nil
}

// An Elf Symbol found in a file (e.g., ELF?)
type ElfSymbol struct {
	Name        string       // symbol name
	Address     uint64       // virtual address of symbol
	Size        int64        // size in bytes
	Code        rune         // nm code (T for text, D for data, and so on)
	Type        string       // string of type calculated from s.Info
	Binding     string       // binding calculated from s.Info
	Relocations []Relocation // in increasing Addr order
	Original    elf.Symbol   // hold the original symbol
}

// And functions required for an elf symbol
func (s *ElfSymbol) GetName() string {
	return s.Name
}

// GetDirection determines if we have import/export based on definition
func (s *ElfSymbol) GetDirection() string {
	// imports are undefined (U)
	direction := "export"
	if s.GetCode() == 'U' {
		direction = "import"
	}
	return direction
}

func (s *ElfSymbol) GetAddress() uint64 {
	return s.Address
}

func (s *ElfSymbol) GetSize() int64 {
	return s.Size
}

func (s *ElfSymbol) GetCode() rune {
	return s.Code
}

func (s *ElfSymbol) GetType() string {
	return s.Type
}

func (s *ElfSymbol) GetBinding() string {
	return s.Binding
}

func (s *ElfSymbol) GetRelocations() []Relocation {
	return s.Relocations
}

func (s *ElfSymbol) GetOriginal() interface{} {
	return s.Original
}

// getSymbolType from a s.Info, which also can derive the binding
// https://refspecs.linuxfoundation.org/elf/elf.pdf section 1-18
func getSymbolType(s elf.Symbol) string {
	symType := int(s.Info) & 0xf

	// Return a human friendly string (this is what Type expects)
	switch symType {
	case 0:
		return "STT_NOTYPE"
	case 1:
		return "STT_OBJECT"
	case 2:
		return "STT_FUNC"
	case 4:
		return "STT_FILE"
	case 13:
		return "STT_LOPROC"
	case 15:
		return "STT_HIPROC"
	}
	return "UNKNOWN"
}

// getSymbolBinding from s.Info
func getSymbolBinding(s elf.Symbol) string {
	binding := s.Info >> 4
	switch binding {
	case 0:
		return "STB_LOCAL"
	case 1:
		return "STB_GLOBAL"
	case 2:
		return "STB_WEAK"
	case 13:
		return "STB_LOPROC"
	case 15:
		return "STB_HIPROC"
	}
	return "UNKNOWN"
}

// setSymbolCode for the symbol and file
func setSymbolCode(s *elf.Symbol, symbol *ElfSymbol, f *ElfFile) {

	switch s.Section {
	case elf.SHN_UNDEF:
		symbol.Code = 'U'
	case elf.SHN_COMMON:
		symbol.Code = 'B'
	default:
		i := int(s.Section)
		if i < 0 || i >= len(f.elf.Sections) {
			break
		}
		sect := f.elf.Sections[i]
		switch sect.Flags & (elf.SHF_WRITE | elf.SHF_ALLOC | elf.SHF_EXECINSTR) {
		case elf.SHF_ALLOC | elf.SHF_EXECINSTR:
			symbol.Code = 'T'
		case elf.SHF_ALLOC:
			symbol.Code = 'R'
		case elf.SHF_ALLOC | elf.SHF_WRITE:
			symbol.Code = 'D'
		}
	}
	if elf.ST_BIND(s.Info) == elf.STB_LOCAL {
		symbol.Code += 'a' - 'A'
	}
}

// Get dynamic symbols for the elf file
func (f *ElfFile) Symbols() ([]Symbol, error) {
	elfSyms, err := f.elf.DynamicSymbols()
	if err != nil {
		return nil, err
	}

	// TODO look up imported symbols to give direction
	// https://cs.opensource.google/go/go/+/master:src/debug/elf/file.go;l=1285?q=DynamicSymbols&ss=go%2Fgo
	var syms []Symbol
	for _, s := range elfSyms {

		// Convert the s.Info (we can use to calculate binding and type) to unsigned int, then string
		symType := getSymbolType(s)
		binding := getSymbolBinding(s)

		// Assume to start we don't know the code
		symbol := ElfSymbol{Address: s.Value, Type: symType, Binding: binding,
			Name: s.Name, Size: int64(s.Size), Code: '?', Original: s}

		// Add the correct code for the symbol
		setSymbolCode(&s, &symbol, f)
		syms = append(syms, &symbol)
	}

	return syms, nil
}

/*func (f *elfFile) pcln() (textStart uint64, symtab, pclntab []byte, err error) {
	if sect := f.elf.Section(".text"); sect != nil {
		textStart = sect.Addr
	}
	if sect := f.elf.Section(".gosymtab"); sect != nil {
		if symtab, err = sect.Data(); err != nil {
			return 0, nil, nil, err
		}
	}
	if sect := f.elf.Section(".gopclntab"); sect != nil {
		if pclntab, err = sect.Data(); err != nil {
			return 0, nil, nil, err
		}
	}
	return textStart, symtab, pclntab, nil
}*/

// Return text section of the ELF file
func (f *ElfFile) text() (textStart uint64, text []byte, err error) {
	sect := f.elf.Section(".text")
	if sect == nil {
		return 0, nil, fmt.Errorf("text section not found")
	}
	textStart = sect.Addr
	text, err = sect.Data()
	return
}

// GoArch returns the architecture of the elf file
func (f *ElfFile) GoArch() string {
	switch f.elf.Machine {
	case elf.EM_386:
		return "386"
	case elf.EM_X86_64:
		return "amd64"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_AARCH64:
		return "arm64"
	case elf.EM_PPC64:
		if f.elf.ByteOrder == binary.LittleEndian {
			return "ppc64le"
		}
		return "ppc64"
	case elf.EM_S390:
		return "s390x"
	}
	return ""
}

// loadAddress returns the load address
func (f *ElfFile) loadAddress() (uint64, error) {
	for _, p := range f.elf.Progs {
		if p.Type == elf.PT_LOAD && p.Flags&elf.PF_X != 0 {
			return p.Vaddr, nil
		}
	}
	return 0, fmt.Errorf("unknown load address")
}

func (f *ElfFile) Dwarf() (*dwarf.Data, error) {
	return f.elf.DWARF()
}
