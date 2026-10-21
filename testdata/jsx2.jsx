// JSX edge cases, kept apart from the readable corpus: a brace that opens
// while the lexer is in :interpol rather than in an expression start, and a
// nested brace that goes through :interpol_inner. Both are lexed by the
// reference gem, so the golden is the gem's answer for them too.
const odd = <div>{foo {bar}}</div>;

// A fragment, a dotted component name, a self-closing tag with a quoted
// attribute, and raw HTML text between the tags.
const frag = (
  <>
    <Ns.Deep.Component name='single' other="double" flag />
    plain &amp; text <b>bold</b>
    {/* a comment inside an interpolation */}
  </>
);
