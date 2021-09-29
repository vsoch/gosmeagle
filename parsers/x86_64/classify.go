package x86_64

// A register class for AMD64 is defined on page 16 of the System V abi pdf

import (
	"fmt"
	"github.com/vsoch/gosmeagle/parsers/file"
	"github.com/vsoch/gosmeagle/pkg/debug/dwarf"
	"strings"
)

type RegisterClass int

const (
	INTEGER     RegisterClass = iota // Integer types that fit into one of the general purpose registers
	SSE                              // Types that fit into an SSE register
	SSEUP                            // ^.. and can ve passed and returned in he most significant half of it
	X87                              // Types that will be returned via the x87 FPU
	X87UP                            // ^
	COMPLEX_X87                      // Types that will be returned via the x87 FPU
	NO_CLASS                         // Initalizer in the algorithms, used for padding and empty structs/unions
	MEMORY                           // Types that will be passed and returned in memory via the stack
)

func (r RegisterClass) String() string {
	switch r {
	case INTEGER:
		return "INTEGER"
	case SSE:
		return "SSE"
	case SSEUP:
		return "SSEUP"
	case X87:
		return "X87"
	case X87UP:
		return "X87UP"
	case COMPLEX_X87:
		return "COMPLEX_X87"
	case NO_CLASS:
		return "NO_CLASS"
	case MEMORY:
		return "MEMORY"
	}
	return "UNKNOWN"
}

type Classification struct {
	Lo                  RegisterClass
	Hi                  RegisterClass
	Name                string
	PointerIndirections int64
}

// ClassifyPointer will classify a pointer
func ClassifyPointer(ptrCount *int64) Classification {
	return Classification{Lo: INTEGER, Hi: NO_CLASS, Name: "Pointer", PointerIndirections: (*ptrCount)}
}

// ClassifyArray will classify an array
func ClassifyArray(t *dwarf.ArrayType, c *file.Component, ptrCount *int64) Classification {

	size := t.Type.Size()
	if size > 64 {
		return Classification{Lo: MEMORY, Hi: NO_CLASS, Name: "Array"}
	}

	// Just classify the base type
	return ClassifyType(c, ptrCount)
}

// ClassifyStruct classifies a struct
func ClassifyStruct(t *dwarf.StructType, c *file.Component, ptrCount *int64) Classification {

	size := t.CommonType.Size()
	kind := strings.Title(t.Kind)

	if size > 64 {
		return Classification{Lo: MEMORY, Hi: NO_CLASS, Name: kind}
	}

	hi := NO_CLASS
	lo := NO_CLASS

	// Merge fields into final classification
	for _, field := range t.Field {

		c := file.Component{Name: field.Name, Class: file.GetStringType(field.Type),
			Size: field.Type.Size(), RawType: field.Type}
		fieldClass := ClassifyType(&c, ptrCount)
		hi = merge(hi, fieldClass.Hi)
		lo = merge(lo, fieldClass.Lo)
	}

	// Run post merge step
	postMerge(&lo, &hi, size)
	return Classification{Lo: lo, Hi: hi, Name: kind}
}

// Merge lo and hi, Page 21 (bottom) AMD64 ABI - method to come up with final classification based on two
func merge(originalReg RegisterClass, newReg RegisterClass) RegisterClass {

	// a. If both classes are equal, this is the resulting class.
	if originalReg == newReg {
		return originalReg
	}

	// b. If one of the classes is NO_CLASS, the resulting class is the other
	if originalReg == NO_CLASS {
		return newReg
	}
	if newReg == NO_CLASS {
		return originalReg
	}

	// (c) If one of the classes is MEMORY, the result is the MEMORY class.
	if newReg == MEMORY || originalReg == MEMORY {
		return MEMORY
	}

	// (d) If one of the classes is INTEGER, the result is the INTEGER.
	if newReg == INTEGER || originalReg == INTEGER {
		return INTEGER
	}

	// (e) If one of the classes is X87, X87UP, COMPLEX_X87 class, MEMORY is used as class.
	if newReg == X87 || newReg == X87UP || newReg == COMPLEX_X87 {
		return MEMORY
	}
	if originalReg == X87 || originalReg == X87UP || originalReg == COMPLEX_X87 {
		return MEMORY
	}

	// (f) Otherwise class SSE is used.
	return SSE
}

// post_merge Page 22 AMD64 ABI point 5 - this is the most merger "cleanup"
func postMerge(lo *RegisterClass, hi *RegisterClass, size int64) {

	// (a) If one of the classes is MEMORY, the whole argument is passed in memory.
	if (*lo) == MEMORY || (*hi) == MEMORY {
		(*lo) = MEMORY
		(*hi) = MEMORY
	}

	// (b) If X87UP is not preceded by X87, the whole argument is passed in memory.
	if (*hi) == X87UP && (*lo) != X87 {
		(*lo) = MEMORY
		(*hi) = MEMORY
	}

	// (c) If the size of the aggregate exceeds two eightbytes and the first eight- byte isn’t SSE
	// or any other eightbyte isn’t SSEUP, the whole argument is passed in memory.
	if size > 128 && ((*lo) != SSE || (*hi) != SSEUP) {
		(*lo) = MEMORY
		(*hi) = MEMORY
	}

	// (d) If SSEUP is // not preceded by SSE or SSEUP, it is converted to SSE.
	if (*hi) == SSEUP && ((*lo) != SSE && (*lo) != SSEUP) {
		(*hi) = SSE
	}
}

