// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// https://cs.opensource.google/go/go/+/refs/tags/go1.17.2:src/cmd/internal/objfile/disasm.go;drc=refs%2Ftags%2Fgo1.17.2;l=43

package file

import (
	"bufio"
	"bytes"
	"container/list"
	"debug/gosym"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"

	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/ppc64/ppc64asm"
	"golang.org/x/arch/x86/x86asm"
)

// Added from cmd/internal/src
const FileSymPrefix = "gofile.."

// Disasm is a disassembler for a given File.
type Disasm struct {
	syms      []Symbol         // symbols in file, sorted by address
	pcln      Liner            // pcln table
	text      []byte           // bytes of text segment (actual instructions)
	textStart uint64           // start PC of text
	textEnd   uint64           // end PC of text
	goarch    string           // GOARCH string
	disasm    disasmFunc       // disassembler function for goarch
	byteOrder binary.ByteOrder // byte order for goarch
	gnulookup gnuFunc          // Lookup GNU assembly
}

// Disasm returns a disassembler for the file f.
func (e *Entry) Disasm() (*Disasm, error) {
	syms, err := e.Symbols()
	if err != nil {
		return nil, err
	}

	// We want to disassemble both known and unknown (imported) symbols
	// TODO we could use this to derive import status?
	dyns, err := e.DynamicSymbols()
	if err != nil {
		return nil, err
	}

	pcln, err := e.PCLineTable()
	if err != nil {
		return nil, err
	}

	syms = append(syms, dyns...)
	textStart, textBytes, err := e.Text()
	if err != nil {
		return nil, err
	}

	goarch := e.GOARCH()
	disasm := disasms[goarch]
	gnulookup := gnuLookup[goarch]
	byteOrder := byteOrders[goarch]
	if disasm == nil || byteOrder == nil {
		return nil, fmt.Errorf("unsupported architecture")
	}

	// Filter out section symbols, overwriting syms in place.
	keep := syms[:0]
	for _, sym := range syms {
		switch sym.GetName() {
		case "runtime.text", "text", "_text", "runtime.etext", "etext", "_etext":
			// drop
		default:
			keep = append(keep, sym)
		}
	}
	syms = keep
	d := &Disasm{
		syms:      syms,
		pcln:      pcln,
		text:      textBytes,
		textStart: textStart,
		textEnd:   textStart + uint64(len(textBytes)),
		goarch:    goarch,
		disasm:    disasm,
		gnulookup: gnulookup,
		byteOrder: byteOrder,
	}
	return d, nil
}

// lookup finds the symbol name containing addr.
func (d *Disasm) Lookup(addr uint64) (name string, base uint64) {
	i := sort.Search(len(d.syms), func(i int) bool { return addr < d.syms[i].GetAddress() })
	if i > 0 {
		s := d.syms[i-1]
		if s.GetAddress() != 0 && s.GetAddress() <= addr && addr < s.GetAddress()+uint64(s.GetSize()) {
			return s.GetName(), s.GetAddress()
		}
	}
	return "", 0
}

