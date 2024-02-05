import React, { useEffect, useState } from "react";
import logo from "./logo.svg";
import "./App.css";

interface Messages {
  role: "user" | "assistant";
  message: string;
}

function App() {
  const [messages, setMessages] = useState<Messages[]>([
    { role: "user", message: "Hello, how are you?" },
    { role: "assistant", message: "Good how are you?" },
  ]);

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
      sse.close();
    };

    return () => {
      sse.close();
    };
  }, []);

  return (
    <div className="App">
      <header>
        {messages.map(({ role, message }, i) => (
          <p key={i}>
            {role}: {message}.
          </p>
        ))}

        <textarea></textarea>
        <button>Ask</button>
      </header>
    </div>
  );
}

export default App;
