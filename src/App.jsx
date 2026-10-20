import { useEffect, useState } from "react";

function App() {
  const [count, setCount] = useState(0);
  const [version, setVersion] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch("/api/version")
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then((data) => setVersion(data.version))
      .catch((err) => setError(err.message));
  }, []);

  return (
    <div style={styles.container}>
      <h1>Hello React</h1>
      <p>Count: {count}</p>
      <button onClick={() => setCount((c) => c + 1)}>+1</button>
      {error ? <p>Error: {error}</p> : <p>App version: {version ?? "Loading…"}</p>}
    </div>
  );
}

const styles = {
  container: {
    height: "100vh",
    background: "rgba(0, 0, 0, 0.5)",
    color: "#fff",
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    gap: "16px",
    overflow: "hidden",
  },
};

export default App;
