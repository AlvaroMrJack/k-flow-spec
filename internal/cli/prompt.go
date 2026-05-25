package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var promptReader = bufio.NewScanner(os.Stdin)

func prompt(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("? %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("? %s: ", label)
	}
	if !promptReader.Scan() {
		return defaultVal
	}
	text := strings.TrimSpace(promptReader.Text())
	if text == "" {
		return defaultVal
	}
	return text
}

func promptBool(label string, defaultVal bool) bool {
	suffix := " (y/N)"
	if defaultVal {
		suffix = " (Y/n)"
	}
	fmt.Printf("? %s%s: ", label, suffix)
	if !promptReader.Scan() {
		return defaultVal
	}
	text := strings.ToLower(strings.TrimSpace(promptReader.Text()))
	if text == "" {
		return defaultVal
	}
	return text == "y" || text == "yes"
}

func promptInt(label string, defaultVal int) int {
	s := prompt(label, fmt.Sprintf("%d", defaultVal))
	if s == "" {
		return defaultVal
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return defaultVal
	}
	return n
}

func promptSelect(label string, options []string, defaultIdx int) int {
	fmt.Printf("? %s:\n", label)
	for i, opt := range options {
		mark := " "
		if i == defaultIdx {
			mark = ">"
		}
		fmt.Printf("  %s %d) %s\n", mark, i+1, opt)
	}
	fmt.Printf("  Elige [%d]: ", defaultIdx+1)
	if !promptReader.Scan() {
		return defaultIdx
	}
	text := strings.TrimSpace(promptReader.Text())
	if text == "" {
		return defaultIdx
	}
	var n int
	if _, err := fmt.Sscanf(text, "%d", &n); err != nil {
		return defaultIdx
	}
	if n < 1 || n > len(options) {
		return defaultIdx
	}
	return n - 1
}
