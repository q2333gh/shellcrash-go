package tui

import (
	"bytes"
	"strings"
	"testing"
)

func TestSeparatorLine(t *testing.T) {
	tests := []struct {
		name          string
		separatorType string
		wantContains  string
	}{
		{
			name:          "equals separator",
			separatorType: "=",
			wantContains:  "===",
		},
		{
			name:          "dash separator",
			separatorType: "-",
			wantContains:  "- ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewWriter(&buf)
			tw.SeparatorLine(tt.separatorType)

			output := buf.String()
			if !strings.Contains(output, tt.wantContains) {
				t.Errorf("SeparatorLine() output = %q, want to contain %q", output, tt.wantContains)
			}
			if !strings.HasSuffix(output, "||\n") {
				t.Errorf("SeparatorLine() output should end with ||\\n, got %q", output)
			}
		})
	}
}

func TestContentLine(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		wantContains string
	}{
		{
			name:         "simple text",
			text:         "Hello World",
			wantContains: "Hello World",
		},
		{
			name:         "empty text",
			text:         "",
			wantContains: "||",
		},
		{
			name:         "text with ANSI color",
			text:         "\033[32mGreen Text\033[0m",
			wantContains: "Green Text",
		},
		{
			name:         "Chinese text",
			text:         "你好世界",
			wantContains: "你好世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := NewWriter(&buf)
			tw.ContentLine(tt.text)

			output := buf.String()
			if !strings.Contains(output, tt.wantContains) {
				t.Errorf("ContentLine() output = %q, want to contain %q", output, tt.wantContains)
			}
			if !strings.Contains(output, "||") {
				t.Errorf("ContentLine() output should contain ||, got %q", output)
			}
		})
	}
}

func TestLineBreak(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	tw.LineBreak()

	output := buf.String()
	if output != "\n\n" {
		t.Errorf("LineBreak() output = %q, want \\n\\n", output)
	}
}

func TestCompBox(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	tw.CompBox("Line 1", "Line 2")

	output := buf.String()
	if !strings.Contains(output, "Line 1") {
		t.Errorf("CompBox() should contain 'Line 1', got %q", output)
	}
	if !strings.Contains(output, "Line 2") {
		t.Errorf("CompBox() should contain 'Line 2', got %q", output)
	}
	// Should have separators at top and bottom
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 4 {
		t.Errorf("CompBox() should have at least 4 lines (separator, content, content, separator), got %d", len(lines))
	}
}

func TestTopBox(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	tw.TopBox("Header")

	output := buf.String()
	if !strings.Contains(output, "Header") {
		t.Errorf("TopBox() should contain 'Header', got %q", output)
	}
	// Should start with separator but not end with one
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Errorf("TopBox() should have at least 2 lines, got %d", len(lines))
	}
}

func TestBtmBox(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	tw.BtmBox("Footer")

	output := buf.String()
	if !strings.Contains(output, "Footer") {
		t.Errorf("BtmBox() should contain 'Footer', got %q", output)
	}
	// Should end with separator
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Errorf("BtmBox() should have at least 2 lines, got %d", len(lines))
	}
}

func TestListBox(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	items := []string{"Item A", "Item B", "Item C"}
	tw.ListBox(items, "")

	output := buf.String()
	if !strings.Contains(output, "1) Item A") {
		t.Errorf("ListBox() should contain '1) Item A', got %q", output)
	}
	if !strings.Contains(output, "2) Item B") {
		t.Errorf("ListBox() should contain '2) Item B', got %q", output)
	}
	if !strings.Contains(output, "3) Item C") {
		t.Errorf("ListBox() should contain '3) Item C', got %q", output)
	}
}

func TestCommonSuccess(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	tw.CommonSuccess("操作成功")

	output := buf.String()
	if !strings.Contains(output, "操作成功") {
		t.Errorf("CommonSuccess() should contain message, got %q", output)
	}
	if !strings.Contains(output, "\033[32m") {
		t.Errorf("CommonSuccess() should contain green color code, got %q", output)
	}
}

func TestCommonFailed(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	tw.CommonFailed("操作失败")

	output := buf.String()
	if !strings.Contains(output, "操作失败") {
		t.Errorf("CommonFailed() should contain message, got %q", output)
	}
	if !strings.Contains(output, "\033[31m") {
		t.Errorf("CommonFailed() should contain red color code, got %q", output)
	}
}

func TestCommonBack(t *testing.T) {
	var buf bytes.Buffer
	tw := NewWriter(&buf)
	tw.CommonBack("返回")

	output := buf.String()
	if !strings.Contains(output, "0) 返回") {
		t.Errorf("CommonBack() should contain '0) 返回', got %q", output)
	}
}

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		name      string
		r         rune
		wantWidth int
	}{
		{"ASCII letter", 'A', 1},
		{"ASCII space", ' ', 1},
		{"Chinese character", '中', 2},
		{"Japanese Hiragana", 'あ', 2},
		{"Korean Hangul", '한', 2},
		{"Fullwidth number", '１', 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runeWidth(tt.r)
			if got != tt.wantWidth {
				t.Errorf("runeWidth(%q) = %d, want %d", tt.r, got, tt.wantWidth)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		width     int
		wantLines int
	}{
		{
			name:      "short text",
			text:      "Hello",
			width:     20,
			wantLines: 1,
		},
		{
			name:      "text that needs wrapping",
			text:      "This is a very long line that should be wrapped",
			width:     20,
			wantLines: 3,
		},
		{
			name:      "Chinese text",
			text:      "这是一个很长的中文句子需要换行",
			width:     10,
			wantLines: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := wrapText(tt.text, tt.width)
			if len(lines) != tt.wantLines {
				t.Errorf("wrapText() returned %d lines, want %d lines. Lines: %v", len(lines), tt.wantLines, lines)
			}
		})
	}
}
