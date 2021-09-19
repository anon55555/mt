package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

var (
	pkg *packages.Package

	serializeFmt   = make(map[string]string)
	deserializeFmt = make(map[string]string)

	uint8T = types.Universe.Lookup("uint8").Type()
	byteT  = types.Universe.Lookup("byte").Type()

	serialize   []*types.Named
	inSerialize = make(map[string]bool)

	consts = make(map[*ast.StructType][]*ast.Comment)
)

func structPragma(c *ast.Comment, sp *[]func(), expr string, de bool) {
	fields := strings.SplitN(strings.TrimPrefix(c.Text, "//mt:"), " ", 2)
	arg := ""
	if len(fields) == 2 {
		arg = fields[1]
	}
	switch fields[0] {
	case "const":
		tv, err := types.Eval(pkg.Fset, pkg.Types, c.Slash, arg)
		if err != nil {
			error(c.Pos(), err)
		}

		if de {
			fmt.Println("{")
			x := newVar()
			fmt.Println("var", x, typeStr(tv.Type))
			y := newVar()
			fmt.Println(y, ":=", arg)
			genSerialize(tv.Type, x, token.NoPos, nil, de)
			fmt.Println("if", x, "!=", y,
				`{ chk(fmt.Errorf("const %v: %v",`, strconv.Quote(arg), ",", x, ")) }")
			fmt.Println("}")
		} else {
			v := newVar()
			fmt.Println("{", v, ":=", arg)
			genSerialize(tv.Type, v, c.Slash+token.Pos(len("//mt:const ")), nil, de)
			fmt.Println("}")
		}
	case "assert":
		fmt.Printf("if !("+arg+") {", expr)
		fmt.Printf("chk(errors.New(%q))\n", "assertion failed: "+arg)
		fmt.Println("}")
	case "zlib":
		if de {
			fmt.Println("{ r, err := zlib.NewReader(byteReader{r}); chk(err)")
			*sp = append(*sp, func() {
				fmt.Println("chk(r.Close()) }")
			})
		} else {
			fmt.Println("{ w := zlib.NewWriter(w)")
			*sp = append(*sp, func() {
				fmt.Println("chk(w.Close()) }")
			})
		}
	case "lenhdr":
		if arg != "8" && arg != "16" && arg != "32" {
			error(c.Pos(), "usage: //mt:lenhdr (8|16|32)")
		}

		fmt.Println("{")

		if !de {
			fmt.Println("ow := w")
			fmt.Println("w := new(bytes.Buffer)")
		}

		var cg ast.CommentGroup
		if de {
			t := types.Universe.Lookup("uint" + arg).Type()
			fmt.Println("var n", t)
			genSerialize(t, "n", token.NoPos, nil, de)
			if arg == "64" {
				fmt.Println(`if n > math.MaxInt64 { panic("too big len") }`)
			}
			fmt.Println("r := &io.LimitedReader{R: r, N: int64(n)}")
		} else {
			switch arg {
			case "8", "32":
				cg.List = []*ast.Comment{{Text: "//mt:len" + arg}}
			case "16":
			}
		}

		*sp = append(*sp, func() {
			if de {
				fmt.Println("if r.N > 0",
					`{ chk(fmt.Errorf("%d bytes of trailing data", r.N)) }`)
			} else {
				fmt.Println("{")
				fmt.Println("buf := w")
				fmt.Println("w := ow")
				byteSlice := types.NewSlice(types.Typ[types.Byte])
				genSerialize(byteSlice, "buf.Bytes()", token.NoPos, &cg, de)
				fmt.Println("}")
			}

			fmt.Println("}")
		})
	case "end":
		(*sp)[len(*sp)-1]()
		*sp = (*sp)[:len(*sp)-1]
	case "if":
		fmt.Printf(strings.TrimPrefix(c.Text, "//mt:")+" {\n", expr)
		*sp = append(*sp, func() {
			fmt.Println("}")
		})
	case "ifde":
		if !de {
			fmt.Println("/*")
		}
	}
}

