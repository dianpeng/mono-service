package pl

import (
	"bytes"
	"fmt"
	"reflect"
)

// this is a simple prototype checking mechanism to simplify the go side
// pl function binding. It gives user a simple DSL to define the type of
// the function prototype and the structure will verify the argument for
// the user and report any error if needed
//
// The descriptor is similar to printf flag with some simple extension
//
// %d -> integer
// %u -> unsigned integer
// %f -> real
// %F -> positive real number, including 0
// %s -> string
// %S -> none empty string
// %b -> boolean
// %Y -> true
// %N -> false
// %n -> null
// %m -> map
// %l -> list
// %p -> pair
// %r -> regex
// %U -> user
//   user can optionally have a type id enclude with '[' and ']'
// %c -> closure
// %C -> Script closure
// %R -> native closure
// %M -> method closure
// %a -> any type
// %0 -> no arguments
// %- -> always pass, do nothing

// each descriptor can be written as OR to indicate different types are allowed
// by grouping them with parenthesis and '|'
// example: %d(%s|%n).

// This example shows that a function accept 2 arguments and one is integer and
// the other can be a string or a null

// for function overload, ie different # of arguments with different types, user
// can specify different configuration by groupping each one into a {...}
// example: {%d%s}{%d%n%n} this function can accept 2 types of argument, one is
// 2 arguments, and the other allows 3 arguments

// lastly, the last element can optionally follows an * which indicates the
// variable length arguments. For example, %d%m*
// means the function accepts at least 2 arguments, one is integer, and the rest
// are all maps

const (
	PInt = iota
	// for reflection usage
	PI8
	PI16
	PI32
	PI64

	PUInt
	// for reflection usage
	PUI8
	PUI16
	PUI32
	PUI64

	PReal

	// for reflection usage
	PR32
	PR64

	PUReal
	PUR32
	PUR64

	PString
	PNEString
	PBool
	PTrue
	PFalse
	PNull
	PMap
	PList
	PPair
	PRegexp
	PUsr
	PClosure
	PNFunc
	PSFunc
	PMFunc
	PAny
	PNone
)

func mapToObjType(ptype int) int {
	switch ptype {
	case PInt, PI8, PI16, PI32, PI64, PUInt, PUI8, PUI16, PUI32, PUI64:
		return ValInt
	case PReal, PR32, PR64, PUReal, PUR32, PUR64:
		return ValReal
	case PString, PNEString:
		return ValStr
	case PBool, PTrue, PFalse:
		return ValBool
	case PNull:
		return ValNull
	case PMap:
		return ValMap
	case PList:
		return ValList
	case PPair:
		return ValPair
	case PRegexp:
		return ValRegexp
	case PClosure, PNFunc, PSFunc, PMFunc:
		return ValClosure
	case PUsr:
		return ValUsr
	default:
		return -1
	}
}

func mapFromObjType(otype int) opc {
	switch otype {
	case ValInt:
		return opc(PInt)
	case ValReal:
		return opc(PReal)
	case ValStr:
		return opc(PString)
	case ValBool:
		return opc(PBool)
	case ValNull:
		return opc(PNull)
	case ValPair:
		return opc(PPair)
	case ValList:
		return opc(PList)
	case ValMap:
		return opc(PMap)
	case ValRegexp:
		return opc(PRegexp)
	default:
		return opc(PUsr)
	}
}

type opc int

func (o *opc) isInt() bool {
	switch *o {
	case PInt, PI8, PI16, PI32, PI64:
		return true
	default:
		return false
	}
}

func (o *opc) isUInt() bool {
	switch *o {
	case PUInt, PUI8, PUI16, PUI32, PUI64:
		return true
	default:
		return false
	}
}

func (o *opc) isReal() bool {
	switch *o {
	case PReal, PR32, PR64:
		return true
	default:
		return false
	}
}

func (o *opc) isUReal() bool {
	switch *o {
	case PUReal, PUR32, PUR64:
		return true
	default:
		return false
	}
}

type protoelem struct {
	opcode opc
	tname  string
}

type argp struct {
	or []protoelem
}

type argpcase struct {
	d      []argp
	noarg  bool
	varlen bool
}

