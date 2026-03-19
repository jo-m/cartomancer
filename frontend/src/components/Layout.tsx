import { useRef, useState } from "react"
import { Link, Outlet, useNavigate } from "react-router-dom"
import { useSession } from "../context/SessionContext"

export default function Layout() {
  const { user, loading, logout } = useSession()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  async function handleLogout() {
    setMenuOpen(false)
    await logout()
    navigate("/login")
  }

  function handleMouseEnter() {
    if (closeTimer.current) clearTimeout(closeTimer.current)
    setMenuOpen(true)
  }

  function handleMouseLeave() {
    closeTimer.current = setTimeout(() => setMenuOpen(false), 150)
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="border-b border-gray-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <Link to="/" className="text-lg font-semibold text-gray-900">
            Detour
          </Link>
          <div className="flex items-center gap-4 text-sm">
            <Link to="/tracks" className="text-gray-700 hover:text-gray-900">
              Public Tracks
            </Link>
            <Link to="/about" className="text-gray-700 hover:text-gray-900">
              About
            </Link>
            {!loading &&
              (user ? (
                <>
                  <Link
                    to="/account/tracks"
                    className="text-gray-700 hover:text-gray-900"
                  >
                    My Tracks
                  </Link>
                  <Link
                    to="/tracks/groups"
                    className="text-gray-700 hover:text-gray-900"
                  >
                    Groups
                  </Link>
                  <Link
                    to="/upload"
                    className="text-gray-700 hover:text-gray-900"
                  >
                    Upload
                  </Link>
                  {user.admin && (
                    <Link
                      to="/admin/users"
                      className="text-gray-700 hover:text-gray-900"
                    >
                      Admin
                    </Link>
                  )}
                  <div
                    className="relative"
                    onMouseEnter={handleMouseEnter}
                    onMouseLeave={handleMouseLeave}
                  >
                    <img
                      src={`/api/users/${user.uuid}/avatar?v=${user.avatarSeed}`}
                      alt={user.name}
                      className="h-8 w-8 cursor-pointer rounded-full"
                    />
                    {menuOpen && (
                      <div className="absolute right-0 top-full z-50 mt-1 w-40 rounded border border-gray-200 bg-white py-1 shadow-md">
                        <div className="px-3 py-2 text-xs font-medium text-gray-500">
                          {user.name}
                        </div>
                        <hr className="border-gray-100" />
                        <Link
                          to="/account"
                          className="block px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          onClick={() => setMenuOpen(false)}
                        >
                          Account
                        </Link>
                        <button
                          onClick={handleLogout}
                          className="w-full cursor-pointer px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-50"
                        >
                          Logout
                        </button>
                      </div>
                    )}
                  </div>
                </>
              ) : (
                <>
                  <Link
                    to="/register"
                    className="text-gray-700 hover:text-gray-900"
                  >
                    Register
                  </Link>
                  <Link
                    to="/login"
                    className="text-gray-700 hover:text-gray-900"
                  >
                    Login
                  </Link>
                </>
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
