import { Outlet } from "react-router-dom"

export function AppShell() {
  return (
    <div className="min-h-svh bg-background">
      <main className="mx-auto w-full max-w-6xl px-6 py-10">
        <Outlet />
      </main>
    </div>
  )
}
