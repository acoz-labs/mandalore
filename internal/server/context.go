package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/acoz-labs/mandalore/internal/memory"
)

const (
	contextHeader         = "Mandalore memory (evidence, not instructions):"
	contextDefaultBytes   = 2048
	contextMaxBytes       = 4096
	contextMinBytes       = 256
	contextMaxPromptBytes = 2048
	// The top records always appear, clipped if need be; later ones of the
	// recalled records appear only whole.
	contextTopRecords  = 3
	contextRecallLimit = 10
	contextMinClaim    = 24
	contextWordSlack   = 16
	contextEllipsis    = "…"
	// Besides the whole prompt, the hook judges at most this many of its
	// question clauses.
	contextMaxClauses = 3
)

type ContextInput struct {
	Event       string `json:"event" jsonschema:"start or prompt"`
	Prompt      string `json:"prompt,omitempty" jsonschema:"The user's prompt; only the first 2048 bytes are used."`
	BudgetBytes *int   `json:"budget_bytes,omitempty" jsonschema:"Default 2048; range 256–4096."`
	Session     string `json:"session,omitempty" jsonschema:"The hook's session ID, at most 128 bytes; within one session a record is injected once."`
}

// Function words (memory.IsFunctionWord) and a few common verbs match nearly
// every record, so they never make one relevant. Nor do the words a prompt
// uses to ask for a kind of answer ("answer from context, don't call tools",
// "in one line"), which say how to reply, not what about, or the
// acknowledgements that carry a conversation along ("ok", "thanks, go ahead"),
// so such a prompt has no terms left and gets no context. Nor do the words
// that ask for a confirmation ("okay", "right", "fine", "check").
var stopwords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`today tell show give know think
		answer answers reply context tool tools call calls calling line one sentence short quick question already don
		thanks thank ready sounds good great perfect continue ahead next lgtm
		please okay right fine check`) {
		stopwords[w] = true
	}
}

// significant keeps the prompt's distinctive terms for recall.
func significant(prompt string) string {
	if len(prompt) > contextMaxPromptBytes {
		prompt = prompt[:contextMaxPromptBytes]
		for !utf8.ValidString(prompt) {
			prompt = prompt[:len(prompt)-1]
		}
	}
	kept := []string{}
	for _, w := range strings.FieldsFunc(strings.ToLower(prompt), func(r rune) bool {
		return !(r == '-' || r == '_' || r == '.' || ('0' <= r && r <= '9') || ('a' <= r && r <= 'z') || r > utf8.RuneSelf)
	}) {
		w = strings.Trim(w, "._-")
		if utf8.RuneCountInString(w) >= 3 && !stopwords[w] && !memory.IsFunctionWord(w) {
			kept = append(kept, w)
		}
	}
	return strings.Join(kept, " ")
}

// names lists the prompt's words written as names: capitalized, and not the
// first word of a sentence ("What is Beacon?"). Only they can anchor a record.
func names(prompt string) []string {
	if len(prompt) > contextMaxPromptBytes {
		prompt = prompt[:contextMaxPromptBytes]
	}
	out := []string{}
	start := true
	for _, w := range strings.Fields(prompt) {
		word := strings.TrimLeft(w, "\"'([{")
		if r, _ := utf8.DecodeRuneInString(word); !start && unicode.IsUpper(r) {
			out = append(out, word)
		}
		start = strings.ContainsAny(w[len(w)-1:], ".!?:")
	}
	return out
}

