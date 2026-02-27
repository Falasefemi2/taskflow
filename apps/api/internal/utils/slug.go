package utils

import (
	"regexp"
	"strings"
)

func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")
	reg2 := regexp.MustCompile(`-+`)
	slug = reg2.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
