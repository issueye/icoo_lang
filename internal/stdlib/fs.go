package stdlib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"icoo_lang/internal/runtime"
)

func loadStdFSModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.fs",
		Path: "std.fs",
		Exports: map[string]runtime.Value{
			"base":      &runtime.NativeFunction{Name: "base", Arity: 1, Fn: fsBase},
			"copyFile":  &runtime.NativeFunction{Name: "copyFile", Arity: 2, Fn: fsCopyFile},
			"dir":       &runtime.NativeFunction{Name: "dir", Arity: 1, Fn: fsDir},
			"exists":    &runtime.NativeFunction{Name: "exists", Arity: 1, Fn: fsExists},
			"join":      &runtime.NativeFunction{Name: "join", Arity: 2, Fn: fsJoin},
			"mkdir":     &runtime.NativeFunction{Name: "mkdir", Arity: 1, Fn: fsMkdir},
			"readDir":   &runtime.NativeFunction{Name: "readDir", Arity: 1, Fn: fsReadDir},
			"readFile":  &runtime.NativeFunction{Name: "readFile", Arity: 1, Fn: fsReadFile},
			"remove":    &runtime.NativeFunction{Name: "remove", Arity: 1, Fn: fsRemove},
			"rename":    &runtime.NativeFunction{Name: "rename", Arity: 2, Fn: fsRename},
			"stat":      &runtime.NativeFunction{Name: "stat", Arity: 1, Fn: fsStat},
			"writeFile": &runtime.NativeFunction{Name: "writeFile", Arity: 2, Fn: fsWriteFile},
		},
		Done: true,
	}
}

func fsReadFile(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("readFile", args[0])
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: string(data)}, nil
}

func fsWriteFile(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("writeFile", args[0])
	if err != nil {
		return nil, err
	}
	content, err := requireStringArg("writeFile", args[1])
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func fsExists(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("exists", args[0])
	if err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	if statErr == nil {
		return runtime.BoolValue{Value: true}, nil
	}
	if os.IsNotExist(statErr) {
		return runtime.BoolValue{Value: false}, nil
	}
	return nil, statErr
}

func fsMkdir(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("mkdir", args[0])
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func fsRemove(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("remove", args[0])
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func fsReadDir(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("readDir", args[0])
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	items := make([]runtime.Value, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		items = append(items, &runtime.ObjectValue{Fields: map[string]runtime.Value{
			"name":  runtime.StringValue{Value: name},
			"path":  runtime.StringValue{Value: filepath.Join(path, name)},
			"isDir": runtime.BoolValue{Value: entry.IsDir()},
		}})
	}
	return &runtime.ArrayValue{Elements: items}, nil
}

func fsStat(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("stat", args[0])
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"name":    runtime.StringValue{Value: info.Name()},
		"size":    runtime.IntValue{Value: info.Size()},
		"isDir":   runtime.BoolValue{Value: info.IsDir()},
		"isFile":  runtime.BoolValue{Value: info.Mode().IsRegular()},
		"mode":    runtime.StringValue{Value: info.Mode().String()},
		"modTime": runtime.IntValue{Value: info.ModTime().Unix()},
	}}, nil
}

func fsRename(args []runtime.Value) (runtime.Value, error) {
	oldPath, err := requireStringArg("rename", args[0])
	if err != nil {
		return nil, err
	}
	newPath, err := requireStringArg("rename", args[1])
	if err != nil {
		return nil, err
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func fsCopyFile(args []runtime.Value) (runtime.Value, error) {
	srcPath, err := requireStringArg("copyFile", args[0])
	if err != nil {
		return nil, err
	}
	dstPath, err := requireStringArg("copyFile", args[1])
	if err != nil {
		return nil, err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer src.Close()
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}
	return runtime.NullValue{}, nil
}

func fsJoin(args []runtime.Value) (runtime.Value, error) {
	left, err := requireStringArg("join", args[0])
	if err != nil {
		return nil, err
	}
	right, err := requireStringArg("join", args[1])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: filepath.Join(left, right)}, nil
}

func fsBase(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("base", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: filepath.Base(path)}, nil
}

func fsDir(args []runtime.Value) (runtime.Value, error) {
	path, err := requireStringArg("dir", args[0])
	if err != nil {
		return nil, err
	}
	return runtime.StringValue{Value: filepath.Dir(path)}, nil
}

func requireStringArg(name string, v runtime.Value) (string, error) {
	text, ok := v.(runtime.StringValue)
	if !ok {
		return "", fmt.Errorf("%s expects string argument", name)
	}
	return text.Value, nil
}
