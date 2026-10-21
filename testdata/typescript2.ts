// Edge cases, kept apart from the readable corpus. Every line here is still
// lexed by the reference gem, so the golden is the gem's answer for them too.

// `as` glued to an alphanumeric: the gem's lookbehind refuses it, and `as` is
// an ordinary name rather than a declaration keyword.
const glued = 1as b;
// `as` after a dot is refused as well (cy.get('kitten').as('friend')).
cy.get("kitten").as("friend");
// but after a spread it is accepted.
const spread = {...cat} as const;

// An optional property label in a statement position.
function f() {
  label?: 1;
}

// The widened TypeScript word sets.
let a: any, b: void, c: number, d: string, e: keyof Person;
declare interface I {}
module M {}
export default class {}
abstract class A { readonly x = 1; }
type Fn = Parameters<typeof f>;
const caps: Capitalize<"x"> = "X";
const nn: NonNullable<string | null> = "y";
const om: Omit<I, "x"> = {} as Omit<I, "x">;
const rq: Required<I> = {} as Required<I>;
const ti: ThisType<I> = {} as ThisType<I>;
const up: Uppercase<"x"> = "X";
if (isit is string) {}
