import api from "./api";

export const serviceService = {
  create: async (projectId, serviceData) => {
    const response = await api.post(
      `/projects/${projectId}/services`,
      serviceData
    );
    return response.data;
  },

  get: async (projectId, serviceId) => {
    const response = await api.get(
      `/projects/${projectId}/services/${serviceId}`
    );
    return response.data;
  },

  getByProject: async (projectId) => {
    const response = await api.get(`/projects/${projectId}/service`);
    return response.data;
  },

  update: async (projectId, serviceId, serviceData) => {
    const response = await api.put(
      `/projects/${projectId}/services/${serviceId}`,
      serviceData
    );
    return response.data;
  },
};
