package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	sessionCookie = "__Host-mandalore-owner"
	sessionTTL    = 12 * time.Hour
	ceremonyTTL   = 5 * time.Minute
)

type ownerSession struct {
	csrf    string
	expires time.Time
}

// ceremony is a passkey registration or login in progress. A registration
// remembers which enrollment link started it.
type ceremony struct {
	data       webauthn.SessionData
	enrollment string
	expires    time.Time
}

// ownerUser is the single owner as a WebAuthn user.
type ownerUser struct {
	state ownerState
	name  string
}

func (u ownerUser) WebAuthnID() []byte                         { return u.state.UserID }
func (u ownerUser) WebAuthnName() string                       { return "owner" }
func (u ownerUser) WebAuthnDisplayName() string                { return u.name }
func (u ownerUser) WebAuthnCredentials() []webauthn.Credential { return u.state.Credentials }

// EnrollOwner creates the one-time owner enrollment link, valid for ten
// minutes. It replaces any earlier unused link. The new passkey is added, or
// with replace it becomes the only one. Shell access to the server is the
// root of trust, so this is also how a lost passkey is recovered.
func EnrollOwner(c Config, replace bool) (string, time.Time, error) {
	if err := c.Validate(); err != nil {
		return "", time.Time{}, err
	}
	if c.auth() != AuthOAuth {
		return "", time.Time{}, errors.New("owner passkeys exist only with auth oauth")
	}
	store, err := openOAuthStore(c.StateDir, time.Now)
	if err != nil {
		return "", time.Time{}, err
	}
	token := secret()
	expires := time.Now().Add(enrollmentTTL)
	err = store.update(func(st *oauthState) error {
		if len(st.Owner.UserID) == 0 {
			st.Owner.UserID = make([]byte, 32)
			_, _ = rand.Read(st.Owner.UserID)
		}
		st.Owner.Enrollment, st.Owner.EnrollmentExpires, st.Owner.EnrollmentReplaces = hashSecret(token), expires, replace
		return nil
	})
	// The token travels in the fragment, which browsers never send to the
	// server or a proxy in front of it.
	return c.origin() + "/owner/enroll#" + token, expires, err
}

func (a *authServer) user() (ownerUser, error) {
	var u ownerUser
	err := a.store.view(func(st *oauthState) error {
		u = ownerUser{state: st.Owner, name: a.owner}
		u.state.Credentials = append([]webauthn.Credential(nil), st.Owner.Credentials...)
		return nil
	})
	return u, err
}

// session returns the owner session named by the request's cookie, if any.
func (a *authServer) session(r *http.Request) *ownerSession {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.sessions[hashSecret(cookie.Value)]
	if s == nil || !a.now().Before(s.expires) {
		return nil
	}
	return s
}

func (a *authServer) startSession(w http.ResponseWriter) {
	value := secret()
	a.mu.Lock()
	a.prune()
	makeRoom(a.sessions, func(s *ownerSession) time.Time { return s.expires })
	a.sessions[hashSecret(value)] = &ownerSession{csrf: secret(), expires: a.now().Add(sessionTTL)}
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: value, Path: "/", MaxAge: int(sessionTTL.Seconds()), Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func (a *authServer) startCeremony(c *ceremony) string {
	id := secret()
	c.expires = a.now().Add(ceremonyTTL)
	a.mu.Lock()
	a.prune()
	makeRoom(a.ceremonies, func(c *ceremony) time.Time { return c.expires })
	a.ceremonies[id] = c
	a.mu.Unlock()
	return id
}

func (a *authServer) takeCeremony(r *http.Request) *ceremony {
	id := r.URL.Query().Get("ceremony")
	a.mu.Lock()
	defer a.mu.Unlock()
	c := a.ceremonies[id]
	delete(a.ceremonies, id)
	if c == nil || !a.now().Before(c.expires) {
		return nil
	}
	return c
}

// ownerPost admits a passkey ceremony request. The handler has already
// refused a foreign Origin.
func ownerPost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	return true
}

func (a *authServer) registerBegin(w http.ResponseWriter, r *http.Request) {
	if !ownerPost(w, r) {
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "malformed request")
		return
	}
	u, err := a.user()
	if err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	hash := hashSecret(in.Token)
	if u.state.Enrollment == "" || subtle.ConstantTimeCompare([]byte(hash), []byte(u.state.Enrollment)) != 1 || !a.now().Before(u.state.EnrollmentExpires) {
		oauthError(w, http.StatusForbidden, "access_denied", "this enrollment link is invalid, used or expired; run mandalore server owner enroll again")
		return
	}
	exclude := []protocol.CredentialDescriptor{}
	for _, c := range u.state.Credentials {
		exclude = append(exclude, c.Descriptor())
	}
	options, data, err := a.webauthn.BeginRegistration(u, webauthn.WithExclusions(exclude),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, UserVerification: protocol.VerificationRequired}))
	if err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", "cannot start passkey registration")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ceremony": a.startCeremony(&ceremony{data: *data, enrollment: hash}), "options": options})
}

