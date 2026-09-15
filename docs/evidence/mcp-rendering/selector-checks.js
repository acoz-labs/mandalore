// Call verifyMandaloreOutput(mandaloreOutput) in the code-mode JavaScript runtime.
// The selector is extracted verbatim from the bundled skill reference; no Node
// dependency or signet access is needed. These are synthetic representation tests.
function verifyMandaloreOutput(select) {
  const envelope = { protocol_version: 1, ok: true, result: {
    current: [{ body: 'Silver Heron < Copper Finch & café\n"yes"' }],
    conflicts: [{ heads: ["revision-a", "revision-b"] }], truncated: true,
    next_offset: 5, source_pin: "synthetic-pin", citation: "synthetic-citation"
  }};
  const goJSON = x => JSON.stringify(x).replace(/[<>&\u2028\u2029]/g,
    c => "\\u" + c.charCodeAt(0).toString(16).padStart(4, "0"));
  const wrap = s => ({ content: [{ type: "text", text: goJSON(s) }], structuredContent: s, isError: !s.ok });
  let count = 0;
  const expect = (name, input, selected) => {
    const before = JSON.stringify(input);
    if (select(input) !== selected) throw Error(name + ": wrong representation");
    if (JSON.stringify(input) !== before) throw Error(name + ": input mutated");
    count++;
  };
  const original = wrap(envelope);
  expect("success, conflicts, truncation, provenance", original, envelope);
  const failure = { protocol_version: 1, ok: false, error: {
    code: "sync.failed", write_may_have_occurred: true, inspect_before_retry: true,
    sync_status: { state: "pending", delivered: false }
  }};
  expect("pending/error", wrap(failure), failure);
  const noFlag = wrap(envelope); delete noFlag.isError;
  expect("omitted false isError", noFlag, envelope);
  expect("already unwrapped", envelope, envelope);
  const textOnly = { content: original.content, isError: false };
  expect("text-only", textOnly, textOnly);
  for (const [name, mutate] of [
    ["different text", x => { x.content[0].text = '{"different":true}'; }],
    ["extra block", x => { x.content.push({ type: "text", text: "distinct evidence" }); }],
    ["annotations", x => { x.content[0].annotations = { audience: ["user"] }; }],
    ["wrapper metadata", x => { x._meta = { notice: "distinct" }; }],
    ["contradictory error", x => { x.isError = true; }],
    ["unknown wrapper field", x => { x.notice = "distinct"; }],
    ["image", x => { x.content = [{type:"image", data:"synthetic", mimeType:"image/png"}]; }],
    ["malformed envelope", x => { x.structuredContent.ok = "true"; }],
    ["unknown version", x => { x.structuredContent.protocol_version = 2; }],
    ["large rounded number", x => { x.content[0].text = '{"protocol_version":1,"ok":true,"result":{"n":9007199254740993}}'; x.structuredContent = JSON.parse(x.content[0].text); }],
    ["duplicate JSON key", x => { x.content[0].text = '{"protocol_version":1,"ok":true,"ok":false,"result":{}}'; }],
  ]) {
    const input = JSON.parse(JSON.stringify(original)); mutate(input);
    expect(name, input, input);
  }
  for (const value of [null, undefined, "already text", [], {}]) expect("non-wrapper", value, value);
  const bytes = value => encodeURIComponent(JSON.stringify(value)).replace(/%[0-9A-F]{2}|./g, "x").length;
  return { passed: count, serialized_wrapper_bytes: bytes(original),
    selected_envelope_bytes: bytes(select(original)),
    note: "synthetic UTF-8 bytes, not native token measurements" };
}
