import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BrowserRouter, Routes, Route } from "react-router-dom"
import { setQueryClient } from "./api/client"
import { SessionProvider } from "./context/SessionContext"
import {
  ProtectedRoute,
  GuestRoute,
  AdminRoute,
} from "./components/ProtectedRoute"
import Layout from "./components/Layout"
import Welcome from "./pages/Welcome"
import Home from "./pages/Home"
import Login from "./pages/Login"
import Register from "./pages/Register"
import ConfirmEmail from "./pages/ConfirmEmail"
import Account from "./pages/Account"
import AccountTracks from "./pages/AccountTracks"
import Upload from "./pages/Upload"
import Track from "./pages/Track"
import AdminUsers from "./pages/AdminUsers"
import AdminForecasts from "./pages/AdminForecasts"

const queryClient = new QueryClient()
setQueryClient(queryClient)

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <SessionProvider>
          <Routes>
            <Route element={<Layout />}>
              <Route path="/" element={<Welcome />} />
              <Route path="/tracks" element={<Home />} />
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
                path="/account/tracks"
                element={
                  <ProtectedRoute>
                    <AccountTracks />
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
              <Route path="/tracks/:uuid" element={<Track />} />
              <Route
                path="/admin/users"
                element={
                  <AdminRoute>
                    <AdminUsers />
                  </AdminRoute>
                }
              />
              <Route
                path="/admin/forecasts"
                element={
                  <AdminRoute>
                    <AdminForecasts />
                  </AdminRoute>
                }
              />
            </Route>
          </Routes>
        </SessionProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
