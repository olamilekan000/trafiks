import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";
import "./Header.css";

export default function Header() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [showDropdown, setShowDropdown] = useState(false);
  const dropdownRef = useRef(null);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setShowDropdown(false);
      }
    };

    if (showDropdown) {
      document.addEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showDropdown]);

  const handleLogout = async () => {
    try {
      await logout();
      navigate("/dashboard/login");
    } catch (error) {
      console.error("Logout error:", error);
    }
  };

  const getUserInitials = () => {
    if (user?.firstName && user?.lastName) {
      return `${user.firstName[0]}${user.lastName[0]}`.toUpperCase();
    }
    if (user?.email) {
      return user.email[0].toUpperCase();
    }
    return "U";
  };

  const getUserDisplayName = () => {
    if (user?.firstName && user?.lastName) {
      return `${user.firstName} ${user.lastName}`;
    }
    if (user?.email) {
      return user.email;
    }
    return "User";
  };

  return (
    <header className="header">
      <div className="header-left">
        <div className="logo" onClick={() => navigate("/dashboard/projects")}>
          <div className="logo-icon">T</div>
          <span className="logo-text">
            <span className="logo-text-fancy">Trafiks</span>
          </span>
        </div>
      </div>
      <div className="header-right">
        <button className="header-link">
          <span>Go to docs</span>
        </button>
        {user && (
          <div className="header-user-container" ref={dropdownRef}>
            <div
              className="header-user"
              onClick={() => setShowDropdown(!showDropdown)}
            >
              <div className="user-avatar">{getUserInitials()}</div>
              <span className="user-name">{getUserDisplayName()}</span>
              <span className="dropdown-arrow">▼</span>
            </div>
            {showDropdown && (
              <div className="user-dropdown">
                <div className="dropdown-item dropdown-header">
                  <div className="dropdown-name">{getUserDisplayName()}</div>
                  <div className="dropdown-email">{user.email}</div>
                </div>
                <div className="dropdown-divider"></div>
                <button
                  className="dropdown-item"
                  onClick={() => {
                    setShowDropdown(false);
                    navigate("/dashboard/profile");
                  }}
                >
                  My account
                </button>
                <button
                  className="dropdown-item dropdown-item-danger"
                  onClick={handleLogout}
                >
                  Logout
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </header>
  );
}
