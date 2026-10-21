// TSX edge cases: the two rules TSX prepends to :element_name -- a generic
// type parameter list written `<T,>` (which TSX pops out of the element it had
// started to open) and a `<` that opens a type argument list on a component.
const identity = <T,>(value: T): T => value;

export const Typed = () => <Box<string> prop="x" />;

export const Both = () => (
  <Wrapper<number, string> id={1}>
    <Inner<boolean> flag />
  </Wrapper>
);
