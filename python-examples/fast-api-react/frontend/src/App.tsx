import React, { useEffect } from "react";
import logo from "./logo.svg";
import "./App.css";

function App() {
  useEffect(() => {
    const sse = new EventSource("http://localhost:8000", {
      withCredentials: true,
    });

    function getRealtimeData(data: any) {
      console.log(data);
      // Process the data here
      // Then pass it to state to be rendered
    }

    sse.onmessage = (e) => {
      console.log(e);
      return getRealtimeData(e);
    };

    sse.onerror = () => {
      // Error log here

      sse.close();
    };

    return () => {
      sse.close();
    };
  }, []);

  return (
    <div className="App">
      <header className="App-header">
        <img src={logo} className="App-logo" alt="logo" />
        <p>
          Edit <code>src/App.tsx</code> and save to reload.
        </p>
        <a
          className="App-link"
          href="https://reactjs.org"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn React
        </a>
      </header>
    </div>
  );
}

export default App;
