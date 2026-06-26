package tui

import "strings"

// This file is a tiny hand-rolled ASCII-art font, so the dashboard brand and the
// timer readout can be big and unmistakable without pulling in a figlet library.
// Glyphs are tall (glyphHeight rows) with doubled strokes so the text reads large
// on screen. bigText stitches glyphs side by side with a one-column gap.

const glyphHeight = 7

// brandFont holds the letters used by the FOCUSBET wordmark. Only the needed
// letters are defined; bigText renders unknown runes as a blank gap.
var brandFont = map[rune][]string{
	'F': {
		"██████",
		"██    ",
		"██    ",
		"█████ ",
		"██    ",
		"██    ",
		"██    ",
	},
	'O': {
		" ████ ",
		"██  ██",
		"██  ██",
		"██  ██",
		"██  ██",
		"██  ██",
		" ████ ",
	},
	'C': {
		" █████",
		"██    ",
		"██    ",
		"██    ",
		"██    ",
		"██    ",
		" █████",
	},
	'U': {
		"██  ██",
		"██  ██",
		"██  ██",
		"██  ██",
		"██  ██",
		"██  ██",
		" ████ ",
	},
	'S': {
		" █████",
		"██    ",
		"██    ",
		" ████ ",
		"    ██",
		"    ██",
		"█████ ",
	},
	'B': {
		"█████ ",
		"██  ██",
		"██  ██",
		"█████ ",
		"██  ██",
		"██  ██",
		"█████ ",
	},
	'E': {
		"██████",
		"██    ",
		"██    ",
		"█████ ",
		"██    ",
		"██    ",
		"██████",
	},
	'T': {
		"██████",
		"  ██  ",
		"  ██  ",
		"  ██  ",
		"  ██  ",
		"  ██  ",
		"  ██  ",
	},
}

// digitFont holds the big glyphs for the timer readout: 0-9 and the colon.
var digitFont = map[rune][]string{
	'0': {" ████ ", "██  ██", "██  ██", "██  ██", "██  ██", "██  ██", " ████ "},
	'1': {"  ██  ", " ███  ", "  ██  ", "  ██  ", "  ██  ", "  ██  ", "██████"},
	'2': {" ████ ", "██  ██", "    ██", "   ██ ", "  ██  ", " ██   ", "██████"},
	'3': {" ████ ", "██  ██", "    ██", "  ███ ", "    ██", "██  ██", " ████ "},
	'4': {"██  ██", "██  ██", "██  ██", "██████", "    ██", "    ██", "    ██"},
	'5': {"██████", "██    ", "█████ ", "    ██", "    ██", "██  ██", " ████ "},
	'6': {" ████ ", "██    ", "██    ", "█████ ", "██  ██", "██  ██", " ████ "},
	'7': {"██████", "    ██", "   ██ ", "  ██  ", " ██   ", " ██   ", " ██   "},
	'8': {" ████ ", "██  ██", "██  ██", " ████ ", "██  ██", "██  ██", " ████ "},
	'9': {" ████ ", "██  ██", "██  ██", " █████", "    ██", "    ██", " ████ "},
	':': {"  ", "  ", "██", "  ", "██", "  ", "  "},
}

// bigText renders s in the given font as a multi-line ASCII-art string. Runes
// absent from the font are rendered as a blank glyph-width gap.
func bigText(s string, font map[rune][]string) string {
	rows := make([]strings.Builder, glyphHeight)
	for i, r := range s {
		glyph, ok := font[r]
		if !ok {
			glyph = blankGlyph(font)
		}
		for row := 0; row < glyphHeight; row++ {
			if i > 0 {
				rows[row].WriteString(" ")
			}
			rows[row].WriteString(glyph[row])
		}
	}
	lines := make([]string, glyphHeight)
	for i := range rows {
		lines[i] = rows[i].String()
	}
	return strings.Join(lines, "\n")
}

// blankGlyph returns a glyph-width run of spaces, sized to the font's glyphs so
// spacing stays even for unknown runes (e.g. a literal space).
func blankGlyph(font map[rune][]string) []string {
	w := 6 // default glyph width
	for _, g := range font {
		w = len([]rune(g[0]))
		break
	}
	blank := make([]string, glyphHeight)
	for i := range blank {
		blank[i] = strings.Repeat(" ", w)
	}
	return blank
}
