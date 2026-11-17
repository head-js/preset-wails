import { useEffect, useState } from "react";

function App() {
  const [count, setCount] = useState(0);
  const [version, setVersion] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch("http://127.0.0.1:20080/api/version")
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((data) => setVersion(data.version))
      .catch((err) => setError(err.message));
  }, []);

  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 overflow-hidden">
      <h1 className="text-2xl font-bold">Hello React</h1>
      <p>Count: {count}</p>
      <button className="btn btn-primary" onClick={() => setCount((c) => c + 1)}>
        +1
      </button>
      {error ? <p>Error: {error}</p> : <p>App version: {version ?? "Loading…"}</p>}
    </div>
  );
}

export default App;
