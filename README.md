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
Name: runtime.end
Address: 7475504
Size: 0
Code: 100
Type: STT_OBJECT
Binding: STB_LOCAL
Relocs: []
Name: runtime.enoptrbss
Address: 7475504
Size: 0
Code: 100
Type: STT_OBJECT
Binding: STB_LOCAL
Relocs: []
```

Note that I added parsing of the Type and Binding. I think I'm going to pull out using just the Dwarf wrapper and remove the internal code that isn't supposed to be accessible :)
See discussion in [this thread](https://twitter.com/vsoch/status/1437535961131352065) for the discovery of the missing variables. 
