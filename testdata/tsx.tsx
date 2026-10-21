import React, { useCallback, type ReactNode } from "react";

interface Item {
  id: number;
  label: string;
}

type Props = {
  title: string;
  items: readonly Item[];
  children?: ReactNode;
  onPick?: (item: Item) => void;
};

const identity = <T,>(value: T): T => value;

export function Panel({ title, items, children, onPick }: Props) {
  const pick = useCallback(
    (item: Item) => onPick?.(item),
    [onPick]
  );

  return (
    <section className="panel">
      <h2>{title}</h2>
      <ul>
        {items.map((item) => (
          <li key={item.id} onClick={() => pick(item)}>
            {item.label as string}
          </li>
        ))}
      </ul>
      {children}
    </section>
  );
}

export default function App(): JSX.Element {
  const items = identity<Item[]>([{ id: 1, label: "one" }]);
  return <Panel title="Things" items={items} />;
}
