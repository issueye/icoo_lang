package database

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"icoo_lang/internal/runtime"
	"icoo_lang/internal/stdlib/utils"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type dbHandle struct {
	db     *sql.DB
	driver string
}

type dbHandleValue struct {
	handle *dbHandle
}

type ormQueryState struct {
	handle       *dbHandle
	table        string
	columnDefs   map[string]string
	selectExpr   string
	whereClauses []string
	whereArgs    []any
	orderBy      string
	limit        *int64
	offset       *int64
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
	return newDBObject(&dbHandle{db: db, driver: normalized}), nil
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
		"table": &runtime.NativeFunction{Name: "db.table", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if _, err := handle.requireDB("table"); err != nil {
				return nil, err
			}
			tableName, err := utils.RequireStringArg("table", args[0])
			if err != nil {
				return nil, err
			}
			tableName = strings.TrimSpace(tableName)
			if tableName == "" {
				return nil, fmt.Errorf("table expects non-empty table name")
			}
			return newORMQueryObject(&ormQueryState{
				handle:     handle,
				table:      tableName,
				selectExpr: "*",
			}), nil
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
	handle, err := requireDBHandle(name, value)
	if err != nil {
		return nil, err
	}
	return handle.requireDB(name)
}

func requireDBHandle(name string, value runtime.Value) (*dbHandle, error) {
	object, ok := value.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects database object", name)
	}
	raw, ok := object.Fields["raw"].(*dbHandleValue)
	if !ok || raw.handle == nil {
		return nil, errors.New("invalid database object")
	}
	return raw.handle, nil
}

func newORMQueryObject(state *ormQueryState) *runtime.ObjectValue {
	fields := map[string]runtime.Value{
		"name": runtime.StringValue{Value: state.table},
		"create": &runtime.NativeFunction{Name: "db.table.create", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormCreateTable(state)
		}},
		"select": &runtime.NativeFunction{Name: "db.table.select", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			selectExpr, err := ormSelectExpr(args[0], state.handle.driver)
			if err != nil {
				return nil, err
			}
			next := state.clone()
			next.selectExpr = selectExpr
			return newORMQueryObject(next), nil
		}},
		"where": &runtime.NativeFunction{Name: "db.table.where", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			next, err := ormApplyWhere(state, args[0])
			if err != nil {
				return nil, err
			}
			return newORMQueryObject(next), nil
		}},
		"whereRaw": &runtime.NativeFunction{Name: "db.table.whereRaw", Arity: -1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("whereRaw expects 1 or 2 arguments")
			}
			clause, err := utils.RequireStringArg("whereRaw", args[0])
			if err != nil {
				return nil, err
			}
			clause = strings.TrimSpace(clause)
			if clause == "" {
				return nil, fmt.Errorf("whereRaw expects non-empty sql")
			}
			queryArgs, err := optionalSQLArgs("whereRaw", args)
			if err != nil {
				return nil, err
			}
			next := state.clone()
			next.whereClauses = append(next.whereClauses, clause)
			next.whereArgs = append(next.whereArgs, queryArgs...)
			return newORMQueryObject(next), nil
		}},
		"orderBy": &runtime.NativeFunction{Name: "db.table.orderBy", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			orderBy, err := utils.RequireStringArg("orderBy", args[0])
			if err != nil {
				return nil, err
			}
			orderBy = strings.TrimSpace(orderBy)
			if orderBy == "" {
				return nil, fmt.Errorf("orderBy expects non-empty sql")
			}
			next := state.clone()
			next.orderBy = orderBy
			return newORMQueryObject(next), nil
		}},
		"limit": &runtime.NativeFunction{Name: "db.table.limit", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			value, err := ormRequireIntArg("limit", args[0])
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return nil, fmt.Errorf("limit expects non-negative integer")
			}
			next := state.clone()
			next.limit = &value
			return newORMQueryObject(next), nil
		}},
		"offset": &runtime.NativeFunction{Name: "db.table.offset", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			value, err := ormRequireIntArg("offset", args[0])
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return nil, fmt.Errorf("offset expects non-negative integer")
			}
			next := state.clone()
			next.offset = &value
			return newORMQueryObject(next), nil
		}},
		"all": &runtime.NativeFunction{Name: "db.table.all", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormQueryAll(state)
		}},
		"get": &runtime.NativeFunction{Name: "db.table.get", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormQueryFirst(state)
		}},
		"first": &runtime.NativeFunction{Name: "db.table.first", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormQueryFirst(state)
		}},
		"count": &runtime.NativeFunction{Name: "db.table.count", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormQueryCount(state)
		}},
		"insert": &runtime.NativeFunction{Name: "db.table.insert", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormInsert(state, args[0])
		}},
		"update": &runtime.NativeFunction{Name: "db.table.update", Arity: 1, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormUpdate(state, args[0])
		}},
		"delete": &runtime.NativeFunction{Name: "db.table.delete", Arity: 0, Fn: func(args []runtime.Value) (runtime.Value, error) {
			return ormDelete(state)
		}},
	}
	return &runtime.ObjectValue{Fields: fields}
}

