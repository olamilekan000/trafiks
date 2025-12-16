import api from "./api";

export const metricsService = {
  getMetrics: async (projectId, filters = {}) => {
    const params = {};
    if (filters.start_time) {
      params.start_time = filters.start_time;
    }
    if (filters.end_time) {
      params.end_time = filters.end_time;
    }
    if (filters.group_by) {
      params.group_by = filters.group_by;
    }
    if (filters.top_paths_limit) {
      params.top_paths_limit = filters.top_paths_limit;
    }

    const response = await api.get(`/projects/${projectId}/metrics`, {
      params,
    });
    return response.data;
  },
};