type FuncProto struct {
	Descriptor string
	Name       string

	// internal fields
	d          []*argpcase
	noarg      bool
	alwayspass bool
	varlen     bool
}

func (p *protoelem) str() string {
	switch p.opcode {
	case PInt:
		return "int"
	case PI8:
		return "int8"
	case PI16:
		return "int16"
	case PI32:
		return "int32"
	case PI64:
		return "int64"
	case PUInt:
		return "uint"
	case PUI8:
		return "uint8"
	case PUI16:
		return "uint16"
	case PUI32:
		return "uint32"
	case PUI64:
		return "uint64"
	case PReal:
		return "real"
	case PR32:
		return "float32"
	case PR64:
		return "float64"
	case PUReal:
		return "postive_real"
	case PUR32:
		return "float32"
	case PUR64:
		return "float64"

	case PString:
		return "string"
	case PNEString:
		return "none_empty_string"
	case PBool:
		return "bool"
	case PTrue:
		return "true"
	case PFalse:
		return "false"
	case PNull:
		return "null"
	case PMap:
		return "map"
	case PList:
		return "list"
	case PPair:
		return "pair"
	case PRegexp:
		return "regexp"
	case PClosure:
		return "closure"
	case PNFunc:
		return "native_function"
	case PSFunc:
		return "script_function"
	case PMFunc:
		return "method_function"
	case PUsr:
		return fmt.Sprintf("user[%s]", p.tname)
	case PAny:
		return "any"
	default:
		return "none"
	}
}

func (p *argp) str() string {
	if len(p.or) == 0 {
		return ""
	} else if len(p.or) == 1 {
		return p.or[0].str()
	} else {
		b := new(bytes.Buffer)
		b.WriteString(p.or[0].str())

		sz := len(p.or)
		for i := 1; i < sz; i++ {
			b.WriteRune('|')
			b.WriteString(p.or[i].str())
		}
		return b.String()
	}
}

func (p *argpcase) str() string {
	b := new(bytes.Buffer)
	for idx, e := range p.d {
		b.WriteString(fmt.Sprintf("%d@(", idx+1))
		b.WriteString(e.str())
		b.WriteRune(')')
	}
	if p.varlen {
		b.WriteRune('*')
	}
	return b.String()
}

func (p *FuncProto) Dump() string {
	b := new(bytes.Buffer)
	b.WriteString(fmt.Sprintf("descriptor> %s\n", p.Descriptor))
	b.WriteString(fmt.Sprintf("name> %s\n", p.Name))
	b.WriteString("bytecode>\n")
	for idx, a := range p.d {
		b.WriteString(fmt.Sprintf("%d> %s\n", idx+1, a.str()))
	}
	return b.String()
}

func (f *FuncProto) scanTName(rList []rune, start int) (int, string, error) {
	bc := rList[start]
	if bc != '\'' && bc != '"' {
		return -1, "", fmt.Errorf("expect ' or \" to indicate type name")
	}
	sz := len(rList)
	b := new(bytes.Buffer)
	for start++; start < sz; start++ {
		c := rList[start]
		if c == bc {
			start++
			return start, b.String(), nil
		} else if c == '\\' {
			// escape sequences, we just support ' and " as escape sequences, any
			// other cannot be represented as type
			if start+1 == sz {
				return -1, "", fmt.Errorf("invalid escape sequences")
			}
			nc := rList[start+1]
			switch nc {
			case '\\':
				b.WriteRune('\\')
				break
			case '"':
				b.WriteRune('"')
				break
			case '\'':
				b.WriteRune('\'')
				break
			default:
				return -1, "", fmt.Errorf("invalid escape sequences")
			}

			start++
			continue
		} else {
			b.WriteRune(c)
		}
	}

	return -1, "", fmt.Errorf("string is not closed by quote")
}

