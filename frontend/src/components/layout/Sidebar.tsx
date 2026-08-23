import {
  Bot,
  Brain,
  CalendarClock,
  Database,
  FileStack,
  Hammer,
  LayoutDashboard,
  MessagesSquare,
  PlugZap,
  ScrollText,
  Settings2,
  Store,
  TerminalSquare,
  X
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";

export const navItems = [
  { to: "/app/chat", key: "nav.chat", icon: MessagesSquare },
  { to: "/app/agent", key: "nav.agent", icon: Bot },
  { to: "/app/providers", key: "nav.providers", icon: Settings2 },
  { to: "/app/memory", key: "nav.memory", icon: Brain },
  { to: "/app/knowledge", key: "nav.knowledge", icon: FileStack },
  { to: "/app/market", key: "nav.market", icon: Store },
  { to: "/app/tools", key: "nav.tools", icon: Hammer },
  { to: "/app/tasks", key: "nav.tasks", icon: CalendarClock },
  { to: "/app/channels", key: "nav.channels", icon: PlugZap },
  { to: "/app/workspace", key: "nav.workspace", icon: TerminalSquare },
  { to: "/app/logs", key: "nav.logs", icon: ScrollText }
];

export function Sidebar() {
  const { t } = useTranslation();

  return (
    <aside className="fixed inset-y-0 left-0 z-40 hidden w-64 border-r border-ink-200 bg-white lg:block">
      <div className="flex h-16 items-center gap-3 border-b border-ink-100 px-5">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-ink-900 text-white">
          <LayoutDashboard className="h-5 w-5" />
        </div>
        <div>
          <div className="text-sm font-semibold text-ink-900">FreeDinnerAgent</div>
          <div className="text-xs text-ink-500">{t("app.subtitle")}</div>
        </div>
      </div>

      <SidebarNav />

      <div className="absolute bottom-0 left-0 right-0 border-t border-ink-100 p-4">
        <div className="flex items-center gap-2 rounded-md bg-ink-50 px-3 py-2 text-xs text-ink-500">
          <Database className="h-4 w-4 text-mint-600" />
          <span>{t("app.backendReady")}</span>
        </div>
      </div>
    </aside>
  );
}

type MobileSidebarProps = {
  open: boolean;
  onClose: () => void;
};

export function MobileSidebar({ onClose, open }: MobileSidebarProps) {
  const { t } = useTranslation();

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 lg:hidden">
      <button
        aria-label={t("common.close")}
        className="absolute inset-0 bg-ink-900/40"
        onClick={onClose}
        type="button"
      />
      <aside className="relative flex h-full w-[min(20rem,calc(100vw-2rem))] flex-col border-r border-ink-200 bg-white shadow-xl">
        <div className="flex h-16 items-center justify-between border-b border-ink-100 px-5">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-ink-900 text-white">
              <LayoutDashboard className="h-5 w-5" />
            </div>
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold text-ink-900">FreeDinnerAgent</div>
              <div className="truncate text-xs text-ink-500">{t("app.subtitle")}</div>
            </div>
          </div>
          <button
            aria-label={t("common.close")}
            className="rounded-md p-2 text-ink-500 hover:bg-ink-100"
            onClick={onClose}
            type="button"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <SidebarNav onNavigate={onClose} />
        <div className="mt-auto border-t border-ink-100 p-4">
          <div className="flex items-center gap-2 rounded-md bg-ink-50 px-3 py-2 text-xs text-ink-500">
            <Database className="h-4 w-4 text-mint-600" />
            <span>{t("app.backendReady")}</span>
          </div>
        </div>
      </aside>
    </div>
  );
}

function SidebarNav({ onNavigate }: { onNavigate?: () => void }) {
  const { t } = useTranslation();

  return (
    <nav className="space-y-1 overflow-y-auto px-3 py-4">
      {navItems.map((item) => {
        const Icon = item.icon;
        return (
          <NavLink
            className={({ isActive }) =>
              [
                "flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium transition",
                isActive
                  ? "bg-ink-900 text-white"
                  : "text-ink-600 hover:bg-ink-100 hover:text-ink-900"
              ].join(" ")
            }
            key={item.to}
            onClick={onNavigate}
            to={item.to}
          >
            <Icon className="h-4 w-4 shrink-0" />
            <span className="truncate">{t(item.key)}</span>
          </NavLink>
        );
      })}
    </nav>
  );
}
