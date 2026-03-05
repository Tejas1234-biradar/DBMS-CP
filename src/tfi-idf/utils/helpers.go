package utils

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/abadojack/whatlanggo"
)

var (
	filenameSplitRegex = regexp.MustCompile(`[-_./\s]+`)
	urlSplitRegex      = regexp.MustCompile(`[.,\-_\/+\:\(\)]`)
	whitespaceRegex    = regexp.MustCompile(`\s+`)
	bracketRegex       = regexp.MustCompile(`\[.*?\]`)
)

var StopWords = map[string]struct{}{
	"the": {}, "is": {}, "and": {}, "a": {}, "an": {}, "to": {},
	"in": {}, "of": {}, "for": {}, "on": {}, "with": {}, "as": {},
	"by": {}, "at": {}, "from": {}, "that": {}, "this": {}, "it": {},
}

type HTMLData struct {
	Title       string
	Description string
	SummaryText string
	Text        []string
	Language    string
}

func SplitName(filename string) []string {

	parts := filenameSplitRegex.Split(filename, -1)

	result := []string{}

	for _, part := range parts {

		part = strings.ToLower(part)

		if part == "" {
			continue
		}

		if _, exists := FileTypes[part]; exists {
			continue
		}

		if strings.Contains(part, "px") {
			continue
		}

		result = append(result, part)
	}

	return result
}

func SplitURL(url string) []string {

	parts := urlSplitRegex.Split(url, -1)

	result := []string{}

	for _, part := range parts {

		part = strings.ToLower(part)

		if part == "" {
			continue
		}

		if _, exists := PopularDomains[part]; exists {
			continue
		}

		result = append(result, part)
	}

	return result
}

func getMetaContent(doc *goquery.Document, propertyValue string, nameValue string) string {

	if propertyValue != "" {

		meta := doc.Find(`meta[property="` + propertyValue + `"]`)

		if content, exists := meta.Attr("content"); exists {
			return content
		}
	}

	if nameValue != "" {

		meta := doc.Find(`meta[name="` + nameValue + `"]`)

		if content, exists := meta.Attr("content"); exists {
			return content
		}
	}

	return ""
}

func tokenize(text string) []string {

	words := strings.Fields(text)

	filtered := []string{}

	for _, word := range words {

		word = strings.ToLower(word)

		if _, stop := StopWords[word]; stop {
			continue
		}

		if !isAlphaNumeric(word) {
			continue
		}

		filtered = append(filtered, word)
	}

	return filtered
}

func isAlphaNumeric(word string) bool {

	for _, r := range word {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') {
			return false
		}
	}

	return true
}

func GetHTMLData(html string) (*HTMLData, error) {

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	title := getMetaContent(doc, "og:title", "title")
	description := getMetaContent(doc, "og:description", "description")

	var pageText strings.Builder

	doc.Find("p").Each(func(i int, s *goquery.Selection) {

		text := s.Text()

		text = bracketRegex.ReplaceAllString(text, "")

		pageText.WriteString(strings.TrimSpace(text))
		pageText.WriteString(" ")
	})

	fullText := pageText.String()

	words := strings.Fields(fullText)

	summaryWords := words

	if len(words) > 500 {
		summaryWords = words[:500]
	}

	summary := strings.Join(summaryWords, " ")

	filtered := tokenize(fullText)

	info := whatlanggo.Detect(summary)

	return &HTMLData{
		Title:       title,
		Description: description,
		SummaryText: summary,
		Text:        filtered,
		Language:    info.Lang.Iso6391(),
	}, nil
}