func (f *FuncProto) compT(rList []rune, cursor int, vlen *bool) (int, protoelem, error) {
	c := rList[cursor]
	sz := len(rList)

	if c != '%' {
		return -1, protoelem{}, fmt.Errorf("expect %% to start type descriptor")
	}
	if cursor+1 == sz {
		return -1, protoelem{}, fmt.Errorf("dangling %%")
	}

	nc := rList[cursor+1]

	var opcode int
	var tname string

	switch nc {
	case 'd', 'i':
		opcode = PInt
		break
	case 'u':
		opcode = PUInt
		break
	case 'f':
		opcode = PReal
		break
	case 'F':
		opcode = PUReal
		break
	case 's':
		opcode = PString
		break
	case 'S':
		opcode = PNEString
		break
	case 'b':
		opcode = PBool
		break
	case 'Y':
		opcode = PTrue
		break
	case 'N':
		opcode = PFalse
		break
	case 'n':
		opcode = PNull
		break
	case 'm':
		opcode = PMap
		break
	case 'p':
		opcode = PPair
		break
	case 'l':
		opcode = PList
		break
	case 'r':
		opcode = PRegexp
		break
	case 'U':
		opcode = PUsr
		break
	case 'c':
		opcode = PClosure
		break
	case 'C':
		opcode = PSFunc
		break
	case 'R':
		opcode = PNFunc
		break
	case 'M':
		opcode = PMFunc
		break
	case 'a':
		opcode = PAny
		break
	case '0':
		opcode = PNone
		break
	case '-':
		return -1, protoelem{}, fmt.Errorf("%%- cannot have any other leading type descriptors")
	default:
		return -1, protoelem{}, fmt.Errorf("unknown type descriptor %c", nc)
	}

	cursor += 2

	// now parsing the trailing part which may or may not have
	if cursor < sz {
		nnc := rList[cursor]

		switch nnc {
		case '[':
			if opcode != PUsr {
				return -1, protoelem{},
					fmt.Errorf("only %%U(user type) can have optional type name")
			}
			cursor++
			if cursor == sz {
				return -1, protoelem{}, fmt.Errorf("unfinished optinoal type name")
			}
			if rList[cursor] != ']' {
				newPos, name, err := f.scanTName(rList, cursor)
				if err != nil {
					return -1, protoelem{}, err
				}
				tname = name
				cursor = newPos
			}

			if rList[cursor] != ']' {
				return -1, protoelem{}, fmt.Errorf("type name must be closed by ]")
			}

			cursor++
			break

		case '1', '2', '4', '8':
			var offset int

			switch nnc {
			case '1':
				offset = 1
				break
			case '2':
				offset = 2
				break
			case '4':
				offset = 3
				break
			default:
				offset = 4
				break
			}

			// possibly the i1, i2, i4, i8 (1, 2, 4, 8 rep the byte count)
			switch opcode {
			case PInt, PUInt:
				opcode += offset
				break
			case PReal:
			case PUReal:
				if offset == 1 || offset == 2 {
					return -1, protoelem{}, fmt.Errorf("real number can only have 4 or 8 byte count mark")
				}
				opcode += offset - 2
				break
			default:
				return -1, protoelem{}, fmt.Errorf("only int, uint, real, ureal can have byte count mark")
			}

			cursor++
			break

		default:
			break
		}

		if cursor < sz {
			nnc = rList[cursor]
		} else {
			// assign a char will not be used as decorator
			nnc = ' '
		}

		// always put * check at very last since it can follow multiple optional
		// suffix after the type descriptor, ie $U[foo.bar]* is allowed, means
		// any numbers of user context with type name foo.bar
		if nnc == '*' {
			*vlen = true

			if (cursor+1 < sz && rList[cursor+1] != '}') &&
				(cursor+1 != sz) {
				return -1, protoelem{}, fmt.Errorf("nothing should be expeceted after %%%c*", nc)
			}
			cursor++
		}
	}

	return cursor, protoelem{
		opcode: opc(opcode),
		tname:  tname,
	}, nil
}

