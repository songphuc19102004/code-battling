package service

import (
	"strings"
)

// NormalizeLanguage returns the standard name for the input progrmaming language
func NormalizeLanguage(lang string) string {

	lang = strings.ToLower(lang)

	languageMap := map[string]string{

		"js":          "js",
		"jscript":     "js",
		"javscript":   "js",
		"javsscript":  "js",
		"javascipt":   "js",
		"javasript":   "js",
		"javascript":  "js",
		"java script": "js",
		"jscipt":      "js",

		"python":  "python",
		"pyt":     "python",
		"pyn":     "python",
		"pythn":   "python",
		"phyton":  "python",
		"py":      "python",
		"py thon": "python",
		"pthon":   "python",

		"go":      "go",
		"golang":  "go",
		"gol":     "go",
		"goo":     "go",
		"g o":     "go",
		"golangg": "go",
	}

	if normalized, ok := languageMap[lang]; ok {
		return normalized
	}

	return lang
}

// GenerateCodeRunCmd will generate a command that combine Run Command and Code
func GenerateCodeRunCmd(runCmd, code string) string {

	return ""
}
