import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Toaster } from "react-hot-toast";
import { AuthProvider } from "./context/AuthContext";
import ProtectedRoute from "./components/ProtectedRoute";
import MainLayout from "./components/layout/MainLayout";
import Login from "./pages/Login";
import Projects from "./pages/Projects";
import CreateProject from "./pages/CreateProject";
import ProjectSettings from "./pages/ProjectSettings";
import ServiceConfig from "./pages/ServiceConfig";
import Metrics from "./pages/Metrics";
import RequestLogs from "./pages/RequestLogs";
import UserProfile from "./pages/UserProfile";
import APIKeys from "./pages/APIKeys";
import Webhooks from "./pages/Webhooks";
import ProjectLayout from "./components/layout/ProjectLayout";
import "./App.css";

function App() {
  return (
    <AuthProvider>
      <Toaster
        position="bottom-center"
        toastOptions={{
          duration: 4000,
          style: {
            background: "#fff",
            color: "#1f2937",
            borderRadius: "8px",
            boxShadow:
              "0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)",
            padding: "16px",
            fontSize: "14px",
          },
          success: {
            iconTheme: {
              primary: "#10b981",
              secondary: "#fff",
            },
          },
          error: {
            iconTheme: {
              primary: "#ef4444",
              secondary: "#fff",
            },
          },
        }}
      />
      <BrowserRouter>
        <Routes>
          <Route path="/dashboard/login" element={<Login />} />
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <MainLayout />
              </ProtectedRoute>
            }
          >
            <Route
              index
              element={<Navigate to="/dashboard/projects" replace />}
            />
            <Route path="profile" element={<UserProfile />} />
            <Route path="api-keys" element={<APIKeys />} />
            <Route path="webhooks" element={<Webhooks />} />
            <Route path="projects" element={<Projects />} />
            <Route path="projects/new" element={<CreateProject />} />
            <Route path="projects/:projectId" element={<ProjectLayout />}>
              <Route index element={<Navigate to="metrics" replace />} />
              <Route path="metrics" element={<Metrics />} />
              <Route path="settings" element={<ProjectSettings />} />
              <Route path="service" element={<ServiceConfig />} />
              <Route path="logs" element={<RequestLogs />} />
            </Route>
          </Route>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}

export default App;
