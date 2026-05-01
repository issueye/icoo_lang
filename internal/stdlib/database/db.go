package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type dbHandle struct {
	db *sql.DB
}

type dbHandleValue struct {
	handle *dbHandle
}

func (v *dbHandleValue) Kind() runtime.ValueKind { return runtime.ObjectKind }

func (v *dbHandleValue) String() string {
	if v == nil || v.handle == nil || v.handle.db == nil {
		return "<db closed>"
	}
	return "<db>"
}

func LoadStdDBModule() *runtime.Module {
	return &runtime.Module{
		Name: "std.db",
		Path: "std.db",
		Exports: map[string]runtime.Value{
			"mysql":    &runtime.NativeFunction{Name: "mysql", Arity: 1, Fn: dbOpenMySQL},
			"open":     &runtime.NativeFunction{Name: "open", Arity: 2, Fn: dbOpen},
			"pgsql":    &runtime.NativeFunction{Name: "pgsql", Arity: 1, Fn: dbOpenPostgres},
			"postgres": &runtime.NativeFunction{Name: "postgres", Arity: 1, Fn: dbOpenPostgres},
			"sqlite":   &runtime.NativeFunction{Name: "sqlite", Arity: 1, Fn: dbOpenSQLite},
		},
		Done: true,
	}
}

func dbOpen(args []runtime.Value) (runtime.Value, error) {
	driver, err := utils.RequireStringArg("open", args[0])
	if err != nil {
		return nil, err
	}
	dsn, err := utils.RequireStringArg("open", args[1])
	if err != nil {
		return nil, err
	}
	return openDB(driver, dsn)
}

func dbOpenSQLite(args []runtime.Value) (runtime.Value, error) {
	dsn, err := utils.RequireStringArg("sqlite", args[0])
	if err != nil {
		return nil, err
	}
	return openDB("sqlite", dsn)
}

func dbOpenPostgres(args []runtime.Value) (runtime.Value, error) {
	dsn, err := utils.RequireStringArg("pgsql", args[0])
	if err != nil {
		return nil, err
	}
	return openDB("pgx", dsn)
}

func dbOpenMySQL(args []runtime.Value) (runtime.Value, error) {
	dsn, err := utils.RequireStringArg("mysql", args[0])
	if err != nil {
		return nil, err
	}
	return openDB("mysql", dsn)
}

func openDB(driver string, dsn string) (runtime.Value, error) {
	normalized, err := normalizeDriver(driver)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open(normalized, dsn)
	if err != nil {
		return nil, err
	}
	if normalized == "sqlite" {
		db.SetMaxOpenConns(1)
	}
	return newDBObject(&dbHandle{db: db}), nil
}

func newDBObject(handle *dbHandle) *runtime.ObjectValue {
	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"close": &runtime.NativeFunction{Name: "db.close", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if handle.db == nil {
				return runtime.NullValue{}, nil
			}
			if err := handle.db.Close(); err != nil {
				return nil, err
			}
			handle.db = nil
			return runtime.NullValue{}, nil
		}},
		"exec": &runtime.NativeFunction{Name: "db.exec", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			db, err := handle.requireDB("exec")
			if err != nil {
				return nil, err
			}
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("exec expects 1 or 2 arguments")
			}
			query, err := utils.RequireStringArg("exec", args[0])
			if err != nil {
				return nil, err
			}
			queryArgs, err := optionalSQLArgs("exec", args)
			if err != nil {
				return nil, err
			}
			result, err := db.Exec(query, queryArgs...)
			if err != nil {
				return nil, err
			}
			return sqlResultToRuntime(result), nil
		}},
		"ping": &runtime.NativeFunction{Name: "db.ping", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			db, err := handle.requireDB("ping")
			if err != nil {
				return nil, err
			}
			if err := db.Ping(); err != nil {
				return nil, err
			}
			return runtime.BoolValue{Value: true}, nil
		}},
		"query": &runtime.NativeFunction{Name: "db.query", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			db, err := handle.requireDB("query")
			if err != nil {
				return nil, err
			}
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("query expects 1 or 2 arguments")
			}
			query, err := utils.RequireStringArg("query", args[0])
			if err != nil {
				return nil, err
			}
			queryArgs, err := optionalSQLArgs("query", args)
			if err != nil {
				return nil, err
			}
			rows, err := db.Query(query, queryArgs...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			return scanRows(rows)
		}},
		"queryOne": &runtime.NativeFunction{Name: "db.queryOne", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			db, err := handle.requireDB("queryOne")
			if err != nil {
				return nil, err
			}
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("queryOne expects 1 or 2 arguments")
			}
			query, err := utils.RequireStringArg("queryOne", args[0])
			if err != nil {
				return nil, err
			}
			queryArgs, err := optionalSQLArgs("queryOne", args)
			if err != nil {
				return nil, err
			}
			rows, err := db.Query(query, queryArgs...)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			result, err := scanRows(rows)
			if err != nil {
				return nil, err
			}
			array := result.(*runtime.ArrayValue)
			if len(array.Elements) == 0 {
				return runtime.NullValue{}, nil
			}
			return array.Elements[0], nil
		}},
		"raw": &dbHandleValue{handle: handle},
	}}
}

