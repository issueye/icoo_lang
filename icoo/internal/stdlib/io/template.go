package io

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"
)

// LoadStdIOTemplateModule 加载 std.io.template 模块
func LoadStdIOTemplateModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.io.template",
		Path: "std.io.template",
		Exports: map[string]runtime.Value{
			"compile": &runtime.NativeFunction{Name: "compile", Arity: 1, Fn: templateCompile},
			"render":  &runtime.NativeFunction{Name: "render", Arity: 2, Fn: templateRender},
		},
		Done: true,
	}
}

// templateCompile 编译模板
func templateCompile(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("compile", args[0])
	if err != nil {
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"render": &runtime.NativeFunction{Name: "template.render", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return renderTemplate(text, args[0])
		}},
		"source": runtime.StringValue{Value: text},
	}}, nil
}

// templateRender 渲染模板
func templateRender(args []runtime.Value) (runtime.Value, error) {
	text, err := utils.RequireStringArg("render", args[0])
	if err != nil {
		return nil, err
	}
	return renderTemplate(text, args[1])
}

// renderTemplate 渲染模板字符串
func renderTemplate(text string, data runtime.Value) (runtime.Value, error) {
	plain, err := utils.RuntimeToPlainValue(data)
	if err != nil {
		return nil, err
	}
	rendered, err := renderTemplateSegment(text, plain, plain)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: rendered}, nil
}

// renderTemplateSegment 渲染模板片段
func renderTemplateSegment(text string, root any, current any) (string, error) {
	var out strings.Builder
	for {
		start := strings.Index(text, "{{")
		if start < 0 {
			out.WriteString(text)
			return out.String(), nil
		}
		out.WriteString(text[:start])
		text = text[start:]
		end := strings.Index(text, "}}")
		if end < 0 {
			return "", fmt.Errorf("template tag is not closed")
		}
		tag := strings.TrimSpace(text[2:end])
		rest := text[end+2:]
		switch {
		case strings.HasPrefix(tag, "#if "):
			expr := strings.TrimSpace(strings.TrimPrefix(tag, "#if "))
			body, tail, err := templateSectionBody(rest, "if")
			if err != nil {
				return "", err
			}
			value, _ := templateResolve(expr, root, current)
			if templateTruthy(value) {
				rendered, err := renderTemplateSegment(body, root, current)
				if err != nil {
					return "", err
				}
				out.WriteString(rendered)
			}
			text = tail
		case strings.HasPrefix(tag, "#each "):
			expr := strings.TrimSpace(strings.TrimPrefix(tag, "#each "))
			body, tail, err := templateSectionBody(rest, "each")
			if err != nil {
				return "", err
			}
			value, _ := templateResolve(expr, root, current)
			items := templateSlice(value)
			for _, item := range items {
				rendered, err := renderTemplateSegment(body, root, item)
				if err != nil {
					return "", err
				}
				out.WriteString(rendered)
			}
			text = tail
		case strings.HasPrefix(tag, "/"):
			return "", fmt.Errorf("unexpected template closing tag %q", tag)
		default:
			value, _ := templateResolve(tag, root, current)
			out.WriteString(templateString(value))
			text = rest
		}
	}
}

// templateSectionBody 提取模板区块体
func templateSectionBody(text string, section string) (string, string, error) {
	depth := 1
	index := 0
	for index < len(text) {
		start := strings.Index(text[index:], "{{")
		if start < 0 {
			return "", "", fmt.Errorf("template section %s is not closed", section)
		}
		start += index
		end := strings.Index(text[start:], "}}")
		if end < 0 {
			return "", "", fmt.Errorf("template tag is not closed")
		}
		end += start
		tag := strings.TrimSpace(text[start+2 : end])
		if strings.HasPrefix(tag, "#"+section+" ") {
			depth++
		} else if tag == "/"+section {
			depth--
			if depth == 0 {
				return text[:start], text[end+2:], nil
			}
		}
		index = end + 2
	}
	return "", "", fmt.Errorf("template section %s is not closed", section)
}

// templateResolve 解析模板变量
func templateResolve(path string, root any, current any) (any, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", true
	}
	if path == "." || path == "this" {
		return current, true
	}
	if strings.HasPrefix(path, "this.") {
		return templateLookupPath(current, strings.TrimPrefix(path, "this."))
	}
	if value, ok := templateLookupPath(current, path); ok {
		return value, true
	}
	return templateLookupPath(root, path)
}

// templateLookupPath 按路径查找值
func templateLookupPath(value any, path string) (any, bool) {
	if path == "" {
		return value, true
	}
	current := value
	parts := strings.Split(path, ".")
	for _, part := range parts {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

// templateTruthy 判断值是否为真
func templateTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case int, int8, int16, int32, int64:
		return fmt.Sprint(typed) != "0"
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(typed) != "0"
	case float32:
		return typed != 0
	case float64:
		return typed != 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

// templateSlice 将值转换为切片
func templateSlice(value any) []any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []any:
		return typed
	default:
		return []any{typed}
	}
}

// templateString 将值转换为字符串
func templateString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(typed)
	case map[string]any, []any:
		data, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(data)
	default:
		return fmt.Sprint(typed)
	}
}
