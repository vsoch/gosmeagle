# Go Smeagle

This is mostly for fun - I wanted to see how hard it would be to parse a binary
with Go. Right now we use the same logic as objdump to load it, and then print
Symbols (and I found an entry to where the Dwarf is).

![img/smeagle.gif](img/smeagle.gif)

🚧️ **under development** 🚧️

## Usage

To build the gosmeagle binary, you can do:

```bash
$ make
```

You can also interact as follows:

```bash
$ go run main.go
```

### Parse

Parsing means outputting a corpus to JSON.

```bash
$ go run main.go parse libtest.so
```
```
{"library":"libtest.so","functions":[{"name":"__printf_chk","type":"Function"},{"parameters":[{"type":"long int","size":8},{"type":"long int","size":8},{"type":"long int","size":8},{"type":"long int","size":8},{"type":"long int","size":8},{"type":"__int128","size":16}],"name":"bigcall","type":"Function"}]}
```

or print pretty:

```
$ go run main.go parse libtest.so --pretty
```
```
{
    "library": "libtest.so",
    "functions": [
        {
            "name": "__printf_chk",
            "type": "Function"
        },
        {
            "parameters": [
                {
                    "type": "long int",
                    "size": 8
                },
                {
                    "type": "long int",
                    "size": 8
                },
                {
                    "type": "long int",
                    "size": 8
                },
                {
                    "type": "long int",
                    "size": 8
                },
                {
                    "type": "long int",
                    "size": 8
                },
                {
                    "type": "__int128",
                    "size": 16
                }
            ],
            "name": "bigcall",
            "type": "Function"
        }
    ]
}
```

### Disasm

Disassembling means printing Assembly.

```bash
$ go run main.go disasm libtest.so
```
```bash
TEXT register_tm_clones(SB) 

TEXT __do_global_dtors_aux(SB) 

TEXT frame_dummy(SB) 

TEXT bigcall(SB) 
  0x1120		f3			?								
  0x1121		0f			?								
  0x1122		1e			?								
  0x1123		fa			CLI                                  // cli			
  0x1124		4883ec10		SUBQ $0x10, SP                       // sub $0x10,%rsp		
  0x1128		4989c9			MOVQ CX, R9                          // mov %rcx,%r9		
  0x112b		31c0			XORL AX, AX                          // xor %eax,%eax		
  0x112d		4889f1			MOVQ SI, CX                          // mov %rsi,%rcx		
  0x1130		ff742418		PUSHQ 0x18(SP)                       // pushq 0x18(%rsp)	
  0x1134		488d35c50e0000		LEAQ 0xec5(IP), SI                   // lea 0xec5(%rip),%rsi	
  0x113b		ff742428		PUSHQ 0x28(SP)                       // pushq 0x28(%rsp)	
  0x113f		4150			PUSHL R8                             // push %r8		
  0x1141		4989d0			MOVQ DX, R8                          // mov %rdx,%r8		
  0x1144		4889fa			MOVQ DI, DX                          // mov %rdi,%rdx		
  0x1147		bf01000000		MOVL $0x1, DI                        // mov $0x1,%edi		
  0x114c		e8fffeffff		CALL 0x1050                          // callq 0x1050		
  0x1151		4883c428		ADDQ $0x28, SP                       // add $0x28,%rsp		
  0x1155		c3			RET                                  // retq
```

Note that this library is under development, so stay tuned!

## Background

I started this library after discussion (see [this thread](https://twitter.com/vsoch/status/1437535961131352065)) and wanting to extend Dwarf a bit and also reproduce [Smeagle](https://github.com/buildsi/Smeagle) in Go.

## Includes

Since I needed some functionality from [debug/dwarf](https://cs.opensource.google/go/go/+/master:src/debug/dwarf/) that was not public, the library is included here (with proper license/credit headers) in [pkg/debug](pkg/debug) along with ELF that needs to return a matching type. The changes I made include

 - renaming readType to ReadType so it's public.
 - also renaming sigToType to SigToType so it's public
 - made typeCache public (TypeCache)
 - Added an "Original" (interface) to a CommonType, and then changed ReadType in [dwarf/debug/type.go](pkg/dwarf/debug/type.go) so that each case sets `t.Original = t` so we can return the original type to further parse (`t.Common().Original`).
 - Added a StructCache to the dwarf.Data in [pkg/debug/dwarf/open.go](pkg/debub/dwarf/open.go) that is populated in [pkg/debug/dwarf/type.go](pkg/debug/dwarf/type.go) as follows:
 
```
// ADDED: save the struct to the struct cache for later lookup
d.StructCache[t.StructName] = t
```

And then used in [parsers/x86_64/parse.go](parsers/x86_64/parse.go) to match a typedef (which only has name and type string) to a fully parsed struct (a struct, union, or class).
