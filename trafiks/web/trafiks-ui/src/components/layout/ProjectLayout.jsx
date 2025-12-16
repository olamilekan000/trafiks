import { Outlet, useParams } from "react-router-dom";
import Sidebar from "./Sidebar";
import { getProjectSidebarItems } from "../../utils/sidebarItems";
import "./ProjectLayout.css";

export default function ProjectLayout() {
  const { projectId } = useParams();
  const sidebarItems = getProjectSidebarItems(projectId);

  return (
    <div className="project-layout-container">
      <Sidebar items={sidebarItems} />
      <div className="project-layout-content">
        <Outlet />
      </div>
    </div>
  );
}
