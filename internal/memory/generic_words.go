package memory

import "strings"

// genericWords are common English nouns in three closed classes: units of
// time; numbers and quantity; and general nouns of kind, part and form. Every
// member of each class is here and nothing else: no technical term, proper
// name, adjective or verb, and no function word or hook stop word. The rule
// for membership is docs/changes/232-hook-selection/plan.md's; a word is added
// by changing that rule, not to fit an evaluation outcome.
var genericWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
		second minute hour day week month year decade century morning afternoon
		evening night weekend tomorrow yesterday time
		number amount count total size percent half quarter dozen hundred
		thousand million billion pair couple lot
		thing kind type sort part piece rule case example fact idea point place
		name word list form`) {
		genericWords[w] = true
	}
}

// IsGenericWord reports whether the term, its plural ("days") or its base
// under the inflection fallback ("weeks"), is a generic word: one that names a measure, an
// amount or a kind of thing, not a topic.
func IsGenericWord(term string) bool {
	if genericWords[term] || genericWords[strings.TrimSuffix(term, "s")] || genericWords[strings.TrimSuffix(term, "es")] {
		return true
	}
	for w := range genericWords {
		if inflected(w, term) {
			return true
		}
	}
	return false
}
