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
	seenPatterns := make(map[string]bool)

	// Common course categories
	categories := map[string][]string{
		"medicine":       {"medicine", "medical", "surgery", "health", "pharm", "anatomy", "optometry", "biomedical", "orthopedic", "physiotherapy"},
		"engineering":    {"engineering", "engineer", "technology", "mechanical", "electrical", "electronic", "civil", "aerospace", "automotive", "chemical", "petroleum"},
		"science":        {"science", "sciences", "biology", "chemistry", "physics", "mathematics", "statistics", "biochemistry", "biotechnology", "microbiology", "geology", "environmental"},
		"arts":          {"art", "arts", "creative", "theatre", "music", "cultural", "literature", "language", "linguistics"},
		"management":     {"management", "business", "admin", "accounting", "finance", "economics", "banking", "entrepreneurship", "logistics", "commerce"},
		"education":      {"education", "teaching", "pedagogy", "curriculum", "instruction"},
		"agriculture":    {"agriculture", "agricultural", "farming", "agronomy", "agribusiness", "crop", "animal science", "fisheries", "forestry"},
		"communication": {"communication", "media", "journalism", "broadcasting", "public relations", "mass communication"},
		"computing":     {"computer", "computing", "software", "information technology", "data", "cybersecurity", "artificial intelligence"},
		"social_sciences": {"sociology", "psychology", "anthropology", "political science", "international relations", "social work", "geography"},
		"languages":      {"english", "french", "arabic", "hausa", "yoruba", "igbo", "linguistics", "literature"},
		"religious_studies": {"islamic studies", "religious studies", "theology", "divinity", "christian religious studies"},
		"architecture":   {"architecture", "building", "construction", "estate management", "quantity surveying", "urban planning"},
		"law":           {"law", "legal studies", "jurisprudence"},
		"environmental": {"environmental", "ecology", "conservation", "climate", "biodiversity"},
		"hospitality":   {"hospitality", "tourism", "hotel management", "catering"},
	}

	// Helper function to add unique patterns
	addPattern := func(pattern string) {
		if !seenPatterns[pattern] {
			patterns = append(patterns, pattern)
			seenPatterns[pattern] = true
		}
	}

	// Check for exact matches first
	for courseName := range cm.courseNames {
		if strings.Contains(query, strings.ToLower(courseName)) {
			addPattern("'%" + cm.courseNames[courseName] + "%'")
		}
	}

	// Check for category matches
	for categoryName, keywords := range categories {
		categoryMatched := false
		for _, keyword := range keywords {
			if strings.Contains(query, keyword) {
				categoryMatched = true
				// Add all courses that belong to this category
				for courseName := range cm.courseNames {
					courseLower := strings.ToLower(courseName)
					// Check if the course contains the category name or any of its keywords
					if strings.Contains(courseLower, categoryName) {
						addPattern("'%" + cm.courseNames[courseName] + "%'")
					} else {
						// If course doesn't contain category name, check keywords
						for _, catKeyword := range keywords {
							if strings.Contains(courseLower, catKeyword) {
								addPattern("'%" + cm.courseNames[courseName] + "%'")
								break
							}
						}
					}
				}
				break // Break after finding a matching keyword for this category
			}
		}
		// If this category matched, no need to check other categories
		if categoryMatched {
			break
		}
	}

	// If no matches found, try word-by-word matching
	if len(patterns) == 0 {
		words := strings.Fields(query)
		for _, word := range words {
			if len(word) < 3 { // Skip very short words
				continue
			}
			for courseName := range cm.courseNames {
				if strings.Contains(strings.ToLower(courseName), word) {
					addPattern("'%" + cm.courseNames[courseName] + "%'")
				}
			}
		}
	}

	// Add fallback pattern for very generic queries
	if len(patterns) == 0 && (strings.Contains(query, "course") || strings.Contains(query, "program")) {
		addPattern("'%COURSE%'")
	}

	return patterns
}
