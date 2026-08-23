import { useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";

import { MobileSidebar, Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";

export function AppShell() {
  const location = useLocation();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    setMobileMenuOpen(false);
  }, [location.pathname]);

  return (
    <div className="min-h-screen bg-ink-50 text-ink-900">
      <Sidebar />
      <MobileSidebar onClose={() => setMobileMenuOpen(false)} open={mobileMenuOpen} />
      <div className="min-h-screen min-w-0 lg:pl-64">
        <TopBar onOpenMenu={() => setMobileMenuOpen(true)} />
        <main className="min-w-0 overflow-x-hidden px-4 py-5 sm:px-6 lg:px-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
