// A TypeScript module exercising the lexer's rules.
import { readFile as read } from "node:fs/promises";
export { Store as default };

type Primitive = string | number | boolean | null | undefined;

interface Person {
  readonly name: string;
  age?: number;
  greet(other: Person): void;
}

declare module "legacy" {
  export function old(): any;
}

namespace Geometry {
  export const PI = 3.14159;
}

abstract class Store<T extends Primitive> implements Iterable<T> {
  private items: T[] = [];
  protected static instances = 0;

  constructor(public label: string) {
    Store.instances++;
  }

  get size(): number {
    return this.items.length;
  }

  set size(_n: number) {
    throw new Error("read only");
  }

  *[Symbol.iterator](): Iterator<T> {
    yield* this.items;
  }

  async load(path: string): Promise<void> {
    const text = await read(path, { encoding: "utf8" });
    const parsed = JSON.parse(text) as T[];
    this.items = parsed ?? [];
  }
}

const cat = { name: "Garfield" } satisfies Person;
const spread = { ...cat } as const;
const maybe = cat?.name ?? "anonymous";
const label: keyof Person = "name";

function isString(value: unknown): value is string {
  return typeof value === "string";
}

const re = /^[a-z][\w-]*$/giu;
const tpl = `name=${cat.name} age=${(cat as any).age ?? 0}`;
const hex = 0xFF, oct = 0o17, bin = 0b1010, flt = 1.5e3, int = 42;

switch (label) {
  case "name":
    console.log(maybe, tpl, hex, oct, bin, flt, int, re.test(label));
    break;
  default:
    break;
}

export type Partialed = Partial<Record<string, ReturnType<typeof isString>>>;
