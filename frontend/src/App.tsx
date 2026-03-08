import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BrowserRouter, Routes, Route } from "react-router-dom"
import { setQueryClient } from "./api/client"
import { SessionProvider } from "./context/SessionContext"
import { ProtectedRoute, GuestRoute } from "./components/ProtectedRoute"
import Layout from "./components/Layout"
import Home from "./pages/Home"
import Login from "./pages/Login"
import Register from "./pages/Register"
import ConfirmEmail from "./pages/ConfirmEmail"
import Account from "./pages/Account"
import Upload from "./pages/Upload"
import TrackList from "./pages/TrackList"
import Track from "./pages/Track"

const queryClient = new QueryClient()
setQueryClient(queryClient)

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <SessionProvider>
          <Routes>
            <Route element={<Layout />}>
              <Route path="/" element={<Home />} />
              <Route
                path="/login"
                element={
                  <GuestRoute>
                    <Login />
                  </GuestRoute>
                }
              />
              <Route
                path="/register"
                element={
                  <GuestRoute>
                    <Register />
                  </GuestRoute>
                }
              />
              <Route path="/confirm-email" element={<ConfirmEmail />} />
              <Route
                path="/account"
                element={
                  <ProtectedRoute>
                    <Account />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/upload"
                element={
                  <ProtectedRoute>
                    <Upload />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/tracks"
                element={
                  <ProtectedRoute>
                    <TrackList />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/tracks/:uuid"
                element={
                  <ProtectedRoute>
                    <Track />
                  </ProtectedRoute>
                }
              />
            </Route>
          </Routes>
        </SessionProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
