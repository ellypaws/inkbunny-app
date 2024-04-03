package table

// SingleType is a Table using only one type for rows allowing for easier AddRows with fewer errors
type SingleType[T Ordered] struct {
	Table
}

// NewTableSingleType initialize TableSingleType object with defaults
func NewTableSingleType[T Ordered](width, height int, columnHeaders []string) *SingleType[T] {
	var defaultTypes []any
	var usedType T

	// set type to selected type
	for range columnHeaders {
		defaultTypes = append(defaultTypes, usedType)
	}

	t := &SingleType[T]{
		Table: *NewTable(width, height, columnHeaders),
	}

	_, err := t.Table.SetTypes(defaultTypes...)
	if err != nil {
		panic(err)
	}

	return t
}

// SetTypes overridden for TableSimple
func (r *SingleType[T]) SetTypes() {
}

func (r *SingleType[T]) AddRows(rows [][]T) *SingleType[T] {
	for _, row := range rows {
		var _row []any
		for _, cell := range row {
			_row = append(_row, cell)
		}
		r.rows = append(r.rows, _row)
	}

	r.applyFilter()
	r.setRowsUpdate()
	return r
}

func (r *SingleType[T]) MustAddRows(rows [][]T) *SingleType[T] {
	return r.AddRows(rows)
}
