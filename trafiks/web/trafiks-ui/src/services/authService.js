import api from "./api";

export const authService = {
  login: async (email, password) => {
    const response = await api.post("/auth/login", {
      Email: email,
      Password: password,
    });
    return response;
  },

  logout: async () => {
    await api.post("/auth/logout");
  },

  getCurrentUser: async () => {
    const response = await api.get("/users/data");
    // Response is wrapped in { message, data, success }
    return response.data || response;
  },
};