func genSerialize(t types.Type, expr string, pos token.Pos, doc *ast.CommentGroup, de bool) {
	var lenhdr types.Type = types.Typ[types.Uint16]

	useMethod := true
	if doc != nil {
		for _, c := range doc.List {
			pragma := true
			switch c.Text {
			case "//mt:32to16":
				t = types.Typ[types.Int16]
				if de {
					v := newVar()
					fmt.Println("var", v, "int16")
					defer fmt.Println(expr + " = int32(" + v + ")")
					expr = v
				} else {
					expr = "int16(" + expr + ")"
				}
				pos = token.NoPos
			case "//mt:32tou16":
				t = types.Typ[types.Uint16]
				if de {
					v := newVar()
					fmt.Println("var", v, "uint16")
					defer fmt.Println(expr + " = int32(" + v + ")")
					expr = v
				} else {
					expr = "uint16(" + expr + ")"
				}
				pos = token.NoPos
			case "//mt:utf16":
				t = types.NewSlice(types.Typ[types.Uint16])
				if de {
					v := newVar()
					fmt.Println("var", v, typeStr(t))
					defer fmt.Println(expr + " = string(utf16.Decode(" + v + "))")
					expr = v
				} else {
					v := newVar()
					fmt.Println(v, ":= utf16.Encode([]rune("+expr+"))")
					expr = v
				}
				pos = token.NoPos
			case "//mt:raw":
				lenhdr = nil
			case "//mt:len8":
				lenhdr = types.Typ[types.Uint8]
			case "//mt:len32":
				lenhdr = types.Typ[types.Uint32]
			case "//mt:opt":
				fmt.Println("if err := pcall(func() {")
				defer fmt.Println("}); err != nil && err != io.EOF",
					"{ chk(err) }")
			default:
				pragma = false
			}
			if pragma {
				useMethod = false
			}
		}
	}

	str := types.TypeString(t, types.RelativeTo(pkg.Types))
	if de {
		if or, ok := deserializeFmt[str]; ok {
			fmt.Println("{")
			fmt.Println("p := &" + expr)
			fmt.Print(or)
			fmt.Println("}")
			return
		}
	} else {
		if or, ok := serializeFmt[str]; ok {
			fmt.Println("{")
			fmt.Println("x := " + expr)
			fmt.Print(or)
			fmt.Println("}")
			return
		}
	}

	expr = "(" + expr + ")"

	switch t := t.(type) {
	case *types.Named:
		if !useMethod {
			t := t.Underlying()
			genSerialize(t, "*(*"+typeStr(t)+")("+"&"+expr+")", pos, doc, de)
			return
		}

		method := "Serialize"
		if de {
			method = "Deserialize"
		}
		for i := 0; i < t.NumMethods(); i++ {
			m := t.Method(i)
			if m.Name() == method {
				rw := "w"
				if de {
					rw = "r"
				}
				fmt.Println("chk(" + expr + "." + method + "(" + rw + "))")
				return
			}
		}

		mkSerialize(t)

		fmt.Println("if err := pcall(func() {")
		if de {
			fmt.Println(expr + ".deserialize(r)")
		} else {
			fmt.Println(expr + ".serialize(w)")
		}
		fmt.Println("}); err != nil",
			`{`,
			`if err == io.EOF { chk(io.EOF) };`,
			`chk(fmt.Errorf("%s: %w", `+strconv.Quote(t.String())+`, err))`,
			`}`)
	case *types.Struct:
		st := pos2node(pos)[0].(*ast.StructType)

		a := consts[st]
		b := st.Fields.List

		// Merge sorted slices.
		c := make([]ast.Node, 0, len(a)+len(b))
		for i, j := 0, 0; i < len(a) || j < len(b); {
			if i < len(a) && (j >= len(b) || a[i].Pos() < b[j].Pos()) {
				c = append(c, a[i])
				i++
			} else {
				c = append(c, b[j])
				j++
			}
		}

		var (
			stk []func()
			i   int
		)
		for _, field := range c {
			switch field := field.(type) {
			case *ast.Comment:
				structPragma(field, &stk, expr, de)
			case *ast.Field:
				n := len(field.Names)
				if n == 0 {
					n = 1
				}
				for ; n > 0; n-- {
					f := t.Field(i)
					genSerialize(f.Type(), expr+"."+f.Name(), field.Type.Pos(), field.Doc, de)
					i++
				}
			}
		}

		if len(stk) > 0 {
			error(pos, "missing //mt:end")
		}
	case *types.Basic:
		switch t.Kind() {
		case types.String:
			byteSlice := types.NewSlice(types.Typ[types.Byte])
			if de {
				v := newVar()
				fmt.Println("var", v, byteSlice)
				genSerialize(byteSlice, v, token.NoPos, doc, de)
				fmt.Println(expr, "=", "string(", v, ")")
			} else {
				genSerialize(byteSlice, "[]byte"+expr, token.NoPos, doc, de)
			}
		default:
			error(pos, "can't serialize ", t)
		}
	case *types.Slice:
		if de {
			if lenhdr != nil {
				v := newVar()
				fmt.Println("var", v, lenhdr)
				genSerialize(lenhdr, v, pos, nil, de)
				fmt.Printf("%s = make(%v, %s)\n",
					expr, typeStr(t), v)
				genSerialize(types.NewArray(t.Elem(), 0), expr, pos, nil, de)
			} else {
				if b, ok := t.Elem().(*types.Basic); ok && b.Kind() == types.Byte {
					fmt.Println("{")
					fmt.Println("var err error")
					fmt.Println(expr, ", err = io.ReadAll(r)")
					fmt.Println("chk(err)")
					fmt.Println("}")
					return
				}

				fmt.Println("for {")
				v := newVar()
				fmt.Println("var", v, typeStr(t.Elem()))
				fmt.Println("err := pcall(func() {")
				if pos.IsValid() {
					pos = pos2node(pos)[0].(*ast.ArrayType).Elt.Pos()
				}
				genSerialize(t.Elem(), v, pos, nil, de)
				fmt.Println("})")
				fmt.Println("if err == io.EOF { break }")
				fmt.Println(expr + " = append(" + expr + ", " + v + ")")
				fmt.Println("chk(err)")
				fmt.Println("}")
			}
		} else {
			if lenhdr != nil {
				fmt.Println("if len("+expr+") >",
					"math.Max"+strings.Title(lenhdr.String()),
					"{ chk(ErrTooLong) }")
				genSerialize(lenhdr, lenhdr.String()+"(len("+expr+"))", pos, nil, de)
			}
			genSerialize(types.NewArray(t.Elem(), 0), expr, pos, nil, de)
		}
	case *types.Array:
		et := t.Elem()
		if et == byteT || et == uint8T {
			if de {
				fmt.Println("{",
					"_, err := io.ReadFull(r, "+expr+"[:]);",
					"chk(err)",
					"}")
			} else {
				fmt.Println("{",
					"_, err := w.Write("+expr+"[:]);",
					"chk(err)",
					"}")
			}
			break
		}
		i := newVar()
		fmt.Println("for", i, ":= range", expr, "{")
		if pos.IsValid() {
			pos = pos2node(pos)[0].(*ast.ArrayType).Elt.Pos()
		}
		genSerialize(et, expr+"["+i+"]", pos, nil, de)
		fmt.Println("}")
	default:
		error(pos, "can't serialize ", t)
	}
}

