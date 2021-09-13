# Go Smeagle

This is mostly for fun - I wanted to see how hard it would be to parse a binary
with Go. Right now we use the same logic as objdump to load it, and then print
Symbols (and I found an entry to where the Dwarf is).


## Usage

To run and preview the output, do:

```bash
$ make
$ go run main.go parse gosmeagle
```
```
...
Name: runtime.memstats
Address: 7456096
Size: 15312
Code: 68
Type: 
Relocs: []
Name: runtime.end
Address: 7471408
Size: 0
Code: 100
Type: 
Relocs: []
Name: runtime.enoptrbss
Address: 7471408
Size: 0
Code: 100
Type: 
Relocs: []
```

I'm not sure why there isn't output for Type, [here](https://cs.opensource.google/go/go/+/refs/tags/go1.17.1:src/cmd/internal/objfile/objfile.go;l=44;drc=refs%2Ftags%2Fgo1.17.1). I am next going to look at the DWARF.
