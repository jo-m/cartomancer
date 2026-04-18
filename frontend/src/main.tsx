import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import "@fontsource/fondamento"
import "@fontsource/crimson-text"
import "@fontsource/crimson-text/700.css"
import "@fontsource/crimson-text/600.css"
import "./index.css"
import App from "./App"
import ErrorBoundary from "./components/ErrorBoundary"

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </StrictMode>
)
