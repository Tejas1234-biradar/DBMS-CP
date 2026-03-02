// utils/nlp.go
package utils

import (
	"strings"
	"unicode"
)

var StopWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "and": {}, "or": {}, "but": {},
	"in": {}, "on": {}, "at": {}, "to": {}, "for": {}, "of": {},
	"with": {}, "by": {}, "from": {}, "is": {}, "are": {}, "was": {},
	"were": {}, "be": {}, "been": {}, "being": {}, "have": {}, "has": {},
	"had": {}, "do": {}, "does": {}, "did": {}, "will": {}, "would": {},
	"could": {}, "should": {}, "may": {}, "might": {}, "this": {}, "that": {},
	"these": {}, "those": {}, "it": {}, "its": {}, "as": {}, "if": {},
	"then": {}, "than": {}, "so": {}, "not": {}, "no": {}, "up": {},
	"out": {}, "about": {}, "into": {}, "through": {}, "what": {}, "which": {},
	"who": {}, "how": {}, "all": {}, "each": {}, "more": {}, "also": {},
	"can": {}, "just": {}, "they": {}, "their": {}, "there": {}, "when": {},
	"we": {}, "you": {}, "he": {}, "she": {}, "his": {}, "her": {},
}

func Tokenize(text string) []string {
	// lowercase and split on non-alphanumeric
	var tokens []string
	word := strings.Builder{}

	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(r)
		} else if word.Len() > 0 {
			token := word.String()
			if _, isStop := StopWords[token]; !isStop && len(token) > 1 {
				tokens = append(tokens, token)
			}
			word.Reset()
		}
	}
	if word.Len() > 0 {
		token := word.String()
		if _, isStop := StopWords[token]; !isStop && len(token) > 1 {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// term-Frequency:= how many tiems a word appears in a doc. First half of the search ranking algo TF-IDF
func ComputeTF(tokens []string) map[string]int {
	freq := make(map[string]int)
	for _, token := range tokens {
		freq[token]++
	}
	return freq
}
