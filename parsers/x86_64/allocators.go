package x86_64

import (
	"log"
)

// A FramebaseAllocator keeps track of framebase index
type FramebaseAllocator struct {
	Framebase int64 // should default to 8

}

// NewFramebaseAllocator creates a new Framebase Allocator (starts at 8)
func NewFramebaseAllocator() *FramebaseAllocator {
	return &FramebaseAllocator{Framebase: 8}
}

// nextMultipleEight gets the next greater multiple of 8
func (a *FramebaseAllocator) nextMultipleEight(number int64) int64 { return ((number + 7) & (-8)) }

// updateFramebaseFromType gets a framebase for a variable based on stack location and type
// Framebase values must be 8 byte aligned.
func (a *FramebaseAllocator) updateFramebaseFromSize(size int64) {
	a.Framebase += a.nextMultipleEight(size)
}

// Get a framebase for a variable based on stack location and type
// Framebase values must be 8 byte aligned.
func (a *FramebaseAllocator) NextFramebaseFromSize(size int64) string {
	result := "framebase+" + string(a.Framebase)

	// Update the framebase for the next parameter based on the type
	a.updateFramebaseFromSize(size)
	return result
}

// A RegisterAllocator can provide the next register location
type RegisterAllocator struct {
	Framebase    int64
	Fallocator   *FramebaseAllocator
	SseRegisters []string
	IntRegisters []string
}

// NewRegisterAllocator creates a new Register Allocator
func NewRegisterAllocator() *RegisterAllocator {

	// Populate the sse register stack
	sse := []string{}
	for i := 7; i >= 0; i-- {
		sse = append(sse, "%xmm"+string(i))
	}

	// Populate the int register stack
	intRegisters := []string{"%r9", "%r8", "%rcx", "%rdx", "%rsi", "%rdi"}

	// Create a new framebase allocator
	fallocator := NewFramebaseAllocator()
	return &RegisterAllocator{SseRegisters: sse, Fallocator: fallocator, IntRegisters: intRegisters, Framebase: 8}

}

// getNextIntRegister gets the next available integer register
func (r *RegisterAllocator) getNextIntRegister() string {

	// If we are empty, return stack
	if len(r.IntRegisters) == 0 {
		return ""
	}

	// Get the next register string (the top of the "stack" or end of list
	top := len(r.IntRegisters) - 1
	regString := r.IntRegisters[top]

	// And update to remove it
	r.IntRegisters = append(r.IntRegisters[:top], r.IntRegisters[top+1:]...)
	return regString
}

// getNextSseRegister gets the next available sse register
func (r *RegisterAllocator) getNextSseRegister() string {

	// If we are empty, return stack
	if len(r.SseRegisters) == 0 {
		return ""
	}

	// Get the next register string (the top of the "stack" or end of list
	top := len(r.SseRegisters) - 1
	regString := r.SseRegisters[top]

	// And update to remove it
	r.IntRegisters = append(r.SseRegisters[:top], r.SseRegisters[top+1:]...)
	return regString
}

// GetRegisterString combines two registers to return one register string depending on the type
func (r *RegisterAllocator) GetRegisterString(lo RegisterClass, hi RegisterClass, size int64, typeString string) string {

	// Empty structs and unions don't have a location
	if lo == NO_CLASS && typeString == "Struct" {
		return "none"
	}

	if lo == NO_CLASS {
		log.Fatalf("Cannot allocate a {NO_CLASS, *} register")
	}

	// Memory lo goes on the stack
	if lo == MEMORY {
		return r.Fallocator.NextFramebaseFromSize(size)
	}

	if lo == INTEGER {
		reg := r.getNextIntRegister()

		// Ran out of registers, put it on the stack
		if reg == "" {
			return r.Fallocator.NextFramebaseFromSize(size)
		}
		return reg
	}

	if lo == SSE {
		reg := r.getNextSseRegister()

		// Ran out of registers, put it on the stack
		if reg == "" {
			return r.Fallocator.NextFramebaseFromSize(size)
		}

		if hi == SSEUP {
			// TODO If the class is SSEUP, the eightbyte is passed in the next available eightbyte
			// chunk of the last used vector register.
		}
		return reg

		/* TODO
		*
		*  For objects allocated in multiple registers, use the syntax '%r1 | %r2 | ...'
		*  to denote this. This can only happen for aggregates.
		*
		*  Use ymm and zmm for larger vector types and check for aliasing
		 */
	}

	// If the class is X87, X87UP or COMPLEX_X87, it is passed in memory
	if lo == X87 || lo == COMPLEX_X87 || hi == X87UP {
		return r.Fallocator.NextFramebaseFromSize(size)
	}

	// This should never be reached
	log.Fatalf("Unknown classification")
	return "unknown"
}