func (f *FuncProto) compCase(d string) (*argpcase, int, error) {
	cursor := 0
	rList := []rune(d)
	sz := len(rList)
	argpc := &argpcase{}
	hasnoarg := false

LOOP:
	for cursor < sz {
		c := rList[cursor]
		switch c {
		case '}':
			cursor++
			break LOOP

		case '%':
			if hasnoarg {
				return nil, -1, fmt.Errorf("%%0 shows up inside of the format list, the " +
					"can just contain exactly one %%0")
			}

			cc, pelem, err := f.compT(rList, cursor, &argpc.varlen)
			if err != nil {
				return nil, -1, err
			}
			cursor = cc

			// special cases that %0 shows up inside of the argpcase and it should
			// just stop at the position
			if pelem.opcode == PNone {
				if len(argpc.d) != 0 {
					return nil, -1, fmt.Errorf("%%0 shows up inside of the format list, the " +
						"can just contain exactly one %%0")
				}
				hasnoarg = true
				argpc.noarg = true
				// notes we do not push PNone into the type list
			} else {
				argpc.d = append(argpc.d, argp{
					or: []protoelem{pelem},
				})
			}

			break

		case '(':
			cursor++

			var orlist []protoelem
		GROUP:
			for {
				if cursor == sz {
					return nil, -1, fmt.Errorf("not closed type group")
				}
				var vlen bool
				cc, pelem, err := f.compT(rList, cursor, &vlen)
				if err != nil {
					return nil, -1, err
				}
				if pelem.opcode == PNone {
					return nil, -1, fmt.Errorf("%%0 cannot show up in type OR list")
				}

				if vlen {
					// notes, we don't support having OR group comes with type descriptor
					// decorated by any decorator. This makes the searching algorithm none
					// trivial to perform and require backtrace to implement
					return nil, -1, fmt.Errorf("or list cannot have type modifier with var length, " +
						"try to rewrite it with different group")
				}
				cursor = cc
				orlist = append(orlist, pelem)

				char := rList[cursor]

				switch char {
				case '|':
					cursor++
					break

				case ')':
					cursor++
					break GROUP

				default:
					return nil, -1, fmt.Errorf("unknown token %c", char)
				}
			}

			argpc.d = append(argpc.d, argp{
				or: orlist,
			})
			break

		default:
			return nil, -1, fmt.Errorf("unknown token %c, expect %% or (", c)
		}
	}
	return argpc, cursor, nil
}

func (f *FuncProto) compile(d string) error {
	if len(d) == 0 || d == "%0" {
		f.noarg = true
		return nil
	} else if d == "%-" {
		f.alwayspass = true
		return nil
	}

	rlist := []rune(d)
	cursor := 0
	sz := len(rlist)

	for cursor < sz {
		c := rlist[cursor]
		if c == '{' {
			cursor++
		}
		ele, cs, err := f.compCase(string(rlist[cursor:]))
		if err != nil {
			return err
		}
		f.d = append(f.d, ele)
		cursor += cs
	}
	return nil
}

type convoneval func(Val, opc)
type convsliceval func(*reflect.Value)

func (f *FuncProto) check0(exp protoelem, got Val, c convoneval) (the_return bool) {
	defer func() {
		if c != nil && the_return {
			c(got, exp.opcode)
		}
	}()

	if exp.opcode == PAny {
		return true
	}

	switch got.Type {
	case ValInt:
		if exp.opcode.isInt() {
			return true
		} else if exp.opcode.isUInt() {
			return got.Int() >= 0
		}
		break

	case ValReal:
		if exp.opcode.isReal() {
			return true
		} else if exp.opcode.isUReal() {
			return got.Real() >= 0.0
		}
		break

	case ValStr:
		if exp.opcode == PString {
			return true
		} else if exp.opcode == PNEString {
			return len(got.String()) != 0
		}
		break

	case ValBool:
		if exp.opcode == PBool {
			return true
		} else if exp.opcode == PTrue {
			return got.Bool()
		} else if exp.opcode == PFalse {
			return !got.Bool()
		}
		break

	case ValNull:
		if exp.opcode == PNull {
			return true
		}
		break

	case ValPair:
		if exp.opcode == PPair {
			return true
		}
		break

	case ValList:
		if exp.opcode == PList {
			return true
		}
		break

	case ValMap:
		if exp.opcode == PMap {
			return true
		}
		break

	case ValRegexp:
		if exp.opcode == PRegexp {
			return true
		}
		break

	case ValClosure:
		if exp.opcode == PClosure {
			return true
		}
		if exp.opcode == PNFunc {
			return got.Closure().Type() == ClosureNative
		}
		if exp.opcode == PSFunc {
			return got.Closure().Type() == ClosureScript
		}
		if exp.opcode == PMFunc {
			return got.Closure().Type() == ClosureMethod
		}
		return false

	case ValIter:
		return false

	default:
		if exp.opcode == PUsr {
			if exp.tname != "" {
				return exp.tname == got.Id()
			} else {
				return true
			}
		}
		break
	}
	return false
}

