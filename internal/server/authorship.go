package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	maxMCPBodyBytes       = 65536
	maxHarnessBytes       = 128
	maxHarnessSuffixBytes = 96
	maxSessionIDBytes     = 128
	sessionIDHeader       = "Mcp-Session-Id"
	protocolVersionHeader = "Mcp-Protocol-Version"
)

// sessionProtocols are the MCP versions with a transport session: the
// initialize handshake and Mcp-Session-Id. Sessionless 2026-07-28 and later
// have neither, so a stray header there records nothing.
var sessionProtocols = map[string]bool{"2024-11-05": true, "2025-03-26": true, "2025-06-18": true, "2025-11-25": true}

// harness is the harness recorded on an OAuth client's writes: its
// registered name and the host that vouches for it, which the client can't
// choose. Both are client-supplied, so both are filtered again here.
func (c *oauthClient) harness() string {
	how := "redirect to"
	if c.Document {
		how = "document on"
	}
	host := printable(c.Kind)
	suffix := " (" + how + " " + keepRight(host, maxHarnessSuffixBytes-len(how)-len(" ( )")) + ")"
	return strings.TrimSpace(keepLeft(displayName(c.Name, host), maxHarnessBytes-len(suffix))) + suffix
}

// keepLeft cuts s to at most n bytes, at a rune boundary.
func keepLeft(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// keepRight keeps at most n bytes of s from its end, behind "…", at a rune
// boundary, so a long host keeps its registrable domain.
func keepRight(s string, n int) string {
	if len(s) <= n {
		return s
	}
	i := len(s) - (n - len("…"))
	for i < len(s) && !utf8.RuneStart(s[i]) {
		i++
	}
	return "…" + s[i:]
}

// issueSession gives an initialize request a transport session ID. The
// server stays stateless: it keeps no table and never refuses an ID, so the
// ID only labels the connection's writes. The body is restored for the SDK,
// which applies its own size limit.
func issueSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.Header.Get(sessionIDHeader) != "" {
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxMCPBodyBytes+1))
	r.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(raw), r.Body), r.Body}
	var msg struct {
		Method string `json:"method"`
	}
	if err == nil && json.Unmarshal(raw, &msg) == nil && msg.Method == "initialize" {
		w.Header().Set(sessionIDHeader, rand.Text())
	}
}

type sessionKey struct{}

func withSession(ctx context.Context, h http.Header) context.Context {
	return context.WithValue(ctx, sessionKey{}, transportSession(h))
}

func sessionFrom(ctx context.Context) *string {
	s, _ := ctx.Value(sessionKey{}).(*string)
	return s
}

// transportSession is the request's transport session ID, or nil when the
// protocol has no sessions or the ID isn't one value of 1–128 visible ASCII
// characters. No protocol header means 2025-03-26, as the specification says.
func transportSession(h http.Header) *string {
	if v := h.Values(protocolVersionHeader); len(v) > 1 || len(v) == 1 && !sessionProtocols[v[0]] {
		return nil
	}
	ids := h.Values(sessionIDHeader)
	if len(ids) != 1 || ids[0] == "" || len(ids[0]) > maxSessionIDBytes || strings.ContainsFunc(ids[0], func(r rune) bool { return r < 0x21 || r > 0x7e }) {
		return nil
	}
	return &ids[0]
}