// ClassifyFunction classifies a function type
func ClassifyFunction(t *dwarf.FuncType, c *file.Component, ptrCount *int64) Classification {
	if (*ptrCount) > 0 {
		return ClassifyPointer(ptrCount)
	}
	return Classification{}
}

// ClassifyEnum classifies an enum type
func ClassifyEnum(t *dwarf.EnumType, c *file.Component, ptrCount *int64) Classification {
	return Classification{Lo: INTEGER, Hi: INTEGER, Name: "Enum"}
}

// ClassifyType takes a general type to classify
func ClassifyType(c *file.Component, ptrCount *int64) Classification {

	if (*ptrCount) > 0 {
		return ClassifyPointer(ptrCount)
	}

	switch c.Class {
	case "Function":
		convert := c.RawType.(*dwarf.FuncType)
		return ClassifyFunction(convert, c, ptrCount)

	case "Array":
		convert := c.RawType.(*dwarf.ArrayType)
		return ClassifyArray(convert, c, ptrCount)

	case "Enum":
		convert := c.RawType.(*dwarf.EnumType)
		return ClassifyEnum(convert, c, ptrCount)

	// TODO need to add scalar type (which are they?)
	// This case actually handles struct, union, and class
	case "Struct":
		convert := c.RawType.(*dwarf.StructType)
		return ClassifyStruct(convert, c, ptrCount)
	default:
		fmt.Println(c.Class)
	}

	/*
	   auto [underlying_type, ptr_cnt] = unwrap_underlying_type(fieldType);

	   if (ptr_cnt > 0) {
	     return classify_pointer(ptr_cnt);
	   }

	   if (auto *t = underlying_type->getScalarType()) {
	     return classify(t);
	   }
	 }*/
	return Classification{Lo: NO_CLASS, Hi: NO_CLASS, Name: "Unknown"}
}

/*
namespace smeagle::x86_64 {


  inline classification classify(st::typeScalar *t) {
    // paramType properties have booleans to indicate types
    auto const &props = t->properties();

    // size in BITS
    const auto size = t->getSize() * 8;

    // Integral types
    if (props.is_integral || props.is_UTF) {
      if (size > 128) {
        return {RegisterClass::SSE, RegisterClass::SSEUP, "IntegerVec"};
      }
      if (size == 128) {
        // __int128 is treated as struct{long,long};
        // This is NOT correct, but we don't handle aggregates yet.
        // How do we differentiate between __int128 and __m128i?
        return {RegisterClass::MEMORY, RegisterClass::NO_CLASS, "Integer"};
      }

      // _Decimal32, _Decimal64, and __m64 are supposed to be SSE.
      // TODO How can we differentiate them here?
      return {RegisterClass::INTEGER, RegisterClass::NO_CLASS, "Integer"};
    }

    if (props.is_floating_point) {
      if (props.is_complex_float) {
        if (size == 128) {
          // x87 `complex long double`
          return {RegisterClass::COMPLEX_X87, RegisterClass::NO_CLASS, "CplxFloat"};
        }
        // This is NOT correct.
        // TODO It should be struct{T r,i;};, but we don't handle aggregates yet
        return {RegisterClass::MEMORY, RegisterClass::NO_CLASS, "CplxFloat"};
      }
      if (size <= 64) {
        // 32- or 64-bit floats
        return {RegisterClass::SSE, RegisterClass::SSEUP, "Float"};
      }
      if (size == 128) {
        // x87 `long double` OR __m128[d]
        // TODO: How do we differentiate the vector type here? Dyninst should help us
        return {RegisterClass::X87, RegisterClass::X87UP, "Float"};
      }
      if (size > 128) {
        return {RegisterClass::SSE, RegisterClass::SSEUP, "FloatVec"};
      }
    }

    // TODO we will eventually want to throw this
    // throw std::runtime_error{"Unknown scalar type"};
    return {RegisterClass::NO_CLASS, RegisterClass::NO_CLASS, "Unknown"};
  }



  // Classify the fields
  std::vector<classification> classify_fields(st::typeStruct *t) {
    std::vector<classification> classes;
    for (auto *f : *t->getFields()) {
      classes.push_back(classify(f));
    }
    return classes;
  }


  // Classify a single field
  classification classify(st::Field *f) {
    // Just classify the type of the field
    return classify_type(f->getType());
  }

}  // namespace smeagle::x86_64
*/
