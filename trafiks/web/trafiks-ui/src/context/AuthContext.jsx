import { createContext, useContext, useState, useEffect, useRef } from "react";
import { authService } from "../services/authService";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const checkingAuth = useRef(false);

  useEffect(() => {
    // Prevent multiple simultaneous auth checks
    if (checkingAuth.current) {
      return;
    }

    // Check if user is already logged in by verifying session
    const checkAuth = async () => {
      checkingAuth.current = true;
      try {
        const response = await authService.getCurrentUser();
        // Handle wrapped response: { message, data, success }
        // axios interceptor already returns response.data, so response is { message, data, success }
        const userData = response.data || response;

        // Backend returns PascalCase: Email, FirstName, LastName, UID
        // Handle both camelCase and PascalCase field names
        const email = userData.Email || userData.email;
        const firstName =
          userData.FirstName || userData.first_name || userData.firstName;
        const lastName =
          userData.LastName || userData.last_name || userData.lastName;
        const uid = userData.UID || userData.uid;

        if (userData && email) {
          setUser({
            email,
            firstName,
            lastName,
            uid,
          });
        } else {
          setUser(null);
        }
      } catch (error) {
        // User is not authenticated, clear any stale state
        // Silently handle 401 errors - they're expected when not logged in
        setUser(null);
      } finally {
        setLoading(false);
        checkingAuth.current = false;
      }
    };

    checkAuth();
  }, []);

  const login = async (email, password) => {
    try {
      const response = await authService.login(email, password);
      // Fetch user profile after successful login
      try {
        const userResponse = await authService.getCurrentUser();
        const userData = userResponse.data || userResponse;

        // Backend returns PascalCase: Email, FirstName, LastName, UID
        // Handle both camelCase and PascalCase field names
        const userEmail = userData.Email || userData.email || email;
        const firstName =
          userData.FirstName || userData.first_name || userData.firstName;
        const lastName =
          userData.LastName || userData.last_name || userData.lastName;
        const uid = userData.UID || userData.uid;

        if (userData && userEmail) {
          setUser({
            email: userEmail,
            firstName,
            lastName,
            uid,
          });
        } else {
          // Fallback to email if profile fetch fails
          setUser({ email });
        }
      } catch (profileError) {
        // If profile fetch fails, use email as fallback
        setUser({ email });
      }
      return response;
    } catch (error) {
      throw error;
    }
  };

  const logout = async () => {
    try {
      // Clear user state immediately to prevent redirect loops
      setUser(null);
      setLoading(false);
      await authService.logout();
    } catch (error) {
      console.error("Logout error:", error);
      // Even if logout fails, clear user state
      setUser(null);
      setLoading(false);
    }
  };

  return (
    <AuthContext.Provider value={{ user, login, logout, loading }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
