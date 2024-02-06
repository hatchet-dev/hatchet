import { useEffect, useState } from "react";
import "./App.css";

interface Messages {
  role: "user" | "assistant";
  content: string;
}

function App() {
  const [openRequest, setOpenRequest] = useState<string>();

  const [messages, setMessages] = useState<Messages[]>([
    { role: "user", content: "Hello, how are you?" },
    { role: "assistant", content: "Good how are you?" },
  ]);

  useEffect(() => {
    if (!openRequest) return;

    const sse = new EventSource(`http://localhost:8000/stream/${openRequest}`, {
      withCredentials: true,
    });

    function getMessageStream(data: any) {
      console.log(data);
    }

    sse.onmessage = (e) => {
      console.log(e);
      return getMessageStream(e);
    };

    sse.onerror = () => {
      sse.close();
    };

    return () => {
      sse.close();
    };
  }, [openRequest]);

  const sendMessage = async (content: string) => {
    try {
      const response = await fetch("http://localhost:8000/message", {
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

        <textarea></textarea>
        <button onClick={() => sendMessage("Your message")}>Ask</button>
      </header>
    </div>
  );
}

export default App;
