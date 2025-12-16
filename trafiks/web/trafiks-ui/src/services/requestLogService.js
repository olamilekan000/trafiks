import api from "./api";

export const requestLogService = {
  list: async (projectId, page = 1, limit = 20, filters = {}) => {
    const params = { page, limit };
    if (filters.method) params.method = filters.method;
    if (filters.status_code) params.status_code = filters.status_code;
    if (filters.start_time) params.start_time = filters.start_time;
    if (filters.end_time) params.end_time = filters.end_time;
    if (filters.cache_hit !== undefined) params.cache_hit = filters.cache_hit;
    if (filters.path) params.path = filters.path;

    const response = await api.get(`/projects/${projectId}/request-logs`, {
      params,
    });
    return response.data;
  },

  get: async (projectId, logId) => {
    const response = await api.get(
      `/projects/${projectId}/request-logs/${logId}`
    );
    return response.data;
  },

  replay: async (projectId, logId) => {
    const response = await api.post(
      `/projects/${projectId}/request-logs/${logId}/replay`
    );
    return response.data;
  },
};
