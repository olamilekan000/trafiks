import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { FiFolder, FiPlus, FiLock } from "react-icons/fi";
import toast from "react-hot-toast";
import { projectService } from "../services/projectService";
import {
  Card,
  Button,
  Badge,
  Loading,
  PageHeader,
  EmptyState,
} from "../components/common";
import "./Projects.css";

export default function Projects() {
  const [projects, setProjects] = useState([]);
  const [projectServices, setProjectServices] = useState({}); // Map of projectId -> service
  const [loading, setLoading] = useState(true);
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 20,
    total: 0,
  });
  const navigate = useNavigate();

  useEffect(() => {
    loadProjects();
  }, [pagination.page]);

  const loadProjects = async () => {
    try {
      setLoading(true);
      const response = await projectService.list(
        pagination.page,
        pagination.limit
      );
      const projectsList = response.projects || [];
      setProjects(projectsList);
      setPagination((prev) => ({
        ...prev,
        total: response.pagination?.total || 0,
      }));

      // Extract proxy URLs and schemes from backend response
      const servicesMap = {};
      projectsList.forEach((project) => {
        if (project.proxy_url) {
          servicesMap[project.uid] = {
            proxy_url: project.proxy_url,
            scheme: project.scheme || "http",
          };
        }
      });
      setProjectServices(servicesMap);
    } catch (error) {
      console.error("Failed to load projects:", error);
      toast.error("Failed to load projects. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleCreateProject = () => {
    navigate("/dashboard/projects/new");
  };

  const handleProjectClick = (projectId) => {
    navigate(`/dashboard/projects/${projectId}`);
  };

  if (loading) {
    return (
      <div className="projects-page page-fade-in">
        <div className="loading-full">
          <Loading size="large" />
        </div>
      </div>
    );
  }

  return (
    <div className="projects-page page-fade-in">
      <PageHeader
        title="Projects"
        subtitle="All your project's summary at a glance"
      />

      {projects.length === 0 ? (
        <EmptyState
          icon={FiFolder}
          title="No projects yet"
          description="Create your first project to get started with Trafiks."
          actionLabel="Create Project"
          onAction={handleCreateProject}
        />
      ) : (
        <div className="projects-grid">
          <Card
            className="project-card project-card-create"
            onClick={handleCreateProject}
            interactive
          >
            <div className="project-card-create-content">
              <div className="project-card-icon">
                <FiPlus size={24} />
              </div>
              <div className="project-card-name">Create new project</div>
            </div>
          </Card>

          {projects.map((project) => {
            const isActive = project.is_active !== false; // Default to true if not set
            return (
              <Card
                key={project.uid}
                className="project-card"
                onClick={() => handleProjectClick(project.uid)}
                interactive
              >
                <div className="project-card-content">
                  <div className="project-card-header">
                    <div className="project-card-icon-small">
                      <FiFolder size={20} />
                    </div>
                    <div className="project-card-status-container">
                      {isActive && <div className="status-dot pulse"></div>}
                      <Badge variant={isActive ? "success" : "warning"}>
                        {isActive ? "ACTIVE" : "INACTIVE"}
                      </Badge>
                    </div>
                  </div>
                  <div className="project-card-name">{project.name}</div>
                  {project.description && (
                    <div className="project-card-description">
                      {project.description}
                    </div>
                  )}
                  {projectServices[project.uid]?.proxy_url && (
                    <div className="project-card-service-url">
                      <span className="service-url-label">Service URL:</span>
                      <div
                        style={{
                          display: "flex",
                          alignItems: "center",
                          gap: "8px",
                          maxWidth: "100%",
                        }}
                      >
                        {projectServices[project.uid].scheme === "https" && (
                          <FiLock
                            size={14}
                            style={{
                              color: "var(--success-green)",
                              flexShrink: 0,
                            }}
                            title="HTTPS Enabled"
                          />
                        )}
                        <code
                          className="code-text"
                          style={{
                            maxWidth: "200px",
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                          }}
                        >
                          {`${projectServices[project.uid].scheme}://${
                            projectServices[project.uid].proxy_url
                          }`}
                        </code>
                      </div>
                    </div>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
