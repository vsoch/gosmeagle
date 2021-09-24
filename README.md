# Go Smeagle

This is mostly for fun - I wanted to see how hard it would be to parse a binary
with Go. Right now we use the same logic as objdump to load it, and then print
Symbols (and I found an entry to where the Dwarf is).

🚧️ **under development** 🚧️

## Usage

To run and preview the output, do:

```bash
$ make
$ go run main.go parse gosmeagle
```
```
...
[{a 8 long int } {b 8 long int } {c 8 long int } {d 8 long int } {e 8 long int } {f 16 __int128 }]
[{__fmt -1  }]
[]
```

Note that this library is under development, so currently I've just finished parsing functions
and formal paramters from the dwarf, and next I'm going to map that to an x86_64 parser to get more
metadata. Stay tuned!

Note that I added parsing of the Type and Binding. I think I'm going to pull out using just the Dwarf wrapper and remove the internal code that isn't supposed to be accessible :)
See discussion in [this thread](https://twitter.com/vsoch/status/1437535961131352065) for the discovery of the missing variables. 

## Includes

Since I needed some functionality from [debug/dwarf](https://cs.opensource.google/go/go/+/master:src/debug/dwarf/) that was not public, the library is included here (with proper license/credit headers) in [pkg/debug](pkg/debug) along with ELF that needs to return a matching type. The changes I made include

 - renaming readType to ReadType so it's public.
 - also renaming sigToType to SigToType so it's public
 - made typeCache public (TypeCache)
