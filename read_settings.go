package main

import (
	"fmt"
	"os"
)

func main() {
	data, err := os.ReadFile("/Users/macbook/Code/9router-go/internal/handlers/dashboard/settings.go")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	lines := string(data)
	lineNum := 1
	inRange := false
	for _, c := range lines {
		if c == '\n' {
			lineNum++
		}
	}
	fmt.Printf("total lines: %d\n", lineNum)

	// Print lines 400-540
	parts := splitLines(string(data))
	for i, line := range parts {
		if i >= 399 && i < 540 {
			fmt.Printf("%d: %s\n", i+1, line)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	var current string
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
