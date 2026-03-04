import { Link, Outlet, useNavigate } from "react-router-dom"
import { useSession } from "../context/SessionContext"

export default function Layout() {
  const { user, loading, logout } = useSession()
  const navigate = useNavigate()

  async function handleLogout() {
    await logout()
    navigate("/login")
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <Link to="/" className="text-lg font-semibold text-gray-900">
            Detour
          </Link>
          <div className="flex items-center gap-4 text-sm">
            {!loading &&
              (user ? (
                <>
                  <span className="text-gray-600">{user.name}</span>
                  <Link
                    to="/account"
                    className="text-gray-700 hover:text-gray-900"
                  >
                    Account
                  </Link>
                  <button
                    onClick={handleLogout}
                    className="cursor-pointer text-gray-700 hover:text-gray-900"
                  >
                    Logout
                  </button>
                </>
              ) : (
                <Link to="/login" className="text-gray-700 hover:text-gray-900">
                  Login
                </Link>
              ))}
          </div>
        </div>
      </nav>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
