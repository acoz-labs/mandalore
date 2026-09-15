# Mandalore results in code mode

When a code-mode call exposes the raw MCP wrapper, present one full envelope,
not both compatibility copies and not only `.result`. Native clients that already
select a representation need no extra wrapping. Never call the tool again just
to render it differently, especially after a write or ambiguous failure.

This conservative example selects structured content only when the single text
block matches its serialization and error state. Otherwise it preserves the
original response. It does not parse text to establish equality: parsing can
hide duplicate keys or round historical JSON numbers. Different object order or
number formatting may safely leave a duplicate rather than risk losing evidence.

```javascript
function mandaloreOutput(r) {
  const object = x => x !== null && typeof x === "object" && !Array.isArray(x);
  const only = (x, names) => object(x) && Object.keys(x).every(k => names.includes(k));
  const own = (x, k) => Object.prototype.hasOwnProperty.call(x, k);
  const s = r?.structuredContent;
  if (!only(r, ["content", "structuredContent", "isError"]) ||
      !only(s, ["protocol_version", "ok", "result", "error"]) ||
      s.protocol_version !== 1 || typeof s.ok !== "boolean" ||
      (s.ok ? (!own(s, "result") || own(s, "error")) :
              (!object(s.error) || own(s, "result"))) ||
      (r.isError !== !s.ok && !(s.ok && r.isError === undefined)) ||
      !Array.isArray(r.content) || r.content.length !== 1) return r;
  const c = r.content[0];
  if (!only(c, ["type", "text"]) || c.type !== "text" || typeof c.text !== "string") return r;
  try {
    // Go's JSON encoder escapes these characters. Compare without parsing text.
    const encoded = JSON.stringify(s).replace(/[<>&\u2028\u2029]/g,
      c => "\\u" + c.charCodeAt(0).toString(16).padStart(4, "0"));
    return c.text === encoded ? s : r;
  } catch { return r; }
}
```

Apply to the result of the actual discovered Mandalore tool, then print once:
`text(mandaloreOutput(result))`. Reuse the helper within an orchestration batch;
do not print the helper, the original wrapper and the selected envelope together.
This example is only for JSON Mandalore responses, not a global renderer for other
plugins. Distinct content, annotations, metadata, malformed/contradictory results
and unexpected versions fall back intact. Preserve errors, conflicts, pending
delivery, truncation/continuation and provenance; none become successful or
complete merely because the presentation is shorter.
