package main

import (
	"fmt"
	"strings"
)

// strips the frontmatter from a markdown file and returns
// 1. content without frontmatter
// 2. frontmatter as an object
func SplitFrontmatterMarkdown(content string) (string, map[string]interface{}) {
	fm := make(map[string]interface{})
	lines := strings.Split(content, "\n")

	// parse frontmatter to find k/v pairs
	if strings.HasPrefix(lines[0], "---") {
		for i, line := range lines {
			if i == 0 {
				continue
			}

			if strings.HasPrefix(line, "---") {
				// return end of content and frontmatter if we hit another ---
				// +2 because first line after last --- is blank
				return strings.Join(lines[i+2:], "\n"), fm
			} else if strings.Contains(line, ":") {
				// otherwise, parse valid line as k/v pair
				parts := strings.Split(line, ":")
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				fm[key] = value
			}
		}
	}

	// if no frontmatter found, return the content as is
	fmt.Println("no frontmatter found")
	return content, fm
}
