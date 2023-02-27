package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	pgs "github.com/lyft/protoc-gen-star"

	"oss.terrastruct.com/d2/d2graph"
	"oss.terrastruct.com/d2/d2layouts/d2dagrelayout"
	"oss.terrastruct.com/d2/d2lib"
	"oss.terrastruct.com/d2/d2renderers/d2svg"
	"oss.terrastruct.com/d2/d2themes/d2themescatalog"
	"oss.terrastruct.com/d2/lib/textmeasure"
)

var typeMap = map[string]string{
	"TYPE_FLOAT":    "float32",
	"TYPE_DOUBLE":   "float64",
	"TYPE_INT32":    "int32",
	"TYPE_INT64":    "int64",
	"TYPE_UINT32":   "uint32",
	"TYPE_UINT64":   "uint64",
	"TYPE_SINT32":   "int",
	"TYPE_SINT64":   "int64",
	"TYPE_FIXED32":  "uint32",
	"TYPE_FIXED64":  "uint64",
	"TYPE_SFIXED32": "int",
	"TYPE_SFIXED64": "int64",
	"TYPE_BOOL":     "bool",
	"TYPE_BYTES":    "[]byte",
	"TYPE_STRING":   "string",
}

type D2 struct {
	*pgs.ModuleBase
	buf io.ReadWriter
}

func newD2() *D2 {
	return &D2{
		ModuleBase: &pgs.ModuleBase{},
		buf:        bytes.NewBuffer(nil),
	}
}

func (e *D2) Name() string {
	return "D2"
}

func (e *D2) parseField(field pgs.FieldType) string {
	var name []string
	if field.IsRepeated() {
		name = append(name, "repeated")
	}

	if enum := field.Enum(); enum != nil {
		name = append(name, "enum")
		name = append(name, enum.Name().String())
	}

	if field.IsEmbed() && !field.IsMap() {
		name = append(name, "message")
		name = append(name, field.Embed().Name().String())
	}

	if field.IsMap() {
		key := field.Key().ProtoType().String()
		if newKey := getGoType(key); newKey != "" {
			key = newKey
		}
		value := field.Element().ProtoType().String()
		if newValue := getGoType(value); newValue != "" {
			value = newValue
		}
		name = append(name, fmt.Sprintf("map<%s, %s>", key, value))
	}

	if typeName := getGoType(field.ProtoType().String()); typeName != "" {
		name = append(name, typeName)
	}

	return strings.Join(name, " ")
}

func (e *D2) addContainer(name string, children func()) {
	e.buf.Write([]byte(fmt.Sprintf("%s: {\n", name)))
	children()
	e.buf.Write([]byte("}\n"))
}

func (e *D2) addEnum(enum pgs.Enum) {
	e.addContainer("Enums", func() {
		e.buf.Write([]byte("  direction: down\n"))
		e.buf.Write([]byte(fmt.Sprintf("  %s: {\n", enum.Name().String())))
		e.buf.Write([]byte("    shape: class\n"))
		for i, value := range enum.Values() {
			e.buf.Write([]byte(fmt.Sprintf("    %s = %d\n", value.Name().String(), i)))
		}
		e.buf.Write([]byte("  }\n"))
	})
}

func (e *D2) addMessage(message pgs.Message) {
	e.addContainer("Messages", func() {
		e.buf.Write([]byte("  direction: down\n"))
		e.buf.Write([]byte(fmt.Sprintf("  %s: {\n", message.Name().String())))
		e.buf.Write([]byte("    shape: class\n"))
		for i, field := range message.Fields() {
			ftype := e.parseField(field.Type())
			e.buf.Write([]byte(fmt.Sprintf("    %s %s = %d\n", ftype, field.Name(), i+1)))
		}
		e.buf.Write([]byte("  }\n"))
	})
}

func (e *D2) addService(service pgs.Service) {
	e.addContainer("Services", func() {
		e.buf.Write([]byte("  direction: down\n"))
		e.buf.Write([]byte(fmt.Sprintf("  %s: {\n", service.Name().String())))
		e.buf.Write([]byte("    shape: class\n"))
		for _, method := range service.Methods() {
			e.buf.Write([]byte(fmt.Sprintf("    %s(%s): %s\n", method.Name(), method.Input().Name(), method.Output().Name())))
		}
		e.buf.Write([]byte("  }\n"))
	})
}

func (e *D2) Execute(targets map[string]pgs.File, packages map[string]pgs.Package) []pgs.Artifact {
	e.buf.Write([]byte("direction: down\n"))

	for _, file := range targets {
		for _, svc := range file.Services() {
			e.addService(svc)
		}
		for _, enum := range file.Enums() {
			e.addEnum(enum)
		}
		for _, message := range file.Messages() {
			e.addMessage(message)
		}

		ruler, _ := textmeasure.NewRuler()
		defaultLayout := func(ctx context.Context, g *d2graph.Graph) error {
			return d2dagrelayout.Layout(ctx, g, nil)
		}
		body, err := ioutil.ReadAll(e.buf)
		if err != nil {
			panic(err)
		}
		println(string(body))
		diagram, _, err := d2lib.Compile(context.Background(), string(body), &d2lib.CompileOptions{
			Layout:  defaultLayout,
			Ruler:   ruler,
			ThemeID: d2themescatalog.GrapeSoda.ID,
		})
		if err != nil {
			panic(err)
		}
		out, err := d2svg.Render(diagram, &d2svg.RenderOpts{
			Pad: d2svg.DEFAULT_PADDING,
		})
		if err != nil {
			panic(err)
		}
		e.AddGeneratorFile(
			file.InputPath().SetExt(".svg").String(),
			string(out),
		)
	}

	return e.Artifacts()
}

func getGoType(proto string) string {
	if name, found := typeMap[proto]; found {
		return name
	}
	return ""
}
