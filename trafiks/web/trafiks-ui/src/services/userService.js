import api from "./api";

export const userService = {
  getProfile: async () => {
    const response = await api.get("/users/data");
    return response.data;
  },

  updateProfile: async (firstName, lastName) => {
    const response = await api.put("/users/data", {
      FirstName: firstName,
      LastName: lastName,
    });
    return response.data;
  },

  changePassword: async (oldPassword, newPassword) => {
    const response = await api.post("/users/change-password", {
      OldPassword: oldPassword,
      NewPassword: newPassword,
    });
    return response.data;
  },
};