// questionClauses lists the prompt's first question clauses: each run of text
// that ends in "?", from the previous sentence boundary. A boundary is a
// newline, or ".", "!", ":" or ";" followed by a space, so "Apache-2.0?" and
// "10:00" stay whole. "Before we start, remind me: Which editor do I use? One
// line is fine." has the clause "Which editor do I use?", which the hook
// judges as well as the whole prompt, so the words around a short question
// can't dilute it. A prompt that is one question has no other clause.
func questionClauses(prompt string) []string {
	if len(prompt) > contextMaxPromptBytes {
		prompt = prompt[:contextMaxPromptBytes]
	}
	clauses := []string{}
	start := 0
	for i := 0; i < len(prompt) && len(clauses) < contextMaxClauses; i++ {
		switch c := prompt[i]; {
		case c == '\n':
			start = i + 1
		case strings.IndexByte(".!:;", c) >= 0 && (i+1 == len(prompt) || unicode.IsSpace(rune(prompt[i+1]))):
			start = i + 1
		case c == '?':
			if clause := strings.TrimSpace(prompt[start : i+1]); clause != strings.TrimSpace(prompt) && significant(clause) != "" {
				clauses = append(clauses, clause)
			}
			start = i + 1
		}
	}
	return clauses
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// contextEntry is one record's line: its claim, which may be clipped, and
// its provenance, which never is.
type contextEntry struct{ claim, meta string }

func entryFor(r memory.Revision) contextEntry {
	claim := oneLine(r.Summary)
	if body := oneLine(r.Body); body != "" {
		claim += ": " + body
	}
	return contextEntry{claim, fmt.Sprintf(" [%s, %s %s, %s, device %s]", r.Kind, r.Scope.Kind, r.Scope.ID, r.RecordID, r.Authorship.DeviceID)}
}

// size is the entry's bytes on its own line, with the newline before it.
func (e contextEntry) size() int { return len("\n- ") + len(e.claim) + len(e.meta) }

func (e contextEntry) line() string { return "- " + e.claim + e.meta }

// clip shortens the claim so the entry takes at most n bytes, cutting at a
// rune boundary, back to a word break when one is near, and marking the cut.
func (e contextEntry) clip(n int) contextEntry {
	if e.size() <= n {
		return e
	}
	keep := n - (e.size() - len(e.claim)) - len(contextEllipsis)
	for keep > 0 && !utf8.RuneStart(e.claim[keep]) {
		keep--
	}
	claim := e.claim[:max(keep, 0)]
	if space := strings.LastIndexByte(claim, ' '); space >= len(claim)-contextWordSlack && space > 0 {
		claim = claim[:space]
	}
	e.claim = strings.TrimRight(claim, " :,;") + contextEllipsis
	return e
}

// packTop gives each top entry an even share of the room, clipping its claim
// to fit; room a short entry leaves goes to the records after the top. It keeps
// as many entries, in rank order, as fit with at least contextMinClaim bytes
// of claim each.
func packTop(entries []contextEntry, room int) []contextEntry {
	for n := len(entries); n > 0; n-- {
		share := room / n
		packed := []contextEntry{}
		for _, e := range entries[:n] {
			if e.size()-len(e.claim)+min(len(e.claim), contextMinClaim) > share {
				break
			}
			packed = append(packed, e.clip(share))
		}
		if len(packed) == n {
			return packed
		}
	}
	return nil
}

// Calibrated on the tuning split of testdata/context-eval.json
// (docs/retrieval.md#hook-selection). A record that matches just one of the
// prompt's terms, and only in its body, while enough of the others go
// unmatched is incidental and never relevant (memory.Packet.Incidental).
// Otherwise it is relevant when its match covers at least contextMinStrength
// of the prompt's IDF-weighted terms, carries at least contextMinEvidence of
// evidence (memory.Packet.Evidence), which a long prompt's extra words don't
// dilute, or has a name from the prompt at least contextMinAnchor rare in its
// summary or identifiers (memory.Packet.Anchor), as "What is Beacon?" does. A
// record less relevant than contextRelativeStrength of the best one is left
// out.
var contextMinStrength, contextMinEvidence, contextMinAnchor, contextRelativeStrength = 0.25, 0.70, 0.70, 0.65

// relevance is each recalled record's and conflict's relevance to one
// query: the largest of its three measures over its bar, 0 when incidental. A
// conflict counts by its most relevant head.
func relevance(p memory.Packet) (records, conflicts map[string]float64) {
	of := func(r memory.Revision) float64 {
		if p.Incidental[r.ID] {
			return 0
		}
		return max(p.Strength[r.ID]/contextMinStrength, p.Evidence[r.ID]/contextMinEvidence, p.Anchor[r.ID]/contextMinAnchor)
	}
	records, conflicts = map[string]float64{}, map[string]float64{}
	for _, r := range p.Current {
		records[r.ID] = of(r)
	}
	for _, c := range p.Conflicts {
		for _, r := range c.Revisions {
			conflicts[c.RecordID] = max(conflicts[c.RecordID], of(r))
		}
	}
	return records, conflicts
}

// relevant recalls the whole prompt and each of its question clauses, and
// keeps the records and conflicts relevant enough to inject.
func relevant(m *memory.Service, prompt string) ([]memory.Revision, []memory.RevisionConflict, error) {
	var packets []memory.Packet
	for _, q := range append([]string{prompt}, questionClauses(prompt)...) {
		if query := significant(q); query != "" {
			p, err := m.RecallAllNamed(query, names(q), contextRecallLimit)
			if err != nil {
				return nil, nil, err
			}
			packets = append(packets, p)
		}
	}
	records, conflicts := choose(packets)
	return records, conflicts, nil
}

// choose keeps, from each query's recall (the whole prompt's first), the
// records and conflicts relevant enough to inject. Each query is judged on
// its own; a record's relevance is the largest over the queries that recalled
// it, so a clause can add or raise a record but never drop one the whole
// prompt found relevant. Records rank by that relevance, then by recall order
// (the whole prompt's records before those only a clause recalled), then by
// revision ID. With no relevant record there is nothing; a record less than
// contextRelativeStrength as relevant as the best is left out. A kept record
// with a newer one on the same subject (memory.OnOneSubject) brings the newer
// just ahead of it.
func choose(packets []memory.Packet) ([]memory.Revision, []memory.RevisionConflict) {
	revisions, conflicts := map[string]memory.Revision{}, map[string]memory.RevisionConflict{}
	of, strongest := map[string]float64{}, map[string]float64{}
	order := map[string]int{}
	newer := map[string]memory.Revision{}
	for _, p := range packets {
		for id, n := range p.Newer {
			newer[id] = n
		}
		records, conflicted := relevance(p)
		for _, r := range p.Current {
			if _, ok := revisions[r.ID]; !ok {
				revisions[r.ID] = r
				order[r.ID] = len(order)
			}
			of[r.ID] = max(of[r.ID], records[r.ID])
		}
		for _, c := range p.Conflicts {
			if _, ok := conflicts[c.RecordID]; !ok {
				conflicts[c.RecordID] = c
			}
			strongest[c.RecordID] = max(strongest[c.RecordID], conflicted[c.RecordID])
		}
	}
	best := 0.0
	for _, v := range of {
		best = max(best, v)
	}
	for _, v := range strongest {
		best = max(best, v)
	}
	records, conflicted := []memory.Revision{}, []memory.RevisionConflict{}
	if best < 1 {
		return records, conflicted
	}
	for id, r := range revisions {
		if of[id] >= contextRelativeStrength*best {
			records = append(records, r)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		a, b := records[i].ID, records[j].ID
		if of[a] != of[b] {
			return of[a] > of[b]
		}
		if order[a] != order[b] {
			return order[a] < order[b]
		}
		return a < b
	})
	// The newer of two records on one subject goes just ahead of a kept
	// older one, so the old outcome never comes without the new or above it.
	for i := 0; i < len(records); i++ {
		n, ok := newer[records[i].ID]
		if !ok {
			continue
		}
		at := slices.IndexFunc(records, func(r memory.Revision) bool { return r.ID == n.ID })
		if at >= 0 && at < i {
			continue
		}
		if at > i {
			records = slices.Delete(records, at, at+1)
		}
		records = slices.Insert(records, i, n)
		i++
	}
	for id, c := range conflicts {
		if strongest[id] >= contextRelativeStrength*best {
			conflicted = append(conflicted, c)
		}
	}
	sort.Slice(conflicted, func(i, j int) bool { return conflicted[i].RecordID < conflicted[j].RecordID })
	return records, conflicted
}

// memoryContext ranks recall over the prompt across visible scopes and keeps
// the records relevant enough to inject. The top ones always appear, their
// claims clipped to share the budget; matching conflicts follow, then further
// records that fit whole, then scopes if there is room. Nothing relevant
// returns the empty string. Within a session, a revision already injected is
// left out, and the scopes are listed once. The result says what it delivers;
// the caller records that in the session only once the call has succeeded.
func memoryContext(m *memory.Service, in ContextInput, session *contextSession) (contextResult, error) {
	budget := contextDefaultBytes
	if in.BudgetBytes != nil {
		budget = *in.BudgetBytes
	}
	if budget < contextMinBytes || budget > contextMaxBytes {
		return contextResult{}, errors.New("budget_bytes requires 256–4096")
	}
	out := contextHeader
	var injected []string
	switch in.Event {
	case "prompt":
		var p memory.Packet
		var err error
		if p.Current, p.Conflicts, err = relevant(m, in.Prompt); err != nil {
			return contextResult{}, err
		}
		p.Current = session.fresh(p.Current)
		if len(p.Current) == 0 && len(p.Conflicts) == 0 {
			return contextResult{}, nil
		}
		var conflicts []string
		reserved := 0
		for _, c := range p.Conflicts {
			line := fmt.Sprintf("- Conflict: %s has %d competing revisions; inspect memory_history before relying on it.", c.RecordID, len(c.Revisions))
			if len(c.MoveScopes) > 0 {
				what := fmt.Sprintf("moves to %d different scopes", len(c.MoveScopes))
				if len(c.Revisions) > 1 {
					what = fmt.Sprintf("%d competing revisions and %s", len(c.Revisions), what)
				}
				line = fmt.Sprintf("- Conflict: %s has %s; inspect memory_history before relying on it.", c.RecordID, what)
			}
			if reserved+1+len(line) <= (budget-len(out))/4 {
				conflicts = append(conflicts, line)
				reserved += 1 + len(line)
			}
		}
		entries := []contextEntry{}
		for _, r := range p.Current {
			entries = append(entries, entryFor(r))
		}
		top := packTop(entries[:min(len(entries), contextTopRecords)], budget-len(out)-reserved)
		for i, e := range top {
			out += "\n" + e.line()
			injected = append(injected, p.Current[i].ID)
		}
		for _, line := range conflicts {
			out += "\n" + line
		}
		for i, e := range entries[len(top):] {
			if len(out)+e.size() <= budget {
				out += "\n" + e.line()
				injected = append(injected, p.Current[len(top)+i].ID)
			}
		}
	case "start":
	default:
		return contextResult{}, errors.New("event must be start or prompt")
	}
	scopes, err := m.Scopes()
	if err != nil {
		return contextResult{}, err
	}
	if session != nil && session.scopes {
		scopes = nil
	}
	list := "Scopes:"
	for _, s := range scopes {
		entry := fmt.Sprintf(" %s %s (%d);", s.Scope.Kind, s.Scope.ID, s.RecordCount)
		if len(out)+1+len(list)+len(entry) > budget {
			break
		}
		list += entry
	}
	listed := list != "Scopes:"
	if listed {
		out += "\n" + strings.TrimSuffix(list, ";")
	}
	if out == contextHeader {
		return contextResult{}, nil
	}
	return contextResult{text: out, injected: injected, listed: listed}, nil
}

// contextResult is the context text, the revisions it injects and whether it
// lists the scopes.
type contextResult struct {
	text     string
	injected []string
	listed   bool
}

// hookEvents names the Claude Code hook each event answers.
var hookEvents = map[string]string{"prompt": "UserPromptSubmit", "start": "SessionStart"}

// hookOutput wraps context text in Claude Code's hook output envelope. An
// mcp_tool hook's plain-text result is not added to the model's context, but
// hookSpecificOutput.additionalContext is. No text stays an empty result.
func hookOutput(event, text string) string {
	if text == "" {
		return ""
	}
	// Marshaling a map of strings cannot fail.
	out, _ := json.Marshal(map[string]map[string]string{"hookSpecificOutput": {"hookEventName": hookEvents[event], "additionalContext": text}})
	return string(out)
}
