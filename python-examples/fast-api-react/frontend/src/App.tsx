import { useEffect, useState } from "react";
import "./App.css";

interface Messages {
  role: "user" | "assistant";
  content: string;
}

const API_URL = "http://localhost:8000";

function App() {
  const [openRequest, setOpenRequest] = useState<string>();

  const [messages, setMessages] = useState<Messages[]>([
    { role: "user", content: "Hello, how are you?" },
    { role: "assistant", content: "Good how are you?" },
  ]);

  const [status, setStatus] = useState("idle");

  useEffect(() => {
    if (!openRequest) return;

    const sse = new EventSource(`${API_URL}/stream/${openRequest}`, {
      withCredentials: true,
    });

    function getMessageStream(data: any) {
      console.log(data);
      if (data === null) return;
      setStatus(data.status);
      if (data.message) {
        setMessages((prev) => [
          ...prev,
          { role: "assistant", content: data.message },
        ]);
      }
    }

    sse.onmessage = (e) => getMessageStream(JSON.parse(e.data));

    sse.onerror = () => {
      setOpenRequest(undefined);
      sse.close();
    };

    return () => {
      setOpenRequest(undefined);
      sse.close();
    };
  }, [openRequest]);

  const sendMessage = async (content: string) => {
    try {
      setMessages((prev) => [...prev, { role: "user", content }]);

      const response = await fetch(`${API_URL}/message`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          messages: [
            ...messages,
            {
              role: "user",
              content,
            },
          ],
        }),
      });

      if (response.ok) {
        // Handle successful response
        setOpenRequest((await response.json()).workflowRunId);
      } else {
        // Handle error response
      }
    } catch (error) {
      // Handle network error
    }
  };

  return (
    <div className="App">
      <header>
        {messages.map(({ role, content }, i) => (
          <p key={i}>
            {role}: {content}.
          </p>
        ))}

        {status}
        <textarea></textarea>
        <button onClick={() => sendMessage("Your message")}>Ask</button>
      </header>
    </div>
  );
}

export default App;
