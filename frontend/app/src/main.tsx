import ReactDOM from "react-dom/client";
import "./index.css";
import { QueryClientProvider } from "@tanstack/react-query";
import queryClient from "./query-client.tsx";
import Router from "./router.tsx";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <QueryClientProvider client={queryClient}>
    <Router />
  </QueryClientProvider>
);
