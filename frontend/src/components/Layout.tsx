import { useCallback, useRef, useState } from "react"
import { Link, Outlet, useNavigate } from "react-router-dom"
import { ChevronDownIcon } from "@heroicons/react/20/solid"
import { useSession } from "../context/SessionContext"
import { useAppConfig } from "../api/client"
import logoSvg from "../assets/logo.svg?raw"
import SvgIcon from "../assets/SvgIcon"
import ornamentDividerSvg from "../assets/ornament-divider.svg?raw"

/** Manages open/close state for a dropdown with both hover and keyboard support. */
function useDropdown() {
  const [open, setOpen] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const clearTimer = useCallback(() => {
    if (timer.current) clearTimeout(timer.current)
  }, [])

  const delayedClose = useCallback(() => {
    timer.current = setTimeout(() => setOpen(false), 150)
  }, [])

  const handleMouseEnter = useCallback(() => {
    clearTimer()
    setOpen(true)
  }, [clearTimer])

  const handleMouseLeave = useCallback(() => {
    delayedClose()
  }, [delayedClose])

  const handleFocusCapture = useCallback(() => {
    clearTimer()
    setOpen(true)
  }, [clearTimer])

  const handleBlurCapture = useCallback(() => {
    delayedClose()
  }, [delayedClose])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false)
        const trigger = containerRef.current?.querySelector<HTMLElement>(
          "[data-dropdown-trigger]"
        )
        trigger?.focus()
        return
      }
      if (!open) return

      const items =
        containerRef.current?.querySelectorAll<HTMLElement>('[role="menuitem"]')
      if (!items?.length) return

      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault()
        const focused = document.activeElement as HTMLElement
        const idx = Array.from(items).indexOf(focused)
        let next: number
        if (e.key === "ArrowDown") {
          next = idx < 0 ? 0 : (idx + 1) % items.length
        } else {
          next = idx <= 0 ? items.length - 1 : idx - 1
        }
        items[next].focus()
      }
    },
    [open]
  )

  const close = useCallback(() => setOpen(false), [])

  return {
    open,
    close,
    containerRef,
    containerProps: {
      ref: containerRef,
      onMouseEnter: handleMouseEnter,
      onMouseLeave: handleMouseLeave,
      onFocusCapture: handleFocusCapture,
      onBlurCapture: handleBlurCapture,
      onKeyDown: handleKeyDown,
    },
  }
}