func (h *dbHandle) requireDB(name string) (*sql.DB, error) {
	if h == nil || h.db == nil {
		return nil, fmt.Errorf("%s called on closed database", name)
	}
	return h.db, nil
}

func normalizeDriver(driver string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlite", "sqlite3":
		return "sqlite", nil
	case "pgsql", "postgres", "postgresql", "pgx":
		return "pgx", nil
	case "mysql":
		return "mysql", nil
	default:
		return "", fmt.Errorf("unsupported database driver %q", driver)
	}
}

func optionalSQLArgs(name string, args []runtime.Value) ([]any, error) {
	if len(args) == 1 {
		return nil, nil
	}
	array, ok := args[1].(*runtime.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("%s expects argument list as array", name)
	}
	values := make([]any, 0, len(array.Elements))
	for _, elem := range array.Elements {
		plain, err := utils.RuntimeToPlainValue(elem)
		if err != nil {
			return nil, err
		}
		values = append(values, plain)
	}
	return values, nil
}

func sqlResultToRuntime(result sql.Result) runtime.Value {
	lastInsertIDValue := runtime.Value(runtime.NullValue{})
	if lastInsertID, err := result.LastInsertId(); err == nil {
		lastInsertIDValue = runtime.IntValue{Value: lastInsertID}
	}

	rowsAffectedValue := runtime.Value(runtime.NullValue{})
	if rowsAffected, err := result.RowsAffected(); err == nil {
		rowsAffectedValue = runtime.IntValue{Value: rowsAffected}
	}

	return &runtime.ObjectValue{Fields: map[string]runtime.Value{
		"lastInsertId": lastInsertIDValue,
		"rowsAffected": rowsAffectedValue,
	}}
}

func scanRows(rows *sql.Rows) (runtime.Value, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := &runtime.ArrayValue{Elements: []runtime.Value{}}
	for rows.Next() {
		values := make([]any, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		fields := make(map[string]runtime.Value, len(columns))
		for i, column := range columns {
			fields[column] = sqlValueToRuntime(values[i])
		}
		result.Elements = append(result.Elements, &runtime.ObjectValue{Fields: fields})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func sqlValueToRuntime(value any) runtime.Value {
	if value == nil {
		return runtime.NullValue{}
	}
	bytes, ok := value.([]byte)
	if ok {
		return runtime.StringValue{Value: string(bytes)}
	}
	return utils.PlainToRuntimeValue(value)
}

func requireDB(name string, value runtime.Value) (*sql.DB, error) {
	object, ok := value.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects database object", name)
	}
	raw, ok := object.Fields["raw"].(*dbHandleValue)
	if !ok || raw.handle == nil {
		return nil, errors.New("invalid database object")
	}
	return raw.handle.requireDB(name)
}
