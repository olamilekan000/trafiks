import api from "./api";

export const webhookService = {
  create: async (webhookData) => {
    const response = await api.post(`/webhooks`, webhookData);
    return response.data;
  },

  list: async () => {
    const response = await api.get(`/webhooks`);
    return response.data;
  },

  get: async (webhookId) => {
    const response = await api.get(`/webhooks/${webhookId}`);
    return response.data;
  },

  update: async (webhookId, webhookData) => {
    const response = await api.put(`/webhooks/${webhookId}`, webhookData);
    return response.data;
  },

  delete: async (webhookId) => {
    const response = await api.delete(`/webhooks/${webhookId}`);
    return response.data;
  },

  // Deliveries
  listDeliveries: async (webhookId = null, params = {}) => {
    let url = `/webhooks/deliveries`;
    if (webhookId) {
      url = `/webhooks/${webhookId}/deliveries`;
    }
    const response = await api.get(url, { params });
    return response.data;
  },

  getDelivery: async (deliveryId) => {
    const response = await api.get(`/webhooks/deliveries/${deliveryId}`);
    return response.data;
  },
};
