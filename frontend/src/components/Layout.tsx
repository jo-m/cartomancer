import { useEffect, useRef, useState } from "react"
import { Link, Outlet, useNavigate } from "react-router-dom"
import { useSession } from "../context/SessionContext"
import { useAppConfig } from "../api/client"
import logoSvg from "../assets/logo.svg"

export default function Layout() {
  const { user, loading, logout } = useSession()
  const { data: appConfig } = useAppConfig()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [tracksMenuOpen, setTracksMenuOpen] = useState(false)
  const tracksMenuTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (appConfig?.instanceName) {
      document.title = appConfig.instanceName
    }
  }, [appConfig?.instanceName])

  async function handleLogout() {
    setMenuOpen(false)
    await logout()
    navigate("/login")
  }

  function handleMouseEnter() {
    if (menuTimer.current) clearTimeout(menuTimer.current)
    setMenuOpen(true)
  }

  function handleMouseLeave() {
    menuTimer.current = setTimeout(() => setMenuOpen(false), 150)
  }

  function handleTracksMenuEnter() {
    if (tracksMenuTimer.current) clearTimeout(tracksMenuTimer.current)
    setTracksMenuOpen(true)
  }

  function handleTracksMenuLeave() {
    tracksMenuTimer.current = setTimeout(() => setTracksMenuOpen(false), 150)
  }

  return (
    <div className="flex min-h-screen flex-col bg-gray-50">
      <nav className="sticky top-0 z-40 border-b border-gray-200 bg-white">
        <div className="relative mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-4">
            <Link to="/" className="text-lg font-semibold text-gray-900">
              {appConfig?.instanceName}
            </Link>
            <Link
              to="/tracks"
              className="text-sm text-gray-700 hover:text-gray-900"
            >
              Public Tracks
            </Link>
          </div>
          <Link to="/" className="absolute left-1/2 -translate-x-1/2">
            <img
              src={logoSvg}
              alt="Logo"
              className="h-16 w-16 rounded-full border border-gray-200"
            />
          </Link>
          <div className="flex items-center gap-4 text-sm">
            {!loading &&
              (user ? (
                <>
                  <div
                    className="relative flex items-center self-stretch"
                    onMouseEnter={handleTracksMenuEnter}
                    onMouseLeave={handleTracksMenuLeave}
                  >
                    <Link
                      to="/account/tracks"
                      className="text-gray-700 hover:text-gray-900"
                    >
                      My Tracks
                    </Link>
                    {tracksMenuOpen && (
                      <div className="absolute left-0 top-full z-50 mt-1 w-40 rounded border border-gray-200 bg-white py-1 shadow-md">
                        <Link
                          to="/account/tracks"
                          className="block px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          onClick={() => setTracksMenuOpen(false)}
                        >
                          My Tracks
                        </Link>
                        <Link
                          to="/tracks/groups"
                          className="block px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          onClick={() => setTracksMenuOpen(false)}
                        >
                          Groups
                        </Link>
                        <Link
                          to="/upload"
                          className="block px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
                          onClick={() => setTracksMenuOpen(false)}
                        >
                          Upload
                        </Link>
                      </div>
                    )}
                  </div>
                  <div
                    className="relative flex items-center self-stretch"
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
                        {user.admin && (
                          <Link
                            to="/admin/users"
                            className="block px-3 py-2 text-sm text-gray-700 hover:bg-gray-50"
                            onClick={() => setMenuOpen(false)}
                          >
                            Admin
                          </Link>
                        )}
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
                  {appConfig?.registrationEnabled && (
                    <Link
                      to="/register"
                      className="text-gray-700 hover:text-gray-900"
                    >
                      Register
                    </Link>
                  )}
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
      <main className="flex-1">
        <Outlet />
      </main>
      <footer className="border-t border-gray-200 bg-white">
        <div className="mx-auto flex max-w-5xl items-center justify-start px-4 py-2 text-xs text-gray-400">
          <Link to="/about" className="hover:text-gray-700">
            About
          </Link>
        </div>
      </footer>
    </div>
  )
}
