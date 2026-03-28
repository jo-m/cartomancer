import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BrowserRouter, Routes, Route } from "react-router-dom"
import { ApiError, setQueryClient } from "./api/client"
import { SessionProvider } from "./context/SessionContext"
import {
  ProtectedRoute,
  GuestRoute,
  AdminRoute,
} from "./components/ProtectedRoute"
import Layout from "./components/Layout"
import Welcome from "./pages/Welcome"
import Tracks from "./pages/Tracks"
import Login from "./pages/Login"
import Register from "./pages/Register"
import ConfirmEmail from "./pages/ConfirmEmail"
import Account from "./pages/Account"
import AccountTracks from "./pages/AccountTracks"
import Upload from "./pages/Upload"
import Track from "./pages/Track"
import Groups from "./pages/Groups"
import GroupDetail from "./pages/GroupDetail"
import Segments from "./pages/Segments"
import SegmentDetail from "./pages/SegmentDetail"
import AdminUsers from "./pages/AdminUsers"
import AdminForecasts from "./pages/AdminForecasts"
import About from "./pages/About"
import Leaving from "./pages/Leaving"
import NotFound from "./pages/NotFound"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        if (
          error instanceof ApiError &&
          error.status >= 400 &&
          error.status < 500
        ) {
          return false
        }
        return failureCount < 3
      },
    },
  },
})
setQueryClient(queryClient)

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <SessionProvider>
          <Routes>
            <Route element={<Layout />}>
              <Route path="/" element={<Welcome />} />
              <Route path="/tracks" element={<Tracks />} />
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
              <Route
                path="/tracks/groups"
                element={
                  <ProtectedRoute>
                    <Groups />
                  </ProtectedRoute>
                }
              />
              <Route
                path="/tracks/groups/:uuid"
                element={
                  <ProtectedRoute>
                    <GroupDetail />
                  </ProtectedRoute>
                }
              />
              <Route path="/tracks/:uuid" element={<Track />} />
              <Route
                path="/admin/segments"
                element={
                  <AdminRoute>
                    <Segments />
                  </AdminRoute>
                }
              />
              <Route
                path="/admin/segments/:uuid"
                element={
                  <AdminRoute>
                    <SegmentDetail />
                  </AdminRoute>
                }
              />
              <Route path="/about" element={<About />} />
              <Route path="/leaving" element={<Leaving />} />
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
              <Route path="*" element={<NotFound />} />
            </Route>
          </Routes>
        </SessionProvider>
      </BrowserRouter>
    </QueryClientProvider>
  )
}
