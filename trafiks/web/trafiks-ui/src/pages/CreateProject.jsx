import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { FiArrowLeft } from "react-icons/fi";
import toast from "react-hot-toast";
import { projectService } from "../services/projectService";
import Card from "../components/common/Card";
import Button from "../components/common/Button";
import ProjectForm from "../components/common/ProjectForm";
import PageHeader from "../components/common/PageHeader";
import "../styles/utilities.css";
import "./CreateProject.css";

export default function CreateProject() {
  const navigate = useNavigate();
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async ({ name, description }) => {
    setError("");
    setLoading(true);

    try {
      const response = await projectService.create(name, description);
      // Response structure: { message, data: { uid, name, ... }, success }
      const projectData = response.data || response;
      toast.success("Project created successfully!");
      // Navigate to the service configuration page for the newly created project
      navigate(`/dashboard/projects/${projectData.uid}/service`, {
        replace: true,
      });
    } catch (err) {
      const errorMessage =
        err.response?.data?.message ||
        "Failed to create project. Please try again.";
      setError(errorMessage);
      toast.error(errorMessage);
      throw err; // Re-throw so ProjectForm can handle it
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    navigate("/dashboard/projects");
  };

  return (
    <div className="create-project-page page-fade-in">
      <PageHeader
        title={
          <>
            <Button
              variant="ghost"
              onClick={() => navigate("/dashboard/projects")}
              style={{ marginRight: "16px" }}
            >
              <FiArrowLeft size={20} />
            </Button>
            Create New Project
          </>
        }
      />

      <Card className="create-project-card">
        <ProjectForm
          onSubmit={handleSubmit}
          onCancel={handleCancel}
          loading={loading}
          error={error}
        />
      </Card>
    </div>
  );
}
