import { useState, useEffect } from "react";
import Input from "./Input";
import Button from "./Button";
import "./ProjectForm.css";

export default function ProjectForm({
  initialName = "",
  initialDescription = "",
  onSubmit,
  onCancel,
  submitLabel = "Create Project",
  cancelLabel = "Cancel",
  loading = false,
  error: externalError = null,
}) {
  const [name, setName] = useState(initialName);
  const [description, setDescription] = useState(initialDescription);
  const [nameError, setNameError] = useState("");
  const [error, setError] = useState(externalError || "");

  // Update error when external error changes
  useEffect(() => {
    if (externalError !== null) {
      setError(externalError);
    }
  }, [externalError]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setNameError("");

    if (!name.trim()) {
      setNameError("Project name is required");
      return;
    }

    try {
      await onSubmit({
        name: name.trim(),
        description: description.trim(),
      });
    } catch (err) {
      const errorMessage =
        err.response?.data?.message ||
        err.message ||
        "An error occurred. Please try again.";
      setError(errorMessage);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="project-form">
      <Input
        label="Project Name"
        type="text"
        value={name}
        onChange={(e) => {
          setName(e.target.value);
          if (nameError) setNameError("");
        }}
        placeholder="Enter project name"
        required
        error={nameError}
        disabled={loading}
      />

      <Input
        label="Description"
        type="text"
        value={description}
        onChange={(e) => setDescription(e.target.value)}
        placeholder="Enter project description (optional)"
        disabled={loading}
      />

      {error && <div className="project-form-error">{error}</div>}

      <div className="project-form-actions">
        {onCancel && (
          <Button
            type="button"
            variant="ghost"
            onClick={onCancel}
            disabled={loading}
          >
            {cancelLabel}
          </Button>
        )}
        <Button
          type="submit"
          variant="primary"
          disabled={loading || !name.trim()}
        >
          {loading ? "Saving..." : submitLabel}
        </Button>
      </div>
    </form>
  );
}
