package cli

import (
	"os"
	"strconv"
	"strings"
)

func formatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	widths := make([]int, len(headers))
	minWidths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = cellWidth(header)
		minWidths[i] = cellWidth(header)
		if minWidths[i] < 3 {
			minWidths[i] = 3
		}
	}
	for _, row := range rows {
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			if cellWidth(cell) > widths[i] {
				widths[i] = cellWidth(cell)
			}
		}
	}
	if maxWidth := terminalTableWidth(); maxWidth > 0 {
		shrinkTableWidths(widths, minWidths, maxWidth)
	}

	var builder strings.Builder
	writeTableSeparator(&builder, widths)
	writeTableCells(&builder, headers, widths)
	writeTableSeparator(&builder, widths)
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			if i < len(row) {
				cells[i] = row[i]
			}
		}
		writeTableCells(&builder, cells, widths)
	}
	writeTableSeparator(&builder, widths)
	return builder.String()
}

func terminalTableWidth() int {
	columns := strings.TrimSpace(os.Getenv("COLUMNS"))
	if columns == "" {
		return 0
	}
	width, err := strconv.Atoi(columns)
	if err != nil || width <= 0 {
		return 0
	}
	if width < 40 {
		return 40
	}
	return width
}

func shrinkTableWidths(widths []int, minWidths []int, maxWidth int) {
	for tableLineWidth(widths) > maxWidth {
		index := -1
		for i, width := range widths {
			if width <= minWidths[i] {
				continue
			}
			if index == -1 || width > widths[index] {
				index = i
			}
		}
		if index == -1 {
			return
		}
		widths[index]--
	}
}

func tableLineWidth(widths []int) int {
	width := 1
	for _, columnWidth := range widths {
		width += columnWidth + 3
	}
	return width
}

func writeTableSeparator(builder *strings.Builder, widths []int) {
	builder.WriteByte('+')
	for _, width := range widths {
		builder.WriteString(strings.Repeat("-", width+2))
		builder.WriteByte('+')
	}
	builder.WriteByte('\n')
}

func writeTableCells(builder *strings.Builder, cells []string, widths []int) {
	builder.WriteByte('|')
	for i, cell := range cells {
		value := truncateTableCell(cell, widths[i])
		builder.WriteByte(' ')
		builder.WriteString(value)
		builder.WriteString(strings.Repeat(" ", widths[i]-cellWidth(value)))
		builder.WriteString(" |")
	}
	builder.WriteByte('\n')
}

func cellWidth(value string) int {
	return len([]rune(value))
}

func truncateTableCell(value string, width int) string {
	if cellWidth(value) <= width {
		return value
	}
	if width <= 0 {
		return ""
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	runes := []rune(value)
	return string(runes[:width-3]) + "..."
}
