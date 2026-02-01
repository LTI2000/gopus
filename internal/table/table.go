// Package table provides utilities for rendering formatted tables in the terminal.
package table

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Column represents a table column with its configuration.
type Column struct {
	Header   string
	MinWidth int
	MaxWidth int
	Align    Alignment
}

// Alignment specifies how content should be aligned within a column.
type Alignment int

const (
	// AlignLeft aligns content to the left.
	AlignLeft Alignment = iota
	// AlignRight aligns content to the right.
	AlignRight
)

// Table represents a table with columns and rows.
type Table struct {
	columns []Column
	rows    [][]string
	widths  []int
}

// New creates a new table with the specified columns.
func New(columns ...Column) *Table {
	t := &Table{
		columns: columns,
		widths:  make([]int, len(columns)),
	}

	// Initialize widths with header lengths and minimum widths
	for i, col := range columns {
		t.widths[i] = len(col.Header)
		if col.MinWidth > t.widths[i] {
			t.widths[i] = col.MinWidth
		}
	}

	return t
}

// AddRow adds a row of values to the table.
func (t *Table) AddRow(values ...string) {
	// Ensure we have the right number of values
	row := make([]string, len(t.columns))
	for i := range t.columns {
		if i < len(values) {
			row[i] = values[i]
		}
	}

	// Update column widths based on content
	for i, val := range row {
		if len(val) > t.widths[i] {
			t.widths[i] = len(val)
		}
	}

	t.rows = append(t.rows, row)
}

// calculateFinalWidths applies max width constraints and returns final widths.
func (t *Table) calculateFinalWidths() []int {
	widths := make([]int, len(t.widths))
	copy(widths, t.widths)

	for i, col := range t.columns {
		if col.MaxWidth > 0 && widths[i] > col.MaxWidth {
			widths[i] = col.MaxWidth
		}
	}

	return widths
}

// truncate truncates a string to the specified width, adding ellipsis if needed.
func truncate(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

// formatCell formats a cell value according to column width and alignment.
func formatCell(value string, width int, align Alignment) string {
	value = truncate(value, width)
	switch align {
	case AlignRight:
		return fmt.Sprintf("%*s", width, value)
	default:
		return fmt.Sprintf("%-*s", width, value)
	}
}

// RenderHeader returns the formatted header row.
func (t *Table) RenderHeader() string {
	widths := t.calculateFinalWidths()
	var parts []string

	for i, col := range t.columns {
		parts = append(parts, formatCell(col.Header, widths[i], col.Align))
	}

	return "\033[1m" + strings.Join(parts, " │ ") + "\033[0m"
}

// RenderSeparator returns the separator line between header and rows.
func (t *Table) RenderSeparator() string {
	widths := t.calculateFinalWidths()
	var parts []string

	for _, w := range widths {
		parts = append(parts, strings.Repeat("─", w))
	}

	return strings.Join(parts, "─┼─")
}

// RenderRow returns a formatted row at the specified index.
func (t *Table) RenderRow(index int) string {
	if index < 0 || index >= len(t.rows) {
		return ""
	}

	widths := t.calculateFinalWidths()
	row := t.rows[index]
	var parts []string

	for i, col := range t.columns {
		parts = append(parts, formatCell(row[i], widths[i], col.Align))
	}

	return strings.Join(parts, " │ ")
}

// RowCount returns the number of rows in the table.
func (t *Table) RowCount() int {
	return len(t.rows)
}

// Render returns the complete table as a string.
func (t *Table) Render() string {
	var lines []string

	lines = append(lines, t.RenderHeader())
	lines = append(lines, t.RenderSeparator())

	for i := range t.rows {
		lines = append(lines, t.RenderRow(i))
	}

	return strings.Join(lines, "\n")
}

// PrintOptions configures how the table is printed.
type PrintOptions struct {
	// Indent is the prefix added to each line (e.g., "  " for two-space indent).
	Indent string
	// HighlightColumn is the index of the column to highlight (0-based), or -1 for none.
	HighlightColumn int
	// HighlightColor is the ANSI color code for highlighting (e.g., "33" for yellow).
	HighlightColor string
	// Writer is the output destination. Defaults to os.Stdout if nil.
	Writer io.Writer
}

// DefaultPrintOptions returns default print options.
func DefaultPrintOptions() PrintOptions {
	return PrintOptions{
		Indent:          "  ",
		HighlightColumn: -1,
		HighlightColor:  "33",
		Writer:          os.Stdout,
	}
}

// Print outputs the table to the configured writer with the specified options.
func (t *Table) Print(opts PrintOptions) {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	fmt.Fprintln(opts.Writer)
	fmt.Fprintf(opts.Writer, "%s%s\n", opts.Indent, t.RenderHeader())
	fmt.Fprintf(opts.Writer, "%s%s\n", opts.Indent, t.RenderSeparator())

	for i := 0; i < t.RowCount(); i++ {
		row := t.RenderRow(i)
		if opts.HighlightColumn >= 0 && opts.HighlightColumn < len(t.columns) {
			// Split the row to apply highlighting to the specified column
			parts := strings.SplitN(row, " │ ", opts.HighlightColumn+2)
			if len(parts) > opts.HighlightColumn {
				var highlighted strings.Builder
				for j, part := range parts {
					if j > 0 {
						highlighted.WriteString(" │ ")
					}
					if j == opts.HighlightColumn {
						highlighted.WriteString(fmt.Sprintf("\033[%sm%s\033[0m", opts.HighlightColor, part))
					} else {
						highlighted.WriteString(part)
					}
				}
				row = highlighted.String()
			}
		}
		fmt.Fprintf(opts.Writer, "%s%s\n", opts.Indent, row)
	}
	fmt.Fprintln(opts.Writer)
}
