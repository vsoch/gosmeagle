package file

import (
	"fmt"
	"github.com/vsoch/gosmeagle/pkg/debug/elf"
)

// https://cs.opensource.google/go/go/+/master:src/debug/elf/elf.go;l=3108
type Relocation struct {
	Address  uint64 // Address of first byte that reloc applies to.
	Size     uint64 // Number of bytes
	Stringer RelocStringer

	// Added to suppored retriving raw
	SymbolName   string
	Offset       uint64
	SymbolValue  uint64
	RelocType    string // can be parse from into
	Info         uint64
	SectionIndex int
}

func getRelocationType(rType uint32, mType elf.Machine) string {
	switch mType {
	case elf.EM_X86_64:
		return fmt.Sprintf("%s", elf.R_X86_64(rType))
	case elf.EM_386:
		return fmt.Sprintf("%s", elf.R_386(rType))
	case elf.EM_ARM:
		return fmt.Sprintf("%s", elf.R_ARM(rType))
	case elf.EM_AARCH64:
		return fmt.Sprintf("%s", elf.R_AARCH64(rType))
	case elf.EM_PPC:
		return fmt.Sprintf("%s", elf.R_PPC(rType))
	case elf.EM_PPC64:
		return fmt.Sprintf("%s", elf.R_PPC64(rType))
	case elf.EM_MIPS:
		return fmt.Sprintf("%s", elf.R_MIPS(rType))
	case elf.EM_RISCV:
		return fmt.Sprintf("%s", elf.R_RISCV(rType))
	case elf.EM_S390:
		return fmt.Sprintf("%s", elf.R_390(rType))
	case elf.EM_SPARCV9:
		return fmt.Sprintf("%s", elf.R_SPARC(rType))
	default:
		return "R_UNKNOWN"
	}
}

type RelocStringer interface {
	// insnOffset is the offset of the instruction containing the relocation
	// from the start of the symbol containing the relocation.
	String(insnOffset uint64) string
}

// Print the relocation table - currently not used
func PrintRelocationTable(relocs []Relocation) {
	fmt.Println("Offset\t\t\tInfo\t\t\t\tType\t\t\tSym.Value\t\t\tSym.Name")
	for _, r := range relocs {
		fmt.Printf("%016x\t%016x\t%s\t%016x\t\t%s\n", r.Offset, r.Info, r.RelocType, r.SymbolValue, r.SymbolName)
	}
}
