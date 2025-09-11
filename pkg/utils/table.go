package utils

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Table provides a simple way to display tabular data
type Table struct {
	headers []string
	rows    [][]string
	writer  *tabwriter.Writer
}

// NewTable creates a new table with the given headers
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		writer:  tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(values ...string) {
	// Ensure we have the right number of columns
	row := make([]string, len(t.headers))
	for i, value := range values {
		if i < len(row) {
			row[i] = value
		}
	}
	// Fill any missing values with "-"
	for i := len(values); i < len(row); i++ {
		row[i] = "-"
	}
	t.rows = append(t.rows, row)
}

// Render displays the table
func (t *Table) Render() {
	// Print headers
	fmt.Fprintf(t.writer, "%s\n", strings.Join(t.headers, "\t"))
	
	// Print rows
	for _, row := range t.rows {
		fmt.Fprintf(t.writer, "%s\n", strings.Join(row, "\t"))
	}
	
	t.writer.Flush()
}

// RenderWithSeparator displays the table with a separator line under headers
func (t *Table) RenderWithSeparator() {
	// Print headers
	fmt.Fprintf(t.writer, "%s\n", strings.Join(t.headers, "\t"))
	
	// Print separator
	separators := make([]string, len(t.headers))
	for i, header := range t.headers {
		separators[i] = strings.Repeat("-", len(header))
	}
	fmt.Fprintf(t.writer, "%s\n", strings.Join(separators, "\t"))
	
	// Print rows
	for _, row := range t.rows {
		fmt.Fprintf(t.writer, "%s\n", strings.Join(row, "\t"))
	}
	
	t.writer.Flush()
}

// SetMinWidth sets minimum column widths
func (t *Table) SetMinWidth(minWidth int) {
	t.writer = tabwriter.NewWriter(os.Stdout, minWidth, 0, 2, ' ', 0)
}

// Count returns the number of rows in the table (excluding header)
func (t *Table) Count() int {
	return len(t.rows)
}