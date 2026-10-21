import React, { useState } from "react";

const Badge = ({ count }) => <span className="badge">{count}</span>;

function Panel({ title, items, onPick }) {
  const [open, setOpen] = useState(false);
  return (
    <section className="panel" data-open={open}>
      <h2 onClick={() => setOpen(!open)}>
        {title} <Badge count={items.length} />
      </h2>
      {open ? (
        <ul>
          {items.map((item) => (
            <li key={item.id} onClick={() => onPick(item)}>
              <a href={`/items/${item.id}`}>{item.label}</a>
            </li>
          ))}
        </ul>
      ) : null}
      <footer>
        plain text &amp; an entity
        <br />
      </footer>
    </section>
  );
}

export default function App() {
  return <Panel title="Things" items={[]} onPick={(x) => console.log(x)} />;
}