func readOverrides(path string, override map[string]string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	b := bufio.NewReader(f)
	line := 0
	col1 := ""
	for {
		ln, err := b.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(ln) > 0 {
					log.Fatal("no newline at end of ", f.Name())
				}
				return
			}
			log.Fatal(err)
		}
		line++

		if ln == "\n" {
			continue
		}

		fields := strings.SplitN(ln, "\t", 2)
		if len(fields) == 1 {
			log.Fatal(f.Name(), ":", line, ": missing tab")
		}
		if fields[0] != "" {
			col1 = fields[0]
		}

		if col1 == "" {
			fmt.Print(fields[1])
			continue
		}

		override[col1] += fields[1]
	}
}

func mkSerialize(t *types.Named) {
	if !inSerialize[t.String()] {
		serialize = append(serialize, t)
		inSerialize[t.String()] = true
	}
}

var varNo int

func newVar() string {
	varNo++
	return fmt.Sprint("local", varNo)
}

func pos2node(pos token.Pos) []ast.Node {
	return interval2node(pos, pos)
}

func interval2node(start, end token.Pos) []ast.Node {
	for _, f := range pkg.Syntax {
		if f.Pos() <= start && end <= f.End() {
			if path, _ := astutil.PathEnclosingInterval(f, start, end); path != nil {
				return path
			}
		}
	}
	return nil
}

func error(pos token.Pos, a ...interface{}) {
	if !pos.IsValid() {
		log.Fatal(a...)
	}
	log.Fatal(append([]interface{}{pkg.Fset.Position(pos), ": "}, a...)...)
}

func typeStr(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p == pkg.Types {
			return ""
		}

		return p.Name()
	})
}

