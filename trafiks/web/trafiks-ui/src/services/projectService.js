import api from "./api";

export const projectService = {
  list: async (page = 1, limit = 20) => {
    const response = await api.get("/projects", {
      params: { page, limit },
    });
    return response.data;
  },

  get: async (projectId) => {
    const response = await api.get(`/projects/${projectId}`);
    return response.data;
  },

  create: async (name, description = "") => {
    const response = await api.post("/projects", {
      Name: name,
      Description: description,
    });
    return response.data;
  },

  update: async (projectId, updates) => {
    const response = await api.put(`/projects/${projectId}`, updates);
    return response.data;
  },

  toggleActive: async (projectId, isActive) => {
    const response = await api.put(`/projects/${projectId}`, {
      IsActive: isActive,
    });
    return response.data;
  },

  delete: async (projectId) => {
    const response = await api.delete(`/projects/${projectId}`);
    return response.data;
  },
};
