import { Navigate, createBrowserRouter } from "react-router-dom";

import { AppShell } from "../components/layout/AppShell";
import { PlaceholderPage } from "../components/layout/PlaceholderPage";
import { LoadingState } from "../components/ui/LoadingState";
import { useCurrentUser } from "../features/auth/hooks";
import { LoginPage } from "../features/auth/pages/LoginPage";
import { RegisterPage } from "../features/auth/pages/RegisterPage";
import { ChannelsPage } from "../features/channels/pages/ChannelsPage";
import { ChatPage } from "../features/chat/pages/ChatPage";
import { KnowledgePage } from "../features/knowledge/pages/KnowledgePage";
import { MarketPage } from "../features/market/pages/MarketPage";
import { MemoryPage } from "../features/memory/pages/MemoryPage";
import { AgentConfigPage } from "../features/settings/pages/AgentConfigPage";
import { ProvidersPage } from "../features/settings/pages/ProvidersPage";
import { TasksPage } from "../features/tasks/pages/TasksPage";
import { ToolsPage } from "../features/tools/pages/ToolsPage";
import { getAccessToken } from "../lib/authToken";

function RequireAuth({ children }: { children: React.ReactNode }) {
  const hasToken = Boolean(getAccessToken());
  const { isError, isLoading } = useCurrentUser();

  if (!hasToken) {
    return <Navigate to="/login" replace />;
  }

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-ink-50">
        <LoadingState />
      </div>
    );
  }
  if (isError) {
    return <Navigate to="/login" replace />;
  }

  return children;
}

function PublicOnly({ children }: { children: React.ReactNode }) {
  if (getAccessToken()) {
    return <Navigate to="/app/chat" replace />;
  }
  return children;
}

export const router = createBrowserRouter([
  {
    path: "/",
    element: <Navigate to="/app/chat" replace />
  },
  {
    path: "/login",
    element: (
      <PublicOnly>
        <LoginPage />
      </PublicOnly>
    )
  },
  {
    path: "/register",
    element: (
      <PublicOnly>
        <RegisterPage />
      </PublicOnly>
    )
  },
  {
    path: "/app",
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="/app/chat" replace /> },
      { path: "chat", element: <ChatPage /> },
      {
        path: "chat/:conversationId",
        element: <ChatPage />
      },
      { path: "agent", element: <AgentConfigPage /> },
      { path: "providers", element: <ProvidersPage /> },
      { path: "memory", element: <MemoryPage /> },
      { path: "knowledge", element: <KnowledgePage /> },
      { path: "market", element: <MarketPage /> },
      { path: "tools", element: <ToolsPage /> },
      { path: "tasks", element: <TasksPage /> },
      { path: "channels", element: <ChannelsPage /> },
      {
        path: "channels/:connectionId",
        element: <ChannelsPage />
      },
      { path: "workspace", element: <PlaceholderPage pageKey="workspace" /> },
      { path: "logs", element: <PlaceholderPage pageKey="logs" /> }
    ]
  },
  {
    path: "*",
    element: <Navigate to="/app/chat" replace />
  }
]);
