import api from "./api";

export const apiKeyService = {
  generate: async (name, expiresInDays = 0) => {
    const response = await api.post("/api-keys", {
      Name: name,
      ExpiresInDays: expiresInDays || 0,
    });
    return response.data;
  },

  list: async () => {
    const response = await api.get("/api-keys");
    return response.data;
  },

  revoke: async (keyId) => {
    const response = await api.delete(`/api-keys/${keyId}`);
    return response.data;
  },
};
