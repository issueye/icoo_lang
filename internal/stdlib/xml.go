package stdlib

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"icoo_lang/internal/runtime"
)

type xmlNode struct {
	Name     string
	Attrs    map[string]string
	Text     string
	Children []*xmlNode
}

func loadStdXMLModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.xml",
		Path: "std.xml",
		Exports: map[string]runtime.Value{
			"encode":     &runtime.NativeFunction{Name: "encode", Arity: 1, Fn: xmlEncode},
			"decode":     &runtime.NativeFunction{Name: "decode", Arity: 1, Fn: xmlDecode},
			"fromFile":   &runtime.NativeFunction{Name: "fromFile", Arity: 1, Fn: xmlFromFile},
			"saveToFile": &runtime.NativeFunction{Name: "saveToFile", Arity: 2, Fn: xmlSaveToFile},
		},
		Done: true,
	}
}

func xmlEncode(args []runtime.Value) (runtime.Value, error) {
	node, err := runtimeToXMLNode(args[0])
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := encodeXMLNode(enc, node); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: buf.String()}, nil
}

func xmlDecode(args []runtime.Value) (runtime.Value, error) {
	text, ok := args[0].(runtime.StringValue)
	if !ok {
		return nil, fmt.Errorf("decode expects string")
	}
	dec := xml.NewDecoder(strings.NewReader(text.Value))
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return runtime.NullValue{}, nil
			}
			return nil, err
		}
		if start, ok := tok.(xml.StartElement); ok {
			node, err := decodeXMLNode(dec, start)
			if err != nil {
				return nil, err
			}
			return xmlNodeToRuntime(node), nil
		}
	}
}

func xmlFromFile(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("fromFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return xmlDecode([]runtime.Value{runtime.StringValue{Value: string(data)}})
}

func xmlSaveToFile(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("saveToFile", args[0])
	if err != nil {
		return nil, err
	}
	encoded, err := xmlEncode([]runtime.Value{args[1]})
	if err != nil {
		return nil, err
	}
	text := encoded.(runtime.StringValue)
	if err := os.WriteFile(path, []byte(text.Value), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func runtimeToXMLNode(v runtime.Value) (*xmlNode, error) {
	obj, ok := v.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("xml.encode expects object node")
	}

	nameValue, ok := obj.Fields["name"].(runtime.StringValue)
	if !ok || nameValue.Value == "" {
		return nil, fmt.Errorf("xml node requires non-empty name")
	}

	node := &xmlNode{
		Name:  nameValue.Value,
		Attrs: make(map[string]string),
	}
	if textValue, ok := obj.Fields["text"].(runtime.StringValue); ok {
		node.Text = textValue.Value
	}
	if attrsValue, ok := obj.Fields["attrs"]; ok {
		attrsObj, ok := attrsValue.(*runtime.ObjectValue)
		if !ok {
			return nil, fmt.Errorf("xml node attrs must be object")
		}
		for key, value := range attrsObj.Fields {
			node.Attrs[key] = value.String()
		}
	}
	if childrenValue, ok := obj.Fields["children"]; ok {
		childrenArray, ok := childrenValue.(*runtime.ArrayValue)
		if !ok {
			return nil, fmt.Errorf("xml node children must be array")
		}
		node.Children = make([]*xmlNode, 0, len(childrenArray.Elements))
		for _, childValue := range childrenArray.Elements {
			childNode, err := runtimeToXMLNode(childValue)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, childNode)
		}
	}
	return node, nil
}

func encodeXMLNode(enc *xml.Encoder, node *xmlNode) error {
	start := xml.StartElement{Name: xml.Name{Local: node.Name}}
	for key, value := range node.Attrs {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: key}, Value: value})
	}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}
	if node.Text != "" {
		if err := enc.EncodeToken(xml.CharData([]byte(node.Text))); err != nil {
			return err
		}
	}
	for _, child := range node.Children {
		if err := encodeXMLNode(enc, child); err != nil {
			return err
		}
	}
	return enc.EncodeToken(start.End())
}

func decodeXMLNode(dec *xml.Decoder, start xml.StartElement) (*xmlNode, error) {
	node := &xmlNode{
		Name:  start.Name.Local,
		Attrs: make(map[string]string, len(start.Attr)),
	}
	for _, attr := range start.Attr {
		node.Attrs[attr.Name.Local] = attr.Value
	}

	var textParts []string
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch value := tok.(type) {
		case xml.StartElement:
			child, err := decodeXMLNode(dec, value)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		case xml.CharData:
			text := strings.TrimSpace(string(value))
			if text != "" {
				textParts = append(textParts, text)
			}
		case xml.EndElement:
			if value.Name.Local == start.Name.Local {
				node.Text = strings.Join(textParts, " ")
				return node, nil
			}
		}
	}
}

func xmlNodeToRuntime(node *xmlNode) runtime.Value {
	fields := map[string]runtime.Value{
		"name": runtime.StringValue{Value: node.Name},
	}
	if len(node.Attrs) > 0 {
		attrs := make(map[string]runtime.Value, len(node.Attrs))
		for key, value := range node.Attrs {
			attrs[key] = runtime.StringValue{Value: value}
		}
		fields["attrs"] = &runtime.ObjectValue{Fields: attrs}
	}
	if node.Text != "" {
		fields["text"] = runtime.StringValue{Value: node.Text}
	}
	if len(node.Children) > 0 {
		children := make([]runtime.Value, 0, len(node.Children))
		for _, child := range node.Children {
			children = append(children, xmlNodeToRuntime(child))
		}
		fields["children"] = &runtime.ArrayValue{Elements: children}
	}
	return &runtime.ObjectValue{Fields: fields}
}
