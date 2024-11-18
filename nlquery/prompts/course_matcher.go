package prompts

import (
	"bufio"
	"os"
	"strings"
)

// CourseNameMatcher helps find exact course names from the database
type CourseNameMatcher struct {
	courseNames map[string]string // lowercase name -> exact name
	loaded     bool
}

// NewCourseNameMatcher creates a new CourseNameMatcher
func NewCourseNameMatcher() *CourseNameMatcher {
	return &CourseNameMatcher{
		courseNames: make(map[string]string),
	}
}

// LoadCourseNames loads course names from the file
func (cm *CourseNameMatcher) LoadCourseNames(filename string) error {
	if cm.loaded {
		return nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			cm.courseNames[strings.ToLower(name)] = name
		}
	}

	cm.loaded = true
	return scanner.Err()
}

// FindMatchingCourses returns SQL patterns for matching course names
func (cm *CourseNameMatcher) FindMatchingCourses(query string) []string {
	query = strings.ToLower(query)
	var patterns []string

	// Common course categories
	categories := map[string][]string{
		"medicine":       {"medicine", "medical", "surgery", "health", "pharm"},
		"engineering":    {"engineering", "engineer"},
		"science":        {"science", "sciences"},
		"arts":          {"art", "arts", "creative"},
		"management":     {"management", "business", "admin"},
		"education":      {"education", "teaching"},
		"agriculture":    {"agriculture", "agricultural", "farming"},
		"communication": {"communication", "media", "journalism"},
	}

	// Check for category matches first
	for _, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(query, keyword) {
				for courseName := range cm.courseNames {
					if strings.Contains(courseName, keyword) {
						patterns = append(patterns, "'%"+cm.courseNames[courseName]+"%'")
					}
				}
			}
		}
	}

	// If no category matches, try direct matching
	if len(patterns) == 0 {
		words := strings.Fields(query)
		for _, word := range words {
			if len(word) < 3 {
				continue
			}
			for courseName := range cm.courseNames {
				if strings.Contains(courseName, word) {
					patterns = append(patterns, "'%"+cm.courseNames[courseName]+"%'")
				}
			}
		}
	}

	// Add fallback pattern for code-based courses
	patterns = append(patterns, "'Course %'")

	return patterns
}