func (s *ormQueryState) clone() *ormQueryState {
	if s == nil {
		return nil
	}
	next := &ormQueryState{
		handle:     s.handle,
		table:      s.table,
		columnDefs: ormCloneColumnDefs(s.columnDefs),
		selectExpr: s.selectExpr,
		orderBy:    s.orderBy,
	}
	if len(s.whereClauses) > 0 {
		next.whereClauses = append([]string{}, s.whereClauses...)
	}
	if len(s.whereArgs) > 0 {
		next.whereArgs = append([]any{}, s.whereArgs...)
	}
	if s.limit != nil {
		value := *s.limit
		next.limit = &value
	}
	if s.offset != nil {
		value := *s.offset
		next.offset = &value
	}
	return next
}

func ormCloneColumnDefs(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func ormSelectExpr(arg runtime.Value, driver string) (string, error) {
	switch value := arg.(type) {
	case runtime.StringValue:
		text := strings.TrimSpace(value.Value)
		if text == "" {
			return "", fmt.Errorf("select expects non-empty column expression")
		}
		return ormQuoteIdentifierMaybe(driver, text), nil
	case *runtime.ArrayValue:
		if len(value.Elements) == 0 {
			return "", fmt.Errorf("select expects at least one column")
		}
		parts := make([]string, 0, len(value.Elements))
		for _, elem := range value.Elements {
			column, ok := elem.(runtime.StringValue)
			if !ok {
				return "", fmt.Errorf("select expects string columns")
			}
			text := strings.TrimSpace(column.Value)
			if text == "" {
				return "", fmt.Errorf("select expects non-empty column")
			}
			parts = append(parts, ormQuoteIdentifierMaybe(driver, text))
		}
		return strings.Join(parts, ", "), nil
	default:
		return "", fmt.Errorf("select expects string or array of strings")
	}
}

func ormRequireIntArg(name string, value runtime.Value) (int64, error) {
	intValue, ok := value.(runtime.IntValue)
	if !ok {
		return 0, fmt.Errorf("%s expects integer argument", name)
	}
	return intValue.Value, nil
}

func ormQueryAll(state *ormQueryState) (runtime.Value, error) {
	db, err := state.handle.requireDB("all")
	if err != nil {
		return nil, err
	}
	query, queryArgs := state.buildSelectSQL()
	rows, err := db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func ormCreateTable(state *ormQueryState) (runtime.Value, error) {
	db, err := state.handle.requireDB("create")
	if err != nil {
		return nil, err
	}
	if len(state.columnDefs) == 0 {
		return nil, fmt.Errorf("create requires model columns")
	}

	keys := make([]string, 0, len(state.columnDefs))
	for key := range state.columnDefs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		definition := strings.TrimSpace(state.columnDefs[key])
		if definition == "" {
			return nil, fmt.Errorf("create requires non-empty column definition for %s", key)
		}
		parts = append(parts, fmt.Sprintf("%s %s", ormQuoteIdentifierMaybe(state.handle.driver, key), definition))
	}

	query := fmt.Sprintf(
		"create table if not exists %s (%s)",
		ormQuoteIdentifierMaybe(state.handle.driver, state.table),
		strings.Join(parts, ", "),
	)
	result, err := db.Exec(query)
	if err != nil {
		return nil, err
	}
	return sqlResultToRuntime(result), nil
}

func ormQueryFirst(state *ormQueryState) (runtime.Value, error) {
	next := state.clone()
	if next.limit == nil || *next.limit > 1 {
		one := int64(1)
		next.limit = &one
	}
	result, err := ormQueryAll(next)
	if err != nil {
		return nil, err
	}
	array := result.(*runtime.ArrayValue)
	if len(array.Elements) == 0 {
		return runtime.NullValue{}, nil
	}
	return array.Elements[0], nil
}

func ormQueryCount(state *ormQueryState) (runtime.Value, error) {
	db, err := state.handle.requireDB("count")
	if err != nil {
		return nil, err
	}
	query, queryArgs := state.buildCountSQL()
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
		return runtime.IntValue{Value: 0}, nil
	}
	row, ok := array.Elements[0].(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("count query returned invalid row")
	}
	if count, ok := row.Fields["count"]; ok {
		return count, nil
	}
	return runtime.IntValue{Value: 0}, nil
}

