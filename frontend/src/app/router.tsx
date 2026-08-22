import { Navigate, createBrowserRouter } from "react-router-dom";

import { AppShell } from "../components/layout/AppShell";
import { PlaceholderPage } from "../components/layout/PlaceholderPage";
import { LoginPage } from "../features/auth/pages/LoginPage";
import { RegisterPage } from "../features/auth/pages/RegisterPage";
import { getAccessToken } from "../lib/authToken";

function RequireAuth({ children }: { children: React.ReactNode }) {
  if (!getAccessToken()) {
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
      { path: "chat", element: <PlaceholderPage pageKey="chat" /> },
      {
        path: "chat/:conversationId",
        element: <PlaceholderPage pageKey="chatDetail" />
      },
      { path: "agent", element: <PlaceholderPage pageKey="agent" /> },
      { path: "providers", element: <PlaceholderPage pageKey="providers" /> },
      { path: "memory", element: <PlaceholderPage pageKey="memory" /> },
      { path: "knowledge", element: <PlaceholderPage pageKey="knowledge" /> },
      { path: "market", element: <PlaceholderPage pageKey="market" /> },
      { path: "tools", element: <PlaceholderPage pageKey="tools" /> },
      { path: "tasks", element: <PlaceholderPage pageKey="tasks" /> },
      { path: "channels", element: <PlaceholderPage pageKey="channels" /> },
      {
        path: "channels/:connectionId",
        element: <PlaceholderPage pageKey="channelDetail" />
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