func (f *FuncProto) check1(index int, exp *argp, got Val, conv convoneval) error {
	for _, c := range exp.or {
		if f.check0(c, got, conv) {
			return nil
		}
	}

	return fmt.Errorf("%d'th argument's type is invalided, "+
		"type input is %s but what we expect is %s",
		index+1, got.Info(), exp.str())
}

func (f *FuncProto) doCheck(d *argpcase, args []Val, conv convsliceval) (int, error) {
	alen := len(args)
	elen := len(d.d)

	if d.noarg {
		if alen != 0 {
			return -1, fmt.Errorf("function(method) call: %s expect 0 arguments, but got %d",
				f.Name, alen)
		}
		return 0, nil
	}

	if alen < elen {
		if d.varlen && alen == elen-1 {
			// this is the special case, ie
			// %s%a* allows at least 1 argument to represent the first string, and
			// the second %a is an optional, ie we can have zero optional arguments
			// example like format function
		} else {
			prefix := ""
			if d.varlen {
				prefix = " at least "
			}
			return alen, fmt.Errorf("function(method) call: %s expects%s%d arguments, "+
				"but got %d arguments", f.Name, prefix, elen, alen)
		}
	}

	// checking variable type one by one
	if conv != nil {
		for idx, a := range d.d {
			if idx == alen {
				break
			}

			arg := args[idx]
			var out *reflect.Value
			outp := &out

			if err := f.check1(idx, &a, arg, func(v Val, t opc) {
				*outp = f.pack(v, t)
			}); err != nil {
				return alen, err
			}

			if out == nil {
				return alen, fmt.Errorf("function(method) call: %s, %dth argument "+
					"is invalid during reflection conversion",
					f.Name, idx)
			} else {
				conv(out)
			}
		}
	} else {
		for idx, a := range d.d {
			if idx == alen {
				break
			}

			arg := args[idx]
			if err := f.check1(idx, &a, arg, nil); err != nil {
				return alen, err
			}
		}
	}

	// checking dangling cases, ie variable length
	if elen < alen {
		if !d.varlen {
			return 0, fmt.Errorf("function(method) call: %s expects %d argument, "+
				"but got %d", f.Name, elen, alen)
		}
		danglingE := &d.d[elen-1]

		if conv != nil {
			for i := elen; i < alen; i++ {
				var out *reflect.Value
				outp := &out

				if err := f.check1(i, danglingE, args[i], func(v Val, t opc) {
					*outp = f.pack(v, t)
				}); err != nil {
					return alen, err
				}

				if out == nil {
					return alen, fmt.Errorf("function(method) call: %s, %dth argument "+
						"is invalid during reflection conversion",
						f.Name, i+1)
				} else {
					conv(out)
				}
			}
		} else {
			for i := elen; i < alen; i++ {
				if err := f.check1(i, danglingE, args[i], nil); err != nil {
					return alen, err
				}
			}
		}
	}

	return alen, nil
}

func (f *FuncProto) Check(args []Val) (int, error) {
	alen := len(args)

	if f.alwayspass {
		return alen, nil
	}
	if f.noarg {
		if alen == 0 {
			return 0, nil
		} else {
			return alen,
				fmt.Errorf("function(method) call: %s expects no argument, "+
					"but got %d arguments", f.Name, alen)
		}
	}
	var err error
	found := false

	for _, c := range f.d {
		sz := len(c.d)
		if sz == alen || (sz < alen && c.varlen) {
			_, err = f.doCheck(c, args, nil)
			if err == nil {
				found = true
				break
			}
		}
	}

	if found {
		return alen, err
	} else {
		return alen, fmt.Errorf("function(method) call: %s invalid arguments, "+
			"no matching argument type can be found", f.Name)
	}
}