func ormInsert(state *ormQueryState, value runtime.Value) (runtime.Value, error) {
	db, err := state.handle.requireDB("insert")
	if err != nil {
		return nil, err
	}
	obj, err := ormRequireObjectArg("insert", value)
	if err != nil {
		return nil, err
	}
	keys := ormSortedKeys(obj.Fields)
	if len(keys) == 0 {
		return nil, fmt.Errorf("insert expects object with at least one field")
	}

	columns := make([]string, 0, len(keys))
	placeholders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for _, key := range keys {
		plain, err := utils.RuntimeToPlainValue(obj.Fields[key])
		if err != nil {
			return nil, err
		}
		columns = append(columns, ormQuoteIdentifierMaybe(state.handle.driver, key))
		placeholders = append(placeholders, "?")
		args = append(args, plain)
	}

	query := fmt.Sprintf(
		"insert into %s (%s) values (%s)",
		ormQuoteIdentifierMaybe(state.handle.driver, state.table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	query = ormRebindSQL(state.handle.driver, query)
	result, err := db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return sqlResultToRuntime(result), nil
}

func ormUpdate(state *ormQueryState, value runtime.Value) (runtime.Value, error) {
	db, err := state.handle.requireDB("update")
	if err != nil {
		return nil, err
	}
	if err := state.requireSafeMutation("update"); err != nil {
		return nil, err
	}
	obj, err := ormRequireObjectArg("update", value)
	if err != nil {
		return nil, err
	}
	keys := ormSortedKeys(obj.Fields)
	if len(keys) == 0 {
		return nil, fmt.Errorf("update expects object with at least one field")
	}

	assignments := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+len(state.whereArgs))
	for _, key := range keys {
		plain, err := utils.RuntimeToPlainValue(obj.Fields[key])
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, fmt.Sprintf("%s = ?", ormQuoteIdentifierMaybe(state.handle.driver, key)))
		args = append(args, plain)
	}

	query := fmt.Sprintf(
		"update %s set %s where %s",
		ormQuoteIdentifierMaybe(state.handle.driver, state.table),
		strings.Join(assignments, ", "),
		strings.Join(state.whereClauses, " and "),
	)
	query = ormRebindSQL(state.handle.driver, query)
	args = append(args, state.whereArgs...)
	result, err := db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return sqlResultToRuntime(result), nil
}

func ormDelete(state *ormQueryState) (runtime.Value, error) {
	db, err := state.handle.requireDB("delete")
	if err != nil {
		return nil, err
	}
	if err := state.requireSafeMutation("delete"); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(
		"delete from %s where %s",
		ormQuoteIdentifierMaybe(state.handle.driver, state.table),
		strings.Join(state.whereClauses, " and "),
	)
	query = ormRebindSQL(state.handle.driver, query)
	result, err := db.Exec(query, state.whereArgs...)
	if err != nil {
		return nil, err
	}
	return sqlResultToRuntime(result), nil
}

func (s *ormQueryState) requireSafeMutation(name string) error {
	if len(s.whereClauses) == 0 {
		return fmt.Errorf("%s requires where(...) or whereRaw(...) to avoid full-table mutation", name)
	}
	if s.orderBy != "" || s.limit != nil || s.offset != nil {
		return fmt.Errorf("%s does not support orderBy/limit/offset", name)
	}
	return nil
}

func ormApplyWhere(state *ormQueryState, value runtime.Value) (*ormQueryState, error) {
	if _, ok := value.(runtime.NullValue); ok {
		return state.clone(), nil
	}
	obj, err := ormRequireObjectArg("where", value)
	if err != nil {
		return nil, err
	}
	next := state.clone()
	for _, key := range ormSortedKeys(obj.Fields) {
		clause, args, err := ormConditionForValue(state.handle.driver, key, obj.Fields[key])
		if err != nil {
			return nil, err
		}
		next.whereClauses = append(next.whereClauses, clause)
		next.whereArgs = append(next.whereArgs, args...)
	}
	return next, nil
}

