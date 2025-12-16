import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  FiArrowLeft,
  FiUser,
  FiLock,
  FiKey,
  FiAlertOctagon,
} from "react-icons/fi";
import toast from "react-hot-toast";
import { useAuth } from "../context/AuthContext";
import { userService } from "../services/userService";
import Card from "../components/common/Card";
import Button from "../components/common/Button";
import Input from "../components/common/Input";
import Loading from "../components/common/Loading";
import IconContainer from "../components/common/IconContainer";
import PageHeader from "../components/common/PageHeader";
import "../styles/utilities.css";
import "./UserProfile.css";

export default function UserProfile() {
  const navigate = useNavigate();
  const { user: authUser } = useAuth();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [changingPassword, setChangingPassword] = useState(false);

  // Profile state
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [profileError, setProfileError] = useState("");
  const [profileSuccess, setProfileSuccess] = useState("");

  // Password state
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");
  const [passwordSuccess, setPasswordSuccess] = useState("");

  useEffect(() => {
    loadProfile();
  }, []);

  const loadProfile = async () => {
    try {
      setLoading(true);
      const response = await userService.getProfile();
      const userData = response.data || response;

      // Handle both PascalCase and camelCase
      setEmail(userData.Email || userData.email || authUser?.email || "");
      setFirstName(
        userData.FirstName || userData.first_name || userData.firstName || ""
      );
      setLastName(
        userData.LastName || userData.last_name || userData.lastName || ""
      );
    } catch (error) {
      console.error("Failed to load profile:", error);
      toast.error("Failed to load profile. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateProfile = async (e) => {
    e.preventDefault();
    setProfileError("");
    setProfileSuccess("");

    if (!firstName.trim() && !lastName.trim()) {
      setProfileError("At least first name or last name is required");
      return;
    }

    try {
      setSaving(true);
      await userService.updateProfile(firstName.trim(), lastName.trim());
      toast.success("Profile updated successfully!");
      setProfileSuccess("Profile updated successfully!");

      // Refresh profile data
      await loadProfile();
    } catch (error) {
      const errorMessage =
        error.response?.data?.message ||
        "Failed to update profile. Please try again.";
      setProfileError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const handleChangePassword = async (e) => {
    e.preventDefault();
    setPasswordError("");
    setPasswordSuccess("");

    if (!oldPassword || !newPassword || !confirmPassword) {
      setPasswordError("All password fields are required");
      return;
    }

    if (newPassword.length < 8) {
      setPasswordError("New password must be at least 8 characters");
      return;
    }

    if (newPassword !== confirmPassword) {
      setPasswordError("New passwords do not match");
      return;
    }

    if (oldPassword === newPassword) {
      setPasswordError("New password must be different from old password");
      return;
    }

    try {
      setChangingPassword(true);
      await userService.changePassword(oldPassword, newPassword);
      setPasswordSuccess("Password changed successfully!");

      // Clear password fields
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (error) {
      const errorMessage =
        error.response?.data?.message ||
        "Failed to change password. Please try again.";
      setPasswordError(errorMessage);
    } finally {
      setChangingPassword(false);
    }
  };

  if (loading) {
    return (
      <div className="user-profile-page page-fade-in">
        <div className="loading-full">
          <Loading size="large" />
        </div>
      </div>
    );
  }

  return (
    <div className="user-profile-page page-fade-in">
      <PageHeader
        title={
          <>
            <Button
              variant="ghost"
              onClick={() => navigate("/dashboard/projects")}
              style={{ marginRight: "16px" }}
            >
              <FiArrowLeft size={20} />
            </Button>
            My Account
          </>
        }
      />

      <div className="user-profile-content">
        {/* Profile Information Card */}
        <Card className="profile-card">
          <div className="profile-card-header">
            <IconContainer size="medium">
              <FiUser size={20} />
            </IconContainer>
            <h2 className="section-title">Profile Information</h2>
          </div>
          <form onSubmit={handleUpdateProfile} className="form-section">
            <div className="profile-form-fields">
              <Input
                label="Email"
                type="email"
                value={email}
                disabled
                className="profile-input"
              />
              <div className="profile-name-fields">
                <Input
                  label="First Name"
                  type="text"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  placeholder="Enter your first name"
                  className="profile-input"
                />
                <Input
                  label="Last Name"
                  type="text"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  placeholder="Enter your last name"
                  className="profile-input"
                />
              </div>
            </div>
            {profileError && (
              <div className="profile-error">{profileError}</div>
            )}
            {profileSuccess && (
              <div className="profile-success">{profileSuccess}</div>
            )}
            <div
              className="form-actions"
              style={{ justifyContent: "flex-start" }}
            >
              <Button
                type="submit"
                variant="primary"
                disabled={saving || (!firstName.trim() && !lastName.trim())}
              >
                {saving ? "Saving..." : "Save Changes"}
              </Button>
            </div>
          </form>
        </Card>

        {/* Change Password Card */}
        <Card className="profile-card">
          <div className="profile-card-header">
            <IconContainer size="medium">
              <FiLock size={20} />
            </IconContainer>
            <h2 className="section-title">Change Passwordzzz</h2>
          </div>
          <form onSubmit={handleChangePassword} className="form-section">
            <div className="profile-form-fields">
              <Input
                label="Current Password"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                placeholder="Enter your current password"
                className="profile-input"
                required
              />
              <Input
                label="New Password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="Enter your new password (min. 8 characters)"
                className="profile-input"
                required
              />
              <Input
                label="Confirm New Password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Confirm your new password"
                className="profile-input"
                required
              />
            </div>
            {passwordError && (
              <div className="profile-error">{passwordError}</div>
            )}
            {passwordSuccess && (
              <div className="profile-success">{passwordSuccess}</div>
            )}
            <div
              className="form-actions"
              style={{ justifyContent: "flex-start" }}
            >
              <Button
                type="submit"
                variant="primary"
                disabled={
                  changingPassword ||
                  !oldPassword ||
                  !newPassword ||
                  !confirmPassword
                }
              >
                {changingPassword ? "Changing..." : "Change Password"}
              </Button>
            </div>
          </form>
        </Card>

        {/* API Keys Card */}
        <Card className="profile-card profile-card-full-width">
          <div className="profile-card-header">
            <IconContainer size="medium">
              <FiKey size={20} />
            </IconContainer>
            <h2 className="section-title">API Keys</h2>
          </div>
          <div className="profile-card-content">
            <p className="profile-card-description">
              Manage your API keys for programmatic access to your projects.
              Generate, view, and revoke API keys to secure your integrations.
            </p>
            <Button
              variant="primary"
              onClick={() => navigate("/dashboard/api-keys")}
              className="profile-card-button"
            >
              Manage API Keys
            </Button>
          </div>
        </Card>

        {/* Webhooks Card */}
        <Card className="profile-card profile-card-full-width">
          <div className="profile-card-header">
            <IconContainer size="medium">
              <FiAlertOctagon size={20} />
            </IconContainer>
            <h2 className="section-title">Webhooks</h2>
          </div>
          <div className="profile-card-content">
            <p className="profile-card-description">
              Configure webhooks to receive real-time notifications about events
              across all your projects. Monitor project activities, API key
              changes, and system events.
            </p>
            <Button
              variant="primary"
              onClick={() => navigate("/dashboard/webhooks")}
              className="profile-card-button"
            >
              Manage Webhooks
            </Button>
          </div>
        </Card>
      </div>
    </div>
  );
}