var typeNames = []string{
	"ToSrvNil",
	"ToSrvInit",
	"ToSrvInit2",
	"ToSrvJoinModChan",
	"ToSrvLeaveModChan",
	"ToSrvMsgModChan",
	"ToSrvPlayerPos",
	"ToSrvGotBlks",
	"ToSrvDeletedBlks",
	"ToSrvInvAction",
	"ToSrvChatMsg",
	"ToSrvFallDmg",
	"ToSrvSelectItem",
	"ToSrvRespawn",
	"ToSrvInteract",
	"ToSrvRemovedSounds",
	"ToSrvNodeMetaFields",
	"ToSrvInvFields",
	"ToSrvReqMedia",
	"ToSrvCltReady",
	"ToSrvFirstSRP",
	"ToSrvSRPBytesA",
	"ToSrvSRPBytesM",

	"ToCltHello",
	"ToCltAcceptAuth",
	"ToCltAcceptSudoMode",
	"ToCltDenySudoMode",
	"ToCltKick",
	"ToCltBlkData",
	"ToCltAddNode",
	"ToCltRemoveNode",
	"ToCltInv",
	"ToCltTimeOfDay",
	"ToCltCSMRestrictionFlags",
	"ToCltAddPlayerVel",
	"ToCltMediaPush",
	"ToCltChatMsg",
	"ToCltAORmAdd",
	"ToCltAOMsgs",
	"ToCltHP",
	"ToCltMovePlayer",
	"ToCltLegacyKick",
	"ToCltFOV",
	"ToCltDeathScreen",
	"ToCltMedia",
	"ToCltNodeDefs",
	"ToCltAnnounceMedia",
	"ToCltItemDefs",
	"ToCltPlaySound",
	"ToCltStopSound",
	"ToCltPrivs",
	"ToCltInvFormspec",
	"ToCltDetachedInv",
	"ToCltShowFormspec",
	"ToCltMovement",
	"ToCltSpawnParticle",
	"ToCltAddParticleSpawner",
	"ToCltAddHUD",
	"ToCltRmHUD",
	"ToCltChangeHUD",
	"ToCltHUDFlags",
	"ToCltSetHotbarParam",
	"ToCltBreath",
	"ToCltSkyParams",
	"ToCltOverrideDayNightRatio",
	"ToCltLocalPlayerAnim",
	"ToCltEyeOffset",
	"ToCltDelParticleSpawner",
	"ToCltCloudParams",
	"ToCltFadeSound",
	"ToCltUpdatePlayerList",
	"ToCltModChanMsg",
	"ToCltModChanSig",
	"ToCltNodeMetasChanged",
	"ToCltSunParams",
	"ToCltMoonParams",
	"ToCltStarParams",
	"ToCltSRPBytesSaltB",
	"ToCltFormspecPrepend",

	"AOCmdProps",
	"AOCmdPos",
	"AOCmdTextureMod",
	"AOCmdSprite",
	"AOCmdHP",
	"AOCmdArmorGroups",
	"AOCmdAnim",
	"AOCmdBonePos",
	"AOCmdAttach",
	"AOCmdPhysOverride",
	"AOCmdSpawnInfant",
	"AOCmdAnimSpeed",

	"NodeMeta",
	"MinimapMode",
	"NodeDef",
	"PointedNode",
	"PointedAO",
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("mkserialize: ")

	flag.Parse()

	cfg := &packages.Config{Mode: packages.NeedSyntax |
		packages.NeedName |
		packages.NeedDeps |
		packages.NeedImports |
		packages.NeedTypes |
		packages.NeedTypesInfo}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		log.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	if len(pkgs) != 1 {
		log.Fatal("must be exactly 1 package")
	}
	pkg = pkgs[0]

	fmt.Println("package", pkg.Name)

	readOverrides("serialize.fmt", serializeFmt)
	readOverrides("deserialize.fmt", deserializeFmt)

	for _, f := range pkg.Syntax {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, "//mt:") {
					continue
				}
				st := interval2node(c.Pos(), c.End())[1].(*ast.StructType)
				consts[st] = append(consts[st], c)
			}
		}
	}

	for _, name := range typeNames {
		obj := pkg.Types.Scope().Lookup(name)
		if obj == nil {
			log.Println("undeclared identifier: ", name)
			continue
		}
		mkSerialize(obj.Type().(*types.Named))
	}

	for i := 0; i < len(serialize); i++ {
		for _, de := range []bool{false, true} {
			t := serialize[i]
			sig := "serialize(w io.Writer)"
			if de {
				sig = "deserialize(r io.Reader)"
			}
			fmt.Println("\nfunc (obj *" + t.Obj().Name() + ") " + sig + " {")
			pos := t.Obj().Pos()
			tExpr := pos2node(pos)[1].(*ast.TypeSpec).Type
			var b strings.Builder
			printer.Fprint(&b, pkg.Fset, tExpr)
			genSerialize(pkg.TypesInfo.Types[tExpr].Type, "*(*("+b.String()+"))(obj)", tExpr.Pos(), nil, de)
			fmt.Println("}")
		}
	}
}
