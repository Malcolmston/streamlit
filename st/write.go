package st

import (
	"fmt"
	"reflect"
)

// Write is the Swiss-army display method, mirroring Streamlit's st.write. It
// inspects each argument's dynamic type and dispatches to the most appropriate
// specialised renderer:
//
//   - string            -> [Container.Markdown]
//   - error             -> [Container.Error]
//   - fmt.Stringer      -> [Container.Text] (via String)
//   - struct / slice of
//     structs / [][]string -> [Container.Table]
//   - map / other slice  -> [Container.JSON]
//   - everything else    -> [Container.Text] (via fmt.Sprint)
//
// Passing multiple arguments renders each in turn.
func (c *Container) Write(args ...any) {
	for _, arg := range args {
		c.writeOne(arg)
	}
}

// writeOne renders a single value according to the Write dispatch rules.
func (c *Container) writeOne(arg any) {
	switch v := arg.(type) {
	case nil:
		c.Text("<nil>")
		return
	case string:
		c.Markdown(v)
		return
	case error:
		c.Error(v.Error())
		return
	case fmt.Stringer:
		c.Text(v.String())
		return
	}

	rv := reflect.ValueOf(arg)
	switch rv.Kind() {
	case reflect.Struct:
		c.Table(arg)
	case reflect.Slice, reflect.Array:
		if isTabular(rv.Type()) {
			c.Table(arg)
		} else {
			c.JSON(arg)
		}
	case reflect.Map:
		c.JSON(arg)
	case reflect.Ptr:
		if rv.IsNil() {
			c.Text("<nil>")
		} else {
			c.writeOne(rv.Elem().Interface())
		}
	default:
		c.Text(fmt.Sprint(arg))
	}
}

// isTabular reports whether a slice/array type should render as a table:
// either a slice of structs or a slice of []string ([][]string).
func isTabular(t reflect.Type) bool {
	el := t.Elem()
	if el.Kind() == reflect.Struct {
		return true
	}
	if el.Kind() == reflect.Slice && el.Elem().Kind() == reflect.String {
		return true
	}
	return false
}

// Table renders tabular data. It accepts:
//
//   - [][]string, where the first row is treated as the header;
//   - a slice of structs, where exported field names become the header and
//     each element becomes a row;
//   - a single struct, rendered as a one-row table.
//
// An optional caption is shown beneath the table. Anything unsupported falls
// back to a JSON view.
func (c *Container) Table(data any, caption ...string) {
	c.table(data, optKey(caption), false)
}

// DataFrame renders tabular data like [Container.Table] but adds a client-side
// sorting hint, so column headers can be clicked to sort the rows in the
// browser. It mirrors Streamlit's st.dataframe entry point and accepts an
// optional caption.
func (c *Container) DataFrame(data any, caption ...string) {
	c.table(data, optKey(caption), true)
}

// table is the shared implementation for [Container.Table] and
// [Container.DataFrame].
func (c *Container) table(data any, caption string, sortable bool) {
	header, rows, ok := tableData(data)
	if !ok {
		c.JSON(data)
		return
	}
	// Convert rows to []any for JSON friendliness.
	jsonRows := make([]any, len(rows))
	for i, r := range rows {
		cells := make([]any, len(r))
		for j, cell := range r {
			cells[j] = cell
		}
		jsonRows[i] = cells
	}
	hdr := make([]any, len(header))
	for i, h := range header {
		hdr[i] = h
	}
	c.add("table", props{"header": hdr, "rows": jsonRows, "caption": caption, "sortable": sortable})
}

// tableData extracts a header and string-cell rows from supported inputs.
func tableData(data any) (header []string, rows [][]string, ok bool) {
	rv := reflect.ValueOf(data)
	switch rv.Kind() {
	case reflect.Struct:
		h, row := structRow(rv)
		return h, [][]string{row}, true
	case reflect.Slice, reflect.Array:
		el := rv.Type().Elem()
		switch {
		case el.Kind() == reflect.Struct:
			for i := 0; i < rv.Len(); i++ {
				h, row := structRow(rv.Index(i))
				if i == 0 {
					header = h
				}
				rows = append(rows, row)
			}
			return header, rows, true
		case el.Kind() == reflect.Slice && el.Elem().Kind() == reflect.String:
			if rv.Len() == 0 {
				return nil, nil, true
			}
			header = toStringSlice(rv.Index(0))
			for i := 1; i < rv.Len(); i++ {
				rows = append(rows, toStringSlice(rv.Index(i)))
			}
			return header, rows, true
		}
	}
	return nil, nil, false
}

// structRow returns the exported field names and stringified values of a
// struct value.
func structRow(rv reflect.Value) (header, row []string) {
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		header = append(header, f.Name)
		row = append(row, fmt.Sprint(rv.Field(i).Interface()))
	}
	return header, row
}

// toStringSlice stringifies each element of a []string-kinded value.
func toStringSlice(rv reflect.Value) []string {
	out := make([]string, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = fmt.Sprint(rv.Index(i).Interface())
	}
	return out
}