func (a *authServer) registerFinish(w http.ResponseWriter, r *http.Request) {
	if !ownerPost(w, r) {
		return
	}
	c := a.takeCeremony(r)
	u, err := a.user()
	if c == nil || c.enrollment == "" || err != nil {
		oauthError(w, http.StatusForbidden, "access_denied", "passkey registration expired; open the enrollment link again")
		return
	}
	credential, err := a.webauthn.FinishRegistration(u, c.data, r)
	if err != nil {
		oauthError(w, http.StatusBadRequest, "access_denied", "the passkey could not be verified")
		return
	}
	err = a.store.update(func(st *oauthState) error {
		// The link is single-use: whichever ceremony finishes first consumes it.
		if st.Owner.Enrollment != c.enrollment || !a.now().Before(st.Owner.EnrollmentExpires) {
			return errors.New("this enrollment link was already used or expired")
		}
		if st.Owner.EnrollmentReplaces {
			st.Owner.Credentials = nil
			// Sessions started with a replaced passkey end too.
			a.mu.Lock()
			clear(a.sessions)
			a.mu.Unlock()
		}
		st.Owner.Enrollment, st.Owner.EnrollmentExpires, st.Owner.EnrollmentReplaces = "", time.Time{}, false
		st.Owner.Credentials = append(st.Owner.Credentials, *credential)
		return nil
	})
	if err != nil {
		oauthError(w, http.StatusForbidden, "access_denied", err.Error())
		return
	}
	a.startSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *authServer) loginBegin(w http.ResponseWriter, r *http.Request) {
	if !ownerPost(w, r) {
		return
	}
	u, err := a.user()
	if err != nil || len(u.state.Credentials) == 0 {
		oauthError(w, http.StatusForbidden, "access_denied", "no owner passkey is enrolled; run mandalore server owner enroll on the server")
		return
	}
	options, data, err := a.webauthn.BeginLogin(u, webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		oauthError(w, http.StatusInternalServerError, "server_error", "cannot start passkey sign-in")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ceremony": a.startCeremony(&ceremony{data: *data}), "options": options})
}

func (a *authServer) loginFinish(w http.ResponseWriter, r *http.Request) {
	if !ownerPost(w, r) {
		return
	}
	c := a.takeCeremony(r)
	u, err := a.user()
	if c == nil || c.enrollment != "" || err != nil {
		oauthError(w, http.StatusForbidden, "access_denied", "passkey sign-in expired; try again")
		return
	}
	credential, err := a.webauthn.FinishLogin(u, c.data, r)
	if err != nil {
		oauthError(w, http.StatusForbidden, "access_denied", "the passkey could not be verified")
		return
	}
	_ = a.store.update(func(st *oauthState) error {
		for i := range st.Owner.Credentials {
			if string(st.Owner.Credentials[i].ID) == string(credential.ID) {
				st.Owner.Credentials[i].Authenticator = credential.Authenticator
			}
		}
		return nil
	})
	a.startSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *authServer) enrollPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	a.page(w, http.StatusOK, enrollPage{})
}

func (a *authServer) loginPage(w http.ResponseWriter, client *oauthClient) {
	a.page(w, http.StatusOK, loginPage{Client: client.Name})
}

// consentPage asks the owner to approve an authorization request. another
// says other installations of the same document client are approved.
func (a *authServer) consentPage(w http.ResponseWriter, req *authRequest, id, csrf string, another bool) {
	u, _ := url.Parse(req.redirect)
	write := req.requested != ScopeRead
	// A document client names itself; the host that publishes the document
	// is what vouches for that name.
	publisher := ""
	if req.client.Document {
		publisher = req.client.Kind
	}
	a.page(w, http.StatusOK, consentPage{Client: req.client.Name, Publisher: publisher, Host: u.Host, Loopback: loopbackRedirect(u), Signet: req.signet,
		Another: another, Write: write, Request: id, CSRF: csrf})
}

type (
	errorPage   struct{ Message string }
	enrollPage  struct{}
	loginPage   struct{ Client string }
	consentPage struct {
		Client, Publisher, Host, Signet, Request, CSRF string
		Another, Loopback, Write                       bool
	}
)

// page renders an owner-facing HTML page with a strict content security
// policy. Pages may not be framed, so consent can't be clickjacked.
func (a *authServer) page(w http.ResponseWriter, status int, data any) {
	nonce := secret()
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+nonce+"'; style-src 'nonce-"+nonce+"'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "same-origin")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	var name string
	switch data.(type) {
	case errorPage:
		name = "error"
	case enrollPage:
		name = "enroll"
	case loginPage:
		name = "login"
	case consentPage:
		name = "consent"
	}
	var body strings.Builder
	if err := pages.ExecuteTemplate(&body, name, data); err != nil {
		return
	}
	_ = pages.ExecuteTemplate(w, "layout", map[string]any{"Nonce": nonce, "Body": template.HTML(body.String())})
}

