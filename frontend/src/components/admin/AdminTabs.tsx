import { Link } from "react-router-dom"

export type AdminTab = "users" | "forecasts" | "maps"

interface AdminTabsProps {
  /** Identifier of the currently active admin tab. */
  current: AdminTab
}

const tabs: ReadonlyArray<{ id: AdminTab; label: string; href: string }> = [
  { id: "users", label: "Users", href: "/admin/users" },
  { id: "forecasts", label: "Forecasts", href: "/admin/forecasts" },
  { id: "maps", label: "Maps", href: "/admin/maps" },
]

/** Page header with the "Admin" title and the tab navigation links. */
export default function AdminTabs({ current }: AdminTabsProps) {
  return (
    <div className="mb-6 flex flex-wrap items-center gap-x-4 gap-y-2">
      <h1 className="text-2xl font-semibold text-text">Admin</h1>
      {tabs.map((t) => (
        <Link
          key={t.id}
          to={t.href}
          className={
            t.id === current
              ? "border-b-2 border-primary pb-0.5 text-sm font-medium text-text"
              : "pb-0.5 text-sm font-medium text-text-muted hover:text-text-secondary transition-colors"
          }
          aria-current={t.id === current ? "page" : undefined}
        >
          {t.label}
        </Link>
      ))}
    </div>
  )
}
