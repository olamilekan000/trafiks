import {
  FiServer,
  FiBarChart2,
  FiFileText,
  FiSettings,
  FiAlertOctagon,
} from "react-icons/fi";

// Shared sidebar configuration for project pages
export function getProjectSidebarItems(projectId) {
  return [
    {
      path: `/dashboard/projects/${projectId}/metrics`,
      label: "Dashboard",
      icon: FiBarChart2,
    },
    {
      path: `/dashboard/projects/${projectId}/service`,
      label: "Service",
      icon: FiServer,
    },
    {
      path: `/dashboard/projects/${projectId}/logs`,
      label: "Request Logs",
      icon: FiFileText,
    },
    {
      path: `/dashboard/projects/${projectId}/settings`,
      label: "Settings",
      icon: FiSettings,
    },
  ];
}
