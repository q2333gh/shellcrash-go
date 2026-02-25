package tui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// TableWidth is the total width of the menu
const TableWidth = 60

// Writer wraps an io.Writer for TUI output
type Writer struct {
	w io.Writer
}

// NewWriter creates a new TUI writer
func NewWriter(w io.Writer) *Writer {
	if w == nil {
		w = os.Stdout
	}
	return &Writer{w: w}
}

// LineBreak prints spacing between forms
func (tw *Writer) LineBreak() {
	fmt.Fprintln(tw.w)
	fmt.Fprintln(tw.w)
}

// SeparatorLine prints a separator line
// separatorType should be "=" or "-"
func (tw *Writer) SeparatorLine(separatorType string) {
	lenLimit := TableWidth - 1
	var line string
	if separatorType == "=" {
		line = strings.Repeat("=", lenLimit)
	} else {
		line = strings.Repeat("- ", lenLimit/2)
		if len(line) > lenLimit {
			line = line[:lenLimit]
		} else {
			for len(line) < lenLimit {
				line += "-"
			}
		}
	}
	fmt.Fprintf(tw.w, "%s||\n", line)
}

// ContentLine prints a content line with proper formatting and word wrapping
// Handles ANSI color codes and wide characters (CJK)
func (tw *Writer) ContentLine(text string) {
	if text == "" {
		fmt.Fprintf(tw.w, " \033[%dG||\n", TableWidth)
		return
	}

	textWidth := TableWidth - 3
	lines := wrapText(text, textWidth)
	for _, line := range lines {
		fmt.Fprintf(tw.w, " %s\033[0m\033[%dG||\n", line, TableWidth)
	}
}

// SubContentLine prints a sub-content line (indented)
func (tw *Writer) SubContentLine(text string) {
	if text == "" {
		fmt.Fprintf(tw.w, " \033[%dG||\n", TableWidth)
		return
	}
	tw.ContentLine("   " + text)
	fmt.Fprintf(tw.w, " \033[%dG||\n", TableWidth)
}

// wrapText wraps text to fit within the specified width
// Handles ANSI escape codes and wide characters
func wrapText(text string, width int) []string {
	var lines []string
	var currentLine strings.Builder
	var currentWidth int
	var lastColor string

	i := 0
	for i < len(text) {
		// Check for ANSI escape sequence
		if i < len(text) && text[i] == '\033' && i+1 < len(text) && text[i+1] == '[' {
			// Find the end of the ANSI sequence
			end := i + 2
			for end < len(text) && text[end] != 'm' {
				end++
			}
			if end < len(text) {
				end++ // include 'm'
				ansiSeq := text[i:end]
				currentLine.WriteString(ansiSeq)
				lastColor = ansiSeq
				i = end
				continue
			}
		}

		// Get the next rune
		r, size := utf8.DecodeRuneInString(text[i:])
		if r == utf8.RuneError {
			i++
			continue
		}

		// Calculate display width (CJK characters are 2 wide)
		charWidth := runeWidth(r)

		// Check if we need to wrap
		if currentWidth+charWidth > width {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			if lastColor != "" {
				currentLine.WriteString(lastColor)
			}
			currentWidth = 0
		}

		currentLine.WriteRune(r)
		currentWidth += charWidth
		i += size
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// runeWidth returns the display width of a rune
// CJK characters and other wide characters return 2, others return 1
func runeWidth(r rune) int {
	// CJK Unified Ideographs
	if r >= 0x4E00 && r <= 0x9FFF {
		return 2
	}
	// CJK Extension A
	if r >= 0x3400 && r <= 0x4DBF {
		return 2
	}
	// Hangul Syllables
	if r >= 0xAC00 && r <= 0xD7AF {
		return 2
	}
	// Hiragana and Katakana
	if r >= 0x3040 && r <= 0x30FF {
		return 2
	}
	// Fullwidth forms
	if r >= 0xFF00 && r <= 0xFFEF {
		return 2
	}
	return 1
}

// MsgAlert displays an alert message with a box and optional sleep
func (tw *Writer) MsgAlert(sleepSeconds int, messages ...string) {
	tw.LineBreak()
	tw.SeparatorLine("=")
	for _, msg := range messages {
		tw.ContentLine(msg)
	}
	tw.SeparatorLine("=")
	if sleepSeconds > 0 {
		// Note: In real usage, caller should handle sleep
		// We don't sleep here to keep the function testable
	}
}

// CompBox displays a complete box
func (tw *Writer) CompBox(messages ...string) {
	tw.LineBreak()
	tw.SeparatorLine("=")
	for _, msg := range messages {
		tw.ContentLine(msg)
	}
	tw.SeparatorLine("=")
}

// TopBox displays a top box (no bottom separator)
func (tw *Writer) TopBox(messages ...string) {
	tw.LineBreak()
	tw.SeparatorLine("=")
	for _, msg := range messages {
		tw.ContentLine(msg)
	}
}

// BtmBox displays a bottom box (no top separator)
func (tw *Writer) BtmBox(messages ...string) {
	for _, msg := range messages {
		tw.ContentLine(msg)
	}
	tw.SeparatorLine("=")
}

// ListBox displays a numbered list
func (tw *Writer) ListBox(items []string, suffix string) {
	for i, item := range items {
		tw.ContentLine(fmt.Sprintf("%d) %s%s", i+1, item, suffix))
	}
}

// CommonSuccess displays a success message
func (tw *Writer) CommonSuccess(message string) {
	if message == "" {
		message = "操作成功"
	}
	tw.MsgAlert(1, "\033[32m"+message+"\033[0m")
}

// CommonFailed displays a failure message
func (tw *Writer) CommonFailed(message string) {
	if message == "" {
		message = "操作失败"
	}
	tw.MsgAlert(1, "\033[31m"+message+"\033[0m")
}

// CommonBack displays the back option
func (tw *Writer) CommonBack(message string) {
	if message == "" {
		message = "返回上级菜单"
	}
	tw.ContentLine(fmt.Sprintf("0) %s", message))
	tw.SeparatorLine("=")
}

// ErrorNum displays a number error message
func (tw *Writer) ErrorNum(message string) {
	if message == "" {
		message = "输入错误，请输入正确的数字！"
	}
	tw.MsgAlert(1, "\033[31m"+message+"\033[0m")
}

// ErrorLetter displays a letter error message
func (tw *Writer) ErrorLetter(message string) {
	if message == "" {
		message = "输入错误，请输入正确的字母！"
	}
	tw.MsgAlert(1, "\033[31m"+message+"\033[0m")
}

// ErrorInput displays an input error message
func (tw *Writer) ErrorInput(message string) {
	if message == "" {
		message = "输入错误！"
	}
	tw.MsgAlert(1, "\033[31m"+message+"\033[0m")
}

// CancelBack displays a cancel message
func (tw *Writer) CancelBack(message string) {
	if message == "" {
		message = "已取消"
	}
	tw.SeparatorLine("-")
	tw.ContentLine(message)
}
