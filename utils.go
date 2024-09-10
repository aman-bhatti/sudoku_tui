package main

import "strings"

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	leftPadding := (width - len(text)) / 2
	return strings.Repeat(" ", leftPadding) + text
}
