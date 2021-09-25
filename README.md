# Go Smeagle

This is mostly for fun - I wanted to see how hard it would be to parse a binary
with Go. Right now we use the same logic as objdump to load it, and then print
Symbols (and I found an entry to where the Dwarf is).

🚧️ **under development** 🚧️

## Usage

To run and preview the output, do:

```bash
$ make
$ go run main.go parse libtest.so
```
```
{"library":"libtest.so","functions":[{"name":"__printf_chk"},{"parameters":[{"name":"a","type":"long int","sizes":8},{"name":"b","type":"long int","sizes":8},{"name":"c","type":"long int","sizes":8},{"name":"d","type":"long int","sizes":8},{"name":"e","type":"long int","sizes":8},{"name":"f","type":"__int128","sizes":16}],"name":"bigcall"}]}
```

or print pretty:

```
$ go run main.go parse libtest.so --pretty
```

Note that this library is under development, so stay tuned!

## Background

I started this library after discussion (see [this thread](https://twitter.com/vsoch/status/1437535961131352065)) and wanting to extend Dwarf a bit and also reproduce [Smeagle](https://github.com/buildsi/Smeagle) in Go.

## Includes

Since I needed some functionality from [debug/dwarf](https://cs.opensource.google/go/go/+/master:src/debug/dwarf/) that was not public, the library is included here (with proper license/credit headers) in [pkg/debug](pkg/debug) along with ELF that needs to return a matching type. The changes I made include

 - renaming readType to ReadType so it's public.
 - also renaming sigToType to SigToType so it's public
 - made typeCache public (TypeCache)

## TODO

 - add variable parsing
 - add allocator to get allocations
 - need to get registers / locations for each type
 
 TODO: the typecache stores a TYPE and we need to also keep track of the CLASS or KIND of type, so add to that.
 
 