export default function Layout() {
  const { user, loading, logout } = useSession()
  const { data: appConfig } = useAppConfig()
  const navigate = useNavigate()
  const tracksMenu = useDropdown()
  const userMenu = useDropdown()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  async function handleLogout() {
    userMenu.close()
    setMobileMenuOpen(false)
    await logout()
    navigate("/login")
  }

  return (
    <div className="flex min-h-screen flex-col bg-surface">
      <nav
        className="sticky top-0 z-40 overflow-visible border-b border-border bg-nav"
        role="navigation"
        aria-label="Main navigation"
      >
        <div className="relative mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
          <div className="flex items-center gap-4">
            <Link
              to="/"
              className="relative z-50 flex flex-col items-center"
              aria-label="Home"
            >
              <span
                className="-my-4 block h-16 w-16 shrink-0 rounded-full border border-border-hover bg-nav text-nav-text [&>svg]:h-full [&>svg]:w-full"
                dangerouslySetInnerHTML={{ __html: logoSvg }}
                aria-hidden="true"
              />
              {appConfig?.instanceName && (
                <svg
                  className="absolute top-full -mt-4.5 -z-10 h-auto w-20 overflow-visible"
                  viewBox="0 0 100 30"
                  aria-hidden="true"
                >
                  <defs>
                    <path
                      id="title-arc"
                      d="M 8,28 A 50,50 0 0,0 92,28"
                      fill="none"
                    />
                  </defs>
                  <path
                    d="M 8,28 A 50,50 0 0,0 92,28"
                    className="stroke-nav"
                    fill="none"
                    strokeWidth="20"
                    strokeLinecap="round"
                  />
                  <text
                    className="fill-nav-text font-semibold"
                    fontSize="13"
                    textAnchor="middle"
                    letterSpacing="1"
                    dy="4"
                  >
                    <textPath href="#title-arc" startOffset="50%">
                      {appConfig.instanceName}
                    </textPath>
                  </text>
                </svg>
              )}
            </Link>
            <Link
              to="/tracks"
              className="hidden text-sm text-nav-text/70 hover:text-nav-text transition-colors sm:block"
            >
              Public Tracks
            </Link>
          </div>
          {/* Mobile menu button */}
          <button
            type="button"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="cursor-pointer rounded p-1.5 text-nav-text/70 hover:text-nav-text sm:hidden"
            aria-expanded={mobileMenuOpen}
            aria-label="Toggle menu"
          >
            <svg
              className="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              {mobileMenuOpen ? (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M6 18L18 6M6 6l12 12"
                />
              ) : (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M4 6h16M4 12h16M4 18h16"
                />
              )}
            </svg>
          </button>
          {/* Desktop nav */}
          <div className="hidden items-center gap-4 text-sm sm:flex">
            {!loading &&
              (user ? (
                <>
                  <div
                    className={`relative -my-3 flex items-center self-stretch px-3 py-3 transition-colors cursor-pointer ${tracksMenu.open ? "bg-panel" : ""}`}
                    {...tracksMenu.containerProps}
                  >
                    <Link
                      to="/account/tracks"
                      data-dropdown-trigger
                      aria-haspopup="true"
                      aria-expanded={tracksMenu.open}
                      className={`flex items-center gap-1 transition-colors ${tracksMenu.open ? "text-text-secondary" : "text-nav-text/70 hover:text-nav-text"}`}
                    >
                      My Tracks
                      <ChevronDownIcon className="h-4 w-4" />
                    </Link>
                    {tracksMenu.open && (
                      <div
                        className="absolute left-0 top-full z-50 w-40 rounded-b border-x border-b border-border bg-panel py-1 shadow-lg"
                        role="menu"
                      >
                        <Link
                          to="/account/tracks"
                          className="block px-3 py-2 text-sm text-text-secondary hover:bg-surface transition-colors"
                          onClick={tracksMenu.close}
                          role="menuitem"
                        >
                          Tracks
                        </Link>
                        <Link
                          to="/tracks/groups"
                          className="block px-3 py-2 text-sm text-text-secondary hover:bg-surface transition-colors"
                          onClick={tracksMenu.close}
                          role="menuitem"
                        >
                          Groups
                        </Link>
                        <Link
                          to="/tracks/upload"
                          className="flex items-center gap-1.5 px-3 py-2 text-sm text-text-secondary hover:bg-surface transition-colors"
                          onClick={tracksMenu.close}
                          role="menuitem"
                        >
                          Upload
                        </Link>
                      </div>
                    )}
                  </div>
                  <div
                    className={`relative -my-3 flex items-center self-stretch px-3 py-3 transition-colors ${userMenu.open ? "bg-panel" : ""}`}
                    {...userMenu.containerProps}
                  >
                    <button
                      type="button"
                      data-dropdown-trigger
                      aria-haspopup="true"
                      aria-expanded={userMenu.open}
                      className={`flex cursor-pointer items-center gap-2 transition-colors ${userMenu.open ? "text-text-secondary" : "text-nav-text/70 hover:text-nav-text"}`}
                    >
                      <img
                        src={`/api/users/${user.uuid}/avatar?v=${user.avatarSeed}`}
                        alt={user.name}
                        className="h-8 w-8 rounded-full border border-border-hover"
                      />
                      <span className="text-sm">{user.name}</span>
                      <ChevronDownIcon className="h-4 w-4" />
                    </button>
                    {userMenu.open && (
                      <div
                        className="absolute right-0 top-full z-50 w-40 rounded-b border-x border-b border-border bg-panel py-1 shadow-lg"
                        role="menu"
                      >
                        <Link
                          to="/account"
                          className="block px-3 py-2 text-sm text-text-secondary hover:bg-surface transition-colors"
                          onClick={userMenu.close}
                          role="menuitem"
                        >
                          Account
                        </Link>
                        {user.admin && (
                          <Link
                            to="/admin/users"
                            className="block px-3 py-2 text-sm text-text-secondary hover:bg-surface transition-colors"
                            onClick={userMenu.close}
                            role="menuitem"
                          >
                            Admin
                          </Link>
                        )}
                        <button
                          onClick={handleLogout}
                          className="w-full cursor-pointer px-3 py-2 text-left text-sm text-text-secondary hover:bg-surface transition-colors"
                          role="menuitem"
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
                      className="text-nav-text/70 hover:text-nav-text transition-colors"
                    >
                      Register
                    </Link>
                  )}
                  <Link
                    to="/login"
                    className="text-nav-text/70 hover:text-nav-text transition-colors"
                  >
                    Login
                  </Link>
                </>
              ))}
          </div>
        </div>
        {/* Mobile nav dropdown */}
        {mobileMenuOpen && (
          <div className="border-t border-border bg-nav px-4 pb-3 pt-2 sm:hidden">
            <Link
              to="/tracks"
              className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
              onClick={() => setMobileMenuOpen(false)}
            >
              Public Tracks
            </Link>
            {!loading &&
              (user ? (
                <>
                  <Link
                    to="/account/tracks"
                    className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    My Tracks
                  </Link>
                  <Link
                    to="/tracks/groups"
                    className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    Groups
                  </Link>
                  <Link
                    to="/tracks/upload"
                    className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    Upload
                  </Link>
                  <Link
                    to="/account"
                    className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    Account
                  </Link>
                  {user.admin && (
                    <Link
                      to="/admin/users"
                      className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
                      onClick={() => setMobileMenuOpen(false)}
                    >
                      Admin
                    </Link>
                  )}
                  <button
                    onClick={handleLogout}
                    className="block w-full cursor-pointer py-2 text-left text-sm text-nav-text/70 hover:text-nav-text"
                  >
                    Logout
                  </button>
                </>
              ) : (
                <>
                  {appConfig?.registrationEnabled && (
                    <Link
                      to="/register"
                      className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
                      onClick={() => setMobileMenuOpen(false)}
                    >
                      Register
                    </Link>
                  )}
                  <Link
                    to="/login"
                    className="block py-2 text-sm text-nav-text/70 hover:text-nav-text"
                    onClick={() => setMobileMenuOpen(false)}
                  >
                    Login
                  </Link>
                </>
              ))}
          </div>
        )}
      </nav>
      <main className="flex-1">
        <Outlet />
      </main>
      <footer className="border-t border-border bg-panel">
        <div className="mx-auto max-w-5xl px-4 pt-3 pb-2">
          <div className="mb-2 flex justify-center">
            <SvgIcon
              svg={ornamentDividerSvg}
              className="h-2.5 w-40 text-border"
            />
          </div>
          <div className="flex items-center gap-4 justify-start text-xs text-text-muted">
            <Link
              to="/about"
              className="hover:text-text-secondary transition-colors"
            >
              About
            </Link>
            <Link
              to="/contact"
              className="hover:text-text-secondary transition-colors"
            >
              Contact
            </Link>
            <Link
              to="/help"
              className="hover:text-text-secondary transition-colors"
            >
              Help
            </Link>
          </div>
        </div>
      </footer>
    </div>
  )
}