func ormConditionForValue(driver string, column string, value runtime.Value) (string, []any, error) {
	quoted := ormQuoteIdentifierMaybe(driver, strings.TrimSpace(column))
	if quoted == "" {
		return "", nil, fmt.Errorf("where expects non-empty column name")
	}

	switch typed := value.(type) {
	case runtime.NullValue:
		return quoted + " is null", nil, nil
	case *runtime.ArrayValue:
		if len(typed.Elements) == 0 {
			return "1 = 0", nil, nil
		}
		nonnull := make([]any, 0, len(typed.Elements))
		hasNull := false
		for _, elem := range typed.Elements {
			if _, ok := elem.(runtime.NullValue); ok {
				hasNull = true
				continue
			}
			plain, err := utils.RuntimeToPlainValue(elem)
			if err != nil {
				return "", nil, err
			}
			nonnull = append(nonnull, plain)
		}
		switch {
		case hasNull && len(nonnull) == 0:
			return quoted + " is null", nil, nil
		case hasNull:
			return fmt.Sprintf("(%s in (%s) or %s is null)", quoted, ormPlaceholders(len(nonnull)), quoted), nonnull, nil
		default:
			return fmt.Sprintf("%s in (%s)", quoted, ormPlaceholders(len(nonnull))), nonnull, nil
		}
	default:
		plain, err := utils.RuntimeToPlainValue(value)
		if err != nil {
			return "", nil, err
		}
		return quoted + " = ?", []any{plain}, nil
	}
}

func ormPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	parts := make([]string, count)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func ormRequireObjectArg(name string, value runtime.Value) (*runtime.ObjectValue, error) {
	obj, ok := value.(*runtime.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("%s expects object argument", name)
	}
	return obj, nil
}

func ormSortedKeys(fields map[string]runtime.Value) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func ormQuoteIdentifierMaybe(driver string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if value == "*" {
		return value
	}
	parts := strings.Split(value, ".")
	for _, part := range parts {
		if part == "*" {
			continue
		}
		if !ormIsSimpleIdentifier(part) {
			return value
		}
	}

	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "*" {
			quoted = append(quoted, "*")
			continue
		}
		quoted = append(quoted, ormQuoteIdentifier(driver, part))
	}
	return strings.Join(quoted, ".")
}

func ormQuoteIdentifier(driver string, value string) string {
	switch driver {
	case "mysql":
		return "`" + value + "`"
	default:
		return `"` + value + `"`
	}
}

func ormIsSimpleIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func (s *ormQueryState) buildSelectSQL() (string, []any) {
	selectExpr := s.selectExpr
	if strings.TrimSpace(selectExpr) == "" {
		selectExpr = "*"
	}
	var b strings.Builder
	b.WriteString("select ")
	b.WriteString(selectExpr)
	b.WriteString(" from ")
	b.WriteString(ormQuoteIdentifierMaybe(s.handle.driver, s.table))
	if len(s.whereClauses) > 0 {
		b.WriteString(" where ")
		b.WriteString(strings.Join(s.whereClauses, " and "))
	}
	if s.orderBy != "" {
		b.WriteString(" order by ")
		b.WriteString(s.orderBy)
	}
	if s.limit != nil {
		b.WriteString(fmt.Sprintf(" limit %d", *s.limit))
	}
	if s.offset != nil {
		b.WriteString(fmt.Sprintf(" offset %d", *s.offset))
	}
	return ormRebindSQL(s.handle.driver, b.String()), append([]any{}, s.whereArgs...)
}

func (s *ormQueryState) buildCountSQL() (string, []any) {
	selectQuery, queryArgs := s.buildSelectSQL()
	return ormRebindSQL(s.handle.driver, "select count(*) as count from ("+selectQuery+") __icoo_count"), queryArgs
}

func ormRebindSQL(driver string, query string) string {
	if driver != "pgx" {
		return query
	}
	var b strings.Builder
	index := 1
	for _, r := range query {
		if r == '?' {
			b.WriteString(fmt.Sprintf("$%d", index))
			index++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