// base returns the final element in the path.
// It works on both Windows and Unix paths,
// regardless of host operating system.
func base(path string) string {
	path = path[strings.LastIndex(path, "/")+1:]
	path = path[strings.LastIndex(path, `\`)+1:]
	return path
}

// CachedFile contains the content of a file split into lines.
type CachedFile struct {
	FileName string
	Lines    [][]byte
}

// FileCache is a simple LRU cache of file contents.
type FileCache struct {
	files  *list.List
	maxLen int
}

// NewFileCache returns a FileCache which can contain up to maxLen cached file contents.
func NewFileCache(maxLen int) *FileCache {
	return &FileCache{
		files:  list.New(),
		maxLen: maxLen,
	}
}

// Line returns the source code line for the given file and line number.
// If the file is not already cached, reads it, inserts it into the cache,
// and removes the least recently used file if necessary.
// If the file is in cache, it is moved to the front of the list.
func (fc *FileCache) Line(filename string, line int) ([]byte, error) {
	if filepath.Ext(filename) != ".go" {
		return nil, nil
	}

	// Clean filenames returned by src.Pos.SymFilename()
	// or src.PosBase.SymFilename() removing
	// the leading src.FileSymPrefix.
	filename = strings.TrimPrefix(filename, FileSymPrefix)

	// Expand literal "$GOROOT" rewritten by obj.AbsFile()
	filename = filepath.Clean(os.ExpandEnv(filename))

	var cf *CachedFile
	var e *list.Element

	for e = fc.files.Front(); e != nil; e = e.Next() {
		cf = e.Value.(*CachedFile)
		if cf.FileName == filename {
			break
		}
	}

	if e == nil {
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		cf = &CachedFile{
			FileName: filename,
			Lines:    bytes.Split(content, []byte{'\n'}),
		}
		fc.files.PushFront(cf)

		if fc.files.Len() >= fc.maxLen {
			fc.files.Remove(fc.files.Back())
		}
	} else {
		fc.files.MoveToFront(e)
	}

	// because //line directives can be out-of-range. (#36683)
	if line-1 >= len(cf.Lines) || line-1 < 0 {
		return nil, nil
	}

	return cf.Lines[line-1], nil
}

// Print prints a disassembly of the file to w.
// If filter is non-nil, the disassembly only includes functions with names matching filter.
// If printCode is true, the disassembly includs corresponding source lines.
// The disassembly only includes functions that overlap the range [start, end).
func (d *Disasm) Print(w io.Writer, filter *regexp.Regexp, start, end uint64, printCode bool, gnuAsm bool) {
	if start < d.textStart {
		start = d.textStart
	}
	if end > d.textEnd {
		end = d.textEnd
	}
	printed := false
	bw := bufio.NewWriter(w)

	var fc *FileCache
	if printCode {
		fc = NewFileCache(8)
	}

	tw := tabwriter.NewWriter(bw, 18, 8, 1, '\t', tabwriter.StripEscape)
	for _, sym := range d.syms {
		symStart := sym.GetAddress()
		symEnd := sym.GetAddress() + uint64(sym.GetSize())
		relocs := sym.GetRelocations()
		if sym.GetCode() != 'T' && sym.GetCode() != 't' ||
			symStart < d.textStart ||
			symEnd <= start || end <= symStart ||
			filter != nil && !filter.MatchString(sym.GetName()) {
			continue
		}
		if printed {
			fmt.Fprintf(bw, "\n")
		}
		printed = true

		file, _, _ := d.pcln.PCToLine(sym.GetAddress())
		fmt.Fprintf(bw, "TEXT %s(SB) %s\n", sym.GetName(), file)

		if symEnd > end {
			symEnd = end
		}
		code := d.text[:end-d.textStart]

		var lastFile string
		var lastLine int

		d.Decode(symStart, symEnd, relocs, gnuAsm, func(pc, size uint64, file string, line int, text string) {
			i := pc - d.textStart

			if printCode {
				if file != lastFile || line != lastLine {
					if srcLine, err := fc.Line(file, line); err == nil {
						fmt.Fprintf(tw, "%s%s%s\n", []byte{tabwriter.Escape}, srcLine, []byte{tabwriter.Escape})
					}

					lastFile, lastLine = file, line
				}

				fmt.Fprintf(tw, "  %#x\t", pc)
			} else {
				fmt.Fprintf(tw, "  %s:%d\t%#x\t", base(file), line, pc)
			}

			if size%4 != 0 || d.goarch == "386" || d.goarch == "amd64" {
				// Print instruction as bytes.
				fmt.Fprintf(tw, "%x", code[i:i+size])
			} else {
				// Print instruction as 32-bit words.
				for j := uint64(0); j < size; j += 4 {
					if j > 0 {
						fmt.Fprintf(tw, " ")
					}
					fmt.Fprintf(tw, "%08x", d.byteOrder.Uint32(code[i+j:]))
				}
			}
			fmt.Fprintf(tw, "\t%s\t\n", text)
		})
		tw.Flush()
	}
	bw.Flush()
}

// GNUAssembly holds parsed assembly for symbols
type Instruction struct {
	Text     string
	Source   string
	Assembly string
	PCLine   uint64
	Size     uint64
}

type GNUAssembly struct {
	SymbolName   string
	File         string
	PCLine       uint64
	SrcLine      string
	Instructions []Instruction
}

type GNUAssemblyLookup map[string]GNUAssembly

// GetAssembly returns a full data structure of instructions to further parse
func (d *Disasm) GetGNUAssembly() GNUAssemblyLookup {
	fc := NewFileCache(8)

	asm := map[string]GNUAssembly{}

	for _, sym := range d.syms {
		symStart := sym.GetAddress()
		symEnd := sym.GetAddress() + uint64(sym.GetSize())
		relocs := sym.GetRelocations()
		if sym.GetCode() != 'T' && sym.GetCode() != 't' ||
			symStart < d.textStart ||
			symEnd <= d.textStart ||
			symEnd-d.textStart > uint64(len(d.text)) {
			continue
		}

		file, _, _ := d.pcln.PCToLine(sym.GetAddress())
		entry := GNUAssembly{SymbolName: sym.GetName(), File: file}
		code := d.text[:symEnd-d.textStart]

		var lastFile string
		var lastLine int

		instructions := d.DecodeGNUAssembly(symStart, symEnd, relocs, func(pc, size uint64, file string, line int, text string) Instruction {
			i := pc - d.textStart

			instruction := Instruction{PCLine: pc, Size: size}
			if file != lastFile || line != lastLine {
				if srcLine, err := fc.Line(file, line); err == nil {
					instruction.Source = fmt.Sprintf("%s", srcLine)
				}
				lastFile, lastLine = file, line
			}

			if size%4 != 0 || d.goarch == "386" || d.goarch == "amd64" {
				// Print instruction as bytes.
				instruction.Text = fmt.Sprintf("%x", code[i:i+size])
			} else {
				// Print instruction as 32-bit words.
				for j := uint64(0); j < size; j += 4 {
					if j > 0 {
						text += " "
					}
					instruction.Text = fmt.Sprintf("%08x", d.byteOrder.Uint32(code[i+j:]))
				}
			}
			return instruction
		})
		entry.Instructions = append(entry.Instructions, instructions...)
		asm[entry.SymbolName] = entry
	}
	return asm
}

// DecodeGNUAssembly disassembles the text segment range [start, end), calling f for each instruction.
func (d *Disasm) DecodeGNUAssembly(start, end uint64, relocs []Relocation, f func(pc, size uint64, file string, line int, text string) Instruction) []Instruction {
	if start < d.textStart {
		start = d.textStart
	}
	if end > d.textEnd {
		end = d.textEnd
	}

	instructions := []Instruction{}
	code := d.text[:end-d.textStart]
	for pc := start; pc < end; {
		i := pc - d.textStart
		text, size := d.gnulookup(code[i:], pc, d.byteOrder)
		file, line, _ := d.pcln.PCToLine(pc)
		sep := "\t"
		for len(relocs) > 0 && relocs[0].Address < i+uint64(size) {
			text += sep + relocs[0].Stringer.String(pc-start)
			sep = " "
			relocs = relocs[1:]
		}
		instruction := f(pc, uint64(size), file, line, text)
		instruction.Text = text
		instructions = append(instructions, instruction)
		pc += uint64(size)
	}
	return instructions
}

// Decode disassembles the text segment range [start, end), calling f for each instruction.
func (d *Disasm) Decode(start, end uint64, relocs []Relocation, gnuAsm bool, f func(pc, size uint64, file string, line int, text string)) {
	if start < d.textStart {
		start = d.textStart
	}
	if end > d.textEnd {
		end = d.textEnd
	}
	code := d.text[:end-d.textStart]
	lookup := d.Lookup
	for pc := start; pc < end; {
		i := pc - d.textStart
		text, size := d.disasm(code[i:], pc, lookup, d.byteOrder, gnuAsm)
		file, line, _ := d.pcln.PCToLine(pc)
		sep := "\t"
		for len(relocs) > 0 && relocs[0].Address < i+uint64(size) {
			text += sep + relocs[0].Stringer.String(pc-start)
			sep = " "
			relocs = relocs[1:]
		}
		f(pc, uint64(size), file, line, text)
		pc += uint64(size)
	}
}

type lookupFunc = func(addr uint64) (sym string, base uint64)
type disasmFunc func(code []byte, pc uint64, lookup lookupFunc, ord binary.ByteOrder, _ bool) (text string, size int)

// Disasm Functions

func disasm_386(code []byte, pc uint64, lookup lookupFunc, _ binary.ByteOrder, gnuAsm bool) (string, int) {
	return disasm_x86(code, pc, lookup, 32, gnuAsm)
}

func disasm_amd64(code []byte, pc uint64, lookup lookupFunc, _ binary.ByteOrder, gnuAsm bool) (string, int) {
	return disasm_x86(code, pc, lookup, 64, gnuAsm)
}

func disasm_x86(code []byte, pc uint64, lookup lookupFunc, arch int, gnuAsm bool) (string, int) {
	inst, err := x86asm.Decode(code, arch)
	var text string
	size := inst.Len
	if err != nil || size == 0 || inst.Op == 0 {
		size = 1
		text = "?"
	} else {
		if gnuAsm {
			text = fmt.Sprintf("%-36s // %s", x86asm.GoSyntax(inst, pc, lookup), x86asm.GNUSyntax(inst, pc, nil))
		} else {
			text = x86asm.GoSyntax(inst, pc, lookup)
		}
	}
	return text, size
}

type textReader struct {
	code []byte
	pc   uint64
}

func (r textReader) ReadAt(data []byte, off int64) (n int, err error) {
	if off < 0 || uint64(off) < r.pc {
		return 0, io.EOF
	}
	d := uint64(off) - r.pc
	if d >= uint64(len(r.code)) {
		return 0, io.EOF
	}
	n = copy(data, r.code[d:])
	if n < len(data) {
		err = io.ErrUnexpectedEOF
	}
	return
}

func disasm_arm(code []byte, pc uint64, lookup lookupFunc, _ binary.ByteOrder, gnuAsm bool) (string, int) {
	inst, err := armasm.Decode(code, armasm.ModeARM)
	var text string
	size := inst.Len
	if err != nil || size == 0 || inst.Op == 0 {
		size = 4
		text = "?"
	} else if gnuAsm {
		text = fmt.Sprintf("%-36s // %s", armasm.GoSyntax(inst, pc, lookup, textReader{code, pc}), armasm.GNUSyntax(inst))
	} else {
		text = armasm.GoSyntax(inst, pc, lookup, textReader{code, pc})
	}
	return text, size
}

func disasm_arm64(code []byte, pc uint64, lookup lookupFunc, byteOrder binary.ByteOrder, gnuAsm bool) (string, int) {
	inst, err := arm64asm.Decode(code)
	var text string
	if err != nil || inst.Op == 0 {
		text = "?"
	} else if gnuAsm {
		text = fmt.Sprintf("%-36s // %s", arm64asm.GoSyntax(inst, pc, lookup, textReader{code, pc}), arm64asm.GNUSyntax(inst))
	} else {
		text = arm64asm.GoSyntax(inst, pc, lookup, textReader{code, pc})
	}
	return text, 4
}

func disasm_ppc64(code []byte, pc uint64, lookup lookupFunc, byteOrder binary.ByteOrder, gnuAsm bool) (string, int) {
	inst, err := ppc64asm.Decode(code, byteOrder)
	var text string
	size := inst.Len
	if err != nil || size == 0 {
		size = 4
		text = "?"
	} else {
		if gnuAsm {
			text = fmt.Sprintf("%-36s // %s", ppc64asm.GoSyntax(inst, pc, lookup), ppc64asm.GNUSyntax(inst, pc))
		} else {
			text = ppc64asm.GoSyntax(inst, pc, lookup)
		}
	}
	return text, size
}

var disasms = map[string]disasmFunc{
	"386":     disasm_386,
	"amd64":   disasm_amd64,
	"arm":     disasm_arm,
	"arm64":   disasm_arm64,
	"ppc64":   disasm_ppc64,
	"ppc64le": disasm_ppc64,
}

// GNU assembly lookup

type gnuFunc func(code []byte, pc uint64, ord binary.ByteOrder) (text string, size int)

func gnulookup_386(code []byte, pc uint64, _ binary.ByteOrder) (string, int) {
	return gnulookup_x86(code, pc, 32)
}

func gnulookup_amd64(code []byte, pc uint64, _ binary.ByteOrder) (string, int) {
	return gnulookup_x86(code, pc, 64)
}

func gnulookup_x86(code []byte, pc uint64, arch int) (string, int) {
	inst, err := x86asm.Decode(code, arch)
	var text string
	size := inst.Len
	if err != nil || size == 0 || inst.Op == 0 {
		size = 1
		text = "?"
	} else {
		text = fmt.Sprintf("%s", x86asm.GNUSyntax(inst, pc, nil))
	}
	return text, size
}

func gnulookup_arm(code []byte, pc uint64, _ binary.ByteOrder) (string, int) {
	inst, err := armasm.Decode(code, armasm.ModeARM)
	var text string
	size := inst.Len
	if err != nil || size == 0 || inst.Op == 0 {
		size = 4
		text = "?"
	} else {
		text = fmt.Sprintf("%s", armasm.GNUSyntax(inst))
	}
	return text, size
}

func gnulookup_arm64(code []byte, pc uint64, byteOrder binary.ByteOrder) (string, int) {
	inst, err := arm64asm.Decode(code)
	var text string
	if err != nil || inst.Op == 0 {
		text = "?"
	} else {
		text = fmt.Sprintf("%s", arm64asm.GNUSyntax(inst))
	}
	return text, 4
}

func gnulookup_ppc64(code []byte, pc uint64, byteOrder binary.ByteOrder) (string, int) {
	inst, err := ppc64asm.Decode(code, byteOrder)
	var text string
	size := inst.Len
	if err != nil || size == 0 {
		size = 4
		text = "?"
	} else {
		text = fmt.Sprintf("%s", ppc64asm.GNUSyntax(inst, pc))
	}
	return text, size
}

var gnuLookup = map[string]gnuFunc{
	"386":     gnulookup_386,
	"amd64":   gnulookup_amd64,
	"arm":     gnulookup_arm,
	"arm64":   gnulookup_arm64,
	"ppc64":   gnulookup_ppc64,
	"ppc64le": gnulookup_ppc64,
}

var byteOrders = map[string]binary.ByteOrder{
	"386":     binary.LittleEndian,
	"amd64":   binary.LittleEndian,
	"arm":     binary.LittleEndian,
	"arm64":   binary.LittleEndian,
	"ppc64":   binary.BigEndian,
	"ppc64le": binary.LittleEndian,
	"s390x":   binary.BigEndian,
}

type Liner interface {
	// Given a pc, returns the corresponding file, line, and function data.
	// If unknown, returns "",0,nil.
	PCToLine(uint64) (string, int, *gosym.Func)
}