// return nil when we cannot convert Val into reflect.Value (ie not supported)
func (f *FuncProto) pack(v Val, t opc) *reflect.Value {
	var rv reflect.Value
	switch int(t) {
	case PInt:
		rv = reflect.ValueOf(int(v.Int()))
		break

	case PI8:
		rv = reflect.ValueOf(int8(v.Int()))
		break

	case PI16:
		rv = reflect.ValueOf(int16(v.Int()))
		break

	case PI32:
		rv = reflect.ValueOf(int32(v.Int()))
		break

	case PI64:
		rv = reflect.ValueOf(int64(v.Int()))
		break

	case PUInt:
		rv = reflect.ValueOf(uint(v.Int()))
		break

	case PUI8:
		rv = reflect.ValueOf(uint8(v.Int()))
		break

	case PUI16:
		rv = reflect.ValueOf(uint16(v.Int()))
		break

	case PUI32:
		rv = reflect.ValueOf(uint32(v.Int()))
		break

	case PUI64:
		rv = reflect.ValueOf(uint64(v.Int()))
		break

	case PReal, PUReal:
		rv = reflect.ValueOf(v.Real())
		break

	case PR32, PUR32:
		rv = reflect.ValueOf(float32(v.Real()))
		break

	case PR64, PUR64:
		rv = reflect.ValueOf(float64(v.Real()))
		break

	case PString, PNEString:
		rv = reflect.ValueOf(v.String())
		break

	case PBool, PTrue, PFalse:
		rv = reflect.ValueOf(v.Bool())
		break

	case PNull:
		rv = reflect.ValueOf(nil)
		break

	case PMap, PPair, PList:
		return nil

	case PClosure, PSFunc, PNFunc:
		return nil

	case PUsr:
		rv = reflect.ValueOf(v.Usr())
		break

	case PRegexp:
		rv = reflect.ValueOf(v.Regexp())
		break

	case PAny:
		return f.pack(v, mapFromObjType(v.Type))

	default:
		return nil
	}

	return &rv
}

// conversion from Val to reflect.Value.
func (f *FuncProto) Pack(args []Val) ([]reflect.Value, error) {
	alen := len(args)

	if f.alwayspass {
		var output []reflect.Value
		for _, a := range args {
			output = append(output, *f.pack(a, PAny))
		}
		return output, nil
	}

	if f.noarg {
		if alen == 0 {
			return nil, nil
		} else {
			return nil,
				fmt.Errorf("function(method) call: %s expects no argument, "+
					"but got %d arguments", f.Name, alen)
		}
	}

	var err error
	var output []reflect.Value
	found := false

	for _, c := range f.d {
		sz := len(c.d)
		if sz == alen || (sz < alen && c.varlen) {
			var oput []reflect.Value

			if _, err = f.doCheck(c, args, func(o *reflect.Value) {
				oput = append(oput, *o)
			}); err == nil {
				found = true
				output = oput
				break
			}
		}
	}

	if found {
		return output, err
	} else {
		return nil, fmt.Errorf("function(method) call: %s invalid arguments, "+
			"no matching argument type can be found", f.Name)
	}
}

func NewFuncProto(name string, descriptor string) (*FuncProto, error) {
	f := &FuncProto{
		Descriptor: descriptor,
		Name:       name,
	}
	if err := f.compile(descriptor); err != nil {
		return nil, err
	}
	return f, nil
}

func NewModFuncProto(mod string, fname string, descriptor string) (*FuncProto, error) {
	return NewFuncProto(modFuncName(mod, fname), descriptor)
}

func MustNewFuncProto(name string, d string) *FuncProto {
	f, err := NewFuncProto(name, d)
	if err != nil {
		musterr("MustNewFuncProto", err)
	}
	return f
}

func MustNewModFuncProto(mod string, f string, d string) *FuncProto {
	pp, err := NewModFuncProto(mod, f, d)
	if err != nil {
		musterr("MustNewModFuncProto", err)
	}
	return pp
}
