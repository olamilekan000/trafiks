import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import toast from "react-hot-toast";
import { useAuth } from "../context/AuthContext";
import Button from "../components/common/Button";
import Input from "../components/common/Input";
import Loading from "../components/common/Loading";
import "./Login.css";

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [emailError, setEmailError] = useState("");
  const [loading, setLoading] = useState(false);
  const { login, user, loading: authLoading } = useAuth();
  const navigate = useNavigate();

  // Redirect if already logged in
  useEffect(() => {
    // Only redirect if auth check is complete and user exists
    // Don't redirect during loading to prevent loops
    if (!authLoading && user) {
      navigate("/dashboard/projects", { replace: true });
    }
  }, [user, authLoading, navigate]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setEmailError("");

    if (!email.trim()) {
      setEmailError("Please enter your email");
      return;
    }

    setLoading(true);

    try {
      await login(email, password);
      toast.success("Welcome back! Login successful.");
      navigate("/dashboard/projects", { replace: true });
    } catch (err) {
      const errorMessage =
        err.response?.data?.message ||
        "Login failed. Please check your credentials.";
      setError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // Show loading while checking auth status
  if (authLoading) {
    return (
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100vh",
        }}
      >
        <Loading size="large" />
      </div>
    );
  }

  // Don't render login form if user is already logged in (will redirect)
  if (user) {
    return null;
  }

  return (
    <div className="login-container">
      <div className="login-left">
        <div className="login-testimonial">
          <p className="testimonial-text">
            "We appreciate that they handle all the complexity of reverse proxy
            and caching for us, letting us focus on our core business."
          </p>
          <div className="testimonial-author">
            <div className="author-name">Rocky Oyeniran</div>
            <div className="author-title">CTO at Emergex AI</div>
          </div>
        </div>
      </div>
      <div className="login-right">
        <div className="login-card">
          <div className="login-logo">
            <div className="logo-icon-large">T</div>
            <span className="logo-text-large">
              <span className="logo-text-fancy">Trafiks</span>
            </span>
          </div>
          <form onSubmit={handleSubmit} className="login-form">
            <Input
              label="Email"
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (emailError) setEmailError("");
              }}
              placeholder="admin@trafiks.local"
              required
              error={emailError}
            />
            <Input
              label="Password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter your password"
              required
            />
            {error && <div className="login-error">{error}</div>}
            <Button
              type="submit"
              variant="primary"
              size="large"
              disabled={loading}
              className="login-button"
            >
              {loading ? "Logging in..." : "Login"}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
