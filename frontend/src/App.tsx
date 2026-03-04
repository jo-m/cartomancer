import { BrowserRouter, Routes, Route } from "react-router-dom"
import { SessionProvider } from "./context/SessionContext"
import Layout from "./components/Layout"
import Home from "./pages/Home"
import Login from "./pages/Login"
import Account from "./pages/Account"

export default function App() {
  return (
    <BrowserRouter>
      <SessionProvider>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<Home />} />
            <Route path="/login" element={<Login />} />
            <Route path="/account" element={<Account />} />
          </Route>
        </Routes>
      </SessionProvider>
    </BrowserRouter>
  )
}
