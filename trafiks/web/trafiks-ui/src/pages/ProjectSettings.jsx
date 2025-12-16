import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { FiArrowLeft, FiX, FiAlertTriangle } from "react-icons/fi";
import toast from "react-hot-toast";
import { projectService } from "../services/projectService";
import {
  Card,
  Button,
  Input,
  Loading,
  Badge,
  Modal,
  PageHeader,
} from "../components/common";
import "./ProjectSettings.css";

export default function ProjectSettings() {
  const { projectId } = useParams();
  const navigate = useNavigate();

  const [project, setProject] = useState(null);
  const [loading, setLoading] = useState(true);
  const [description, setDescription] = useState("");
  const [isActive, setIsActive] = useState(true);
  const [saving, setSaving] = useState(false);
  const [togglingStatus, setTogglingStatus] = useState(false);
  const [showDeactivateConfirm, setShowDeactivateConfirm] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  useEffect(() => {
    loadProject();
  }, [projectId]);

  const loadProject = async () => {
    try {
      setLoading(true);
      const response = await projectService.get(projectId);
      const projectData = response.data || response;
      setProject(projectData);
      setDescription(projectData.description || "");
      setIsActive(projectData.is_active !== false); // Default to true if not set
    } catch (error) {
      console.error("Failed to load project:", error);
      toast.error("Failed to load project. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      await projectService.update(projectId, {
        Description: description,
      });
      toast.success("Project updated successfully!");
      await loadProject();
    } catch (error) {
      console.error("Failed to update project:", error);
      const errorMessage =
        error.response?.data?.message ||
        "Failed to update project. Please try again.";
      toast.error(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async () => {
    // If deactivating, show confirmation modal first
    if (isActive) {
      setShowDeactivateConfirm(true);
      return;
    }

    // If activating, proceed directly
    await performToggleActive();
  };

  const performToggleActive = async () => {
    try {
      setTogglingStatus(true);
      const newStatus = !isActive;
      await projectService.toggleActive(projectId, newStatus);
      setIsActive(newStatus);
      setShowDeactivateConfirm(false);
      toast.success(
        newStatus
          ? "Project activated successfully!"
          : "Project deactivated successfully!"
      );
    } catch (error) {
      console.error("Failed to toggle project status:", error);
      const errorMessage =
        error.response?.data?.message ||
        "Failed to update project status. Please try again.";
      toast.error(errorMessage);
    } finally {
      setTogglingStatus(false);
    }
  };

  const handleDelete = async () => {
    try {
      setDeleting(true);
      await projectService.delete(projectId);
      toast.success("Project deleted successfully!");
      navigate("/dashboard/projects");
    } catch (error) {
      console.error("Failed to delete project:", error);
      const errorMessage =
        error.response?.data?.message ||
        "Failed to delete project. Please try again.";
      toast.error(errorMessage);
      setDeleting(false);
      setShowDeleteConfirm(false);
    }
  };

  if (loading) {
    return (
      <div className="project-settings-page page-fade-in">
        <div className="loading-full">
          <Loading size="large" />
        </div>
      </div>
    );
  }

  if (!project) {
    return (
      <div className="project-settings-page page-fade-in">
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            padding: "48px",
            minHeight: "400px",
            color: "var(--error-red)",
          }}
        >
          Project not found
        </div>
      </div>
    );
  }

  return (
    <div className="project-settings-page page-fade-in">
      <PageHeader
        title={
          <>
            <Button
              variant="ghost"
              onClick={() => navigate("/dashboard/projects")}
              style={{ marginRight: "16px" }}
            >
              <FiArrowLeft size={18} />
            </Button>
            Project Configurations
          </>
        }
      />

      <Card>
        <div className="form-section">
          <Input
            label="Project name"
            value={project.name}
            disabled
            style={{ maxWidth: "600px" }}
          />
          <Input
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Enter project description"
            style={{ maxWidth: "600px" }}
          />
          <div className="form-actions" style={{ maxWidth: "600px" }}>
            <Button
              variant="primary"
              onClick={handleSave}
              disabled={saving}
              style={{ maxWidth: "200px" }}
            >
              {saving ? "Saving..." : "Save Changes"}
            </Button>
          </div>
        </div>
      </Card>

      <Card className="project-settings-card">
        <div className="settings-section">
          <div className="status-section">
            <div className="status-header">
              <div>
                <h3 className="status-title">Project Status</h3>
                <p className="status-description">
                  Activate or deactivate this project. Inactive projects will
                  not process proxy requests.
                </p>
              </div>
              <div className="status-indicator-container">
                {isActive && <div className="status-dot pulse"></div>}
                <Badge variant={isActive ? "success" : "warning"}>
                  {isActive ? "ACTIVE" : "INACTIVE"}
                </Badge>
              </div>
            </div>
            <Button
              variant={isActive ? "secondary" : "primary"}
              onClick={handleToggleActive}
              disabled={togglingStatus}
              style={{ maxWidth: "200px" }}
            >
              {togglingStatus
                ? "Updating..."
                : isActive
                ? "Deactivate Project"
                : "Activate Project"}
            </Button>
          </div>
        </div>
      </Card>

      <Card className="project-settings-card danger-zone">
        <div className="danger-zone-content">
          <h2 className="danger-zone-title">Danger zone</h2>
          <p className="danger-zone-description">
            Deleting this project will delete all of its data including
            services, request logs, and configurations. Are you sure you want to
            delete this project?
          </p>
          {!showDeleteConfirm ? (
            <Button
              variant="danger"
              onClick={() => setShowDeleteConfirm(true)}
              className="danger-button"
            >
              🗑️ Delete Project
            </Button>
          ) : (
            <div className="delete-confirm">
              <p className="delete-confirm-text">
                Are you absolutely sure? This action cannot be undone.
              </p>
              <div className="delete-confirm-buttons">
                <Button
                  variant="danger"
                  onClick={handleDelete}
                  disabled={deleting}
                >
                  {deleting ? "Deleting..." : "Yes, Delete Project"}
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => setShowDeleteConfirm(false)}
                  disabled={deleting}
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}
        </div>
      </Card>

      {/* Deactivate Confirmation Modal */}
      <Modal
        isOpen={showDeactivateConfirm}
        onClose={() => setShowDeactivateConfirm(false)}
        title={
          <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
            <div
              style={{
                width: "40px",
                height: "40px",
                borderRadius: "50%",
                backgroundColor: "#fef3c7",
                color: "var(--warning)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                flexShrink: 0,
              }}
            >
              <FiAlertTriangle size={24} />
            </div>
            Deactivate Project
          </div>
        }
        size="small"
      >
        <p
          style={{
            fontSize: "18px",
            fontWeight: 600,
            color: "var(--text-primary)",
            margin: "0 0 12px 0",
          }}
        >
          Are you sure you want to deactivate this project?
        </p>
        <p
          style={{
            fontSize: "14px",
            color: "var(--text-secondary)",
            lineHeight: 1.6,
            margin: "0 0 24px 0",
          }}
        >
          Deactivating this project will prevent it from processing any proxy
          requests. You can reactivate it at any time from the project settings.
        </p>
        <div className="form-actions">
          <Button
            variant="secondary"
            onClick={() => setShowDeactivateConfirm(false)}
            disabled={togglingStatus}
          >
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={performToggleActive}
            disabled={togglingStatus}
          >
            {togglingStatus ? "Deactivating..." : "Yes, Deactivate"}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