var pages = template.Must(template.New("").Parse(`
{{define "layout"}}<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Mandalore</title>
<style nonce="{{.Nonce}}">
body{font:16px/1.5 system-ui,sans-serif;max-width:32rem;margin:2rem auto;padding:0 1rem;color:#1d1d1f;background:#fff}
@media (prefers-color-scheme:dark){body{color:#f2f2f2;background:#161616}}
h1{font-size:1.4rem}.host{font-size:2rem;font-weight:700;word-break:break-all;margin:.5rem 0}
.warn{border-left:4px solid #c62;padding:.5rem 1rem;background:rgba(204,102,34,.12)}
button{font:inherit;padding:.6rem 1.2rem;margin:.5rem .5rem 0 0;border-radius:.4rem;border:1px solid #888;cursor:pointer}
button.primary{background:#1d5bd8;color:#fff;border-color:#1d5bd8}label{display:block;margin:.3rem 0}
</style></head><body>{{.Body}}
<script nonce="{{.Nonce}}">
const b64 = {
  dec: s => Uint8Array.from(atob(s.replace(/-/g, "+").replace(/_/g, "/")), c => c.charCodeAt(0)),
  enc: b => btoa(String.fromCharCode(...new Uint8Array(b))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, ""),
};
async function post(path, body) {
  const r = await fetch(path, {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(body)});
  const v = await r.json();
  if (!r.ok) throw new Error(v.error_description || "request failed");
  return v;
}
function credentialJSON(c) {
  const out = {id: c.id, rawId: b64.enc(c.rawId), type: c.type, response: {clientDataJSON: b64.enc(c.response.clientDataJSON)}};
  for (const k of ["attestationObject", "authenticatorData", "signature", "userHandle"]) {
    if (c.response[k]) out.response[k] = b64.enc(c.response[k]);
  }
  return out;
}
const say = m => { document.getElementById("status").textContent = m; };
async function enroll() {
  const begin = await post("/owner/register/begin", {token: location.hash.slice(1)});
  const o = begin.options.publicKey;
  o.challenge = b64.dec(o.challenge); o.user.id = b64.dec(o.user.id);
  (o.excludeCredentials || []).forEach(c => { c.id = b64.dec(c.id); });
  const c = await navigator.credentials.create({publicKey: o});
  await post("/owner/register/finish?ceremony=" + encodeURIComponent(begin.ceremony), credentialJSON(c));
  history.replaceState(null, "", location.pathname);
  say("Passkey saved. You can close this page.");
}
async function login() {
  const begin = await post("/owner/login/begin", {});
  const o = begin.options.publicKey;
  o.challenge = b64.dec(o.challenge);
  (o.allowCredentials || []).forEach(c => { c.id = b64.dec(c.id); });
  const c = await navigator.credentials.get({publicKey: o});
  await post("/owner/login/finish?ceremony=" + encodeURIComponent(begin.ceremony), credentialJSON(c));
  location.reload();
}
for (const [id, fn] of [["enroll", enroll], ["login", login]]) {
  const b = document.getElementById(id);
  if (b) b.addEventListener("click", () => fn().catch(e => say(e.message)));
}
</script></body></html>{{end}}

{{define "error"}}<h1>Mandalore</h1><p>{{.Message}}</p>{{end}}

{{define "enroll"}}<h1>Register the owner passkey</h1>
<p>This one-time link adds a passkey that approves applications for this Mandalore server.</p>
<button id="enroll" class="primary">Create passkey</button><p id="status" role="status"></p>{{end}}

{{define "login"}}<h1>Owner sign-in</h1>
<p><strong>{{.Client}}</strong> wants to connect to your memory. Sign in with your owner passkey to review it.</p>
<button id="login" class="primary">Sign in with passkey</button><p id="status" role="status"></p>{{end}}

{{define "consent"}}<h1>Connect {{.Client}}?</h1>
{{if .Publisher}}<p>Published by <strong>{{.Publisher}}</strong>. The application chooses its own name; trust it only if you know this publisher.</p>{{end}}
{{if .Another}}<p>Other installations of this application are connected already. This is a new installation: approving it doesn't change their access, and they don't share this one.</p>{{end}}
<p>It will reach the <strong>{{.Signet}}</strong> signet and send its access to</p>
<p class="host">{{.Host}}</p>
{{if .Loopback}}<p class="warn">This address is on the device that started the connection. Any program on that device can claim it, so approve only if you started this connection yourself just now.</p>{{end}}
<form method="post" action="/consent">
<input type="hidden" name="request" value="{{.Request}}"><input type="hidden" name="csrf" value="{{.CSRF}}">
<fieldset><legend>Access</legend>
<label><input type="radio" name="scope" value="read"{{if not .Write}} checked{{end}}> Read memory</label>
<label><input type="radio" name="scope" value="read-write"{{if .Write}} checked{{end}}> Read and save memory</label>
</fieldset>
<button name="decision" value="approve" class="primary">Approve</button><button name="decision" value="deny">Deny</button>
</form>{{end}}
`))
