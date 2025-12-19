import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  FiArrowLeft,
  FiKey,
  FiPlus,
  FiTrash2,
  FiCopy,
  FiCheck,
  FiX,
} from "react-icons/fi";
import toast from "react-hot-toast";
import { apiKeyService } from "../services/apiKeyService";
import { copyToClipboard } from "../utils/clipboard";
import {
  Card,
  Button,
  Input,
  Badge,
  Loading,
  Modal,
  Table,
  EmptyState,
  PageHeader,
} from "../components/common";
import "./APIKeys.css";

export default function APIKeys() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [apiKeys, setApiKeys] = useState([]);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [expiresInDays, setExpiresInDays] = useState("");
  const [newKey, setNewKey] = useState(null);
  const [error, setError] = useState("");
  const [copiedKeyId, setCopiedKeyId] = useState(null);

  useEffect(() => {
    loadAPIKeys();
  }, []);

  const loadAPIKeys = async () => {
    try {
      setLoading(true);
      const response = await apiKeyService.list();
      const keys = response.data?.api_keys || response?.api_keys || [];
      setApiKeys(keys);
    } catch (error) {
      console.error("Failed to load API keys:", error);
      toast.error("Failed to load API keys. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    setError("");

    if (!newKeyName.trim()) {
      setError("Key name is required");
      return;
    }

    try {
      setCreating(true);
      const response = await apiKeyService.generate(
        newKeyName.trim(),
        expiresInDays ? parseInt(expiresInDays) : 0
      );
      const keyData = response.data || response;
      setNewKey(keyData);
      setShowCreateForm(false);
      setNewKeyName("");
      setExpiresInDays("");
      toast.success("API key generated successfully!");
      await loadAPIKeys();
    } catch (error) {
      const errorMessage =
        error.response?.data?.message ||
        "Failed to create API key. Please try again.";
      setError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (keyId) => {
    if (
      !window.confirm(
        "Are you sure you want to revoke this API key? This action cannot be undone."
      )
    ) {
      return;
    }

    try {
      await apiKeyService.revoke(keyId);
      toast.success("API key revoked successfully!");
      await loadAPIKeys();
    } catch (error) {
      const errorMessage =
        error.response?.data?.message ||
        "Failed to revoke API key. Please try again.";
      toast.error(errorMessage);
    }
  };

  const handleCopyKey = async (keyValue) => {
    try {
      await copyToClipboard(keyValue);
      setCopiedKeyId("new-key");
      toast.success("API key copied to clipboard!");
      setTimeout(() => setCopiedKeyId(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
      toast.error("Failed to copy API key. Please try again.");
    }
  };

  const handleCopyKeyId = async (keyId) => {
    try {
      await copyToClipboard(keyId);
      setCopiedKeyId(keyId);
      toast.success("Key ID copied to clipboard!");
      setTimeout(() => setCopiedKeyId(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
      toast.error("Failed to copy key ID. Please try again.");
    }
  };

  const formatDate = (dateString) => {
    if (!dateString) return "Never";
    const date = new Date(dateString);
    return date.toLocaleString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  };

  const isExpired = (expiresAt) => {
    if (!expiresAt) return false;
    return new Date(expiresAt) < new Date();
  };

  if (loading) {
    return (
      <div className="api-keys-page page-fade-in">
        <div className="loading-full">
          <Loading size="large" />
        </div>
      </div>
    );
  }

  return (
    <div className="api-keys-page page-fade-in">
      <PageHeader
        title={
          <>
            <Button
              variant="ghost"
              onClick={() => navigate("/dashboard/projects")}
              style={{ marginRight: "16px" }}
            >
              <FiArrowLeft size={18} />
            </Button>
            API Keys
          </>
        }
        actions={
          !showCreateForm && apiKeys.filter((k) => k.is_active).length === 0 ? (
            <Button variant="primary" onClick={() => setShowCreateForm(true)}>
              <FiPlus style={{ marginRight: "8px" }} />
              Generate API Key
            </Button>
          ) : null
        }
      />

      {/* New Key Modal */}
      <Modal
        isOpen={!!newKey}
        onClose={() => setNewKey(null)}
        title="API Key Created"
        size="medium"
      >
        {newKey && (
          <>
            <div
              style={{
                backgroundColor: "#fef3c7",
                border: "1px solid #f59e0b",
                color: "#92400e",
                padding: "12px 16px",
                borderRadius: "6px",
                marginBottom: "24px",
                fontSize: "14px",
              }}
            >
              ⚠️ Make sure to copy your API key now. You won't be able to see it
              again!
            </div>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "12px",
                marginBottom: "24px",
                padding: "16px",
                backgroundColor: "var(--bg-gray)",
                borderRadius: "6px",
                border: "1px solid var(--border-gray)",
              }}
            >
              <code
                className="code-text"
                style={{
                  flex: 1,
                  wordBreak: "break-all",
                  padding: 0,
                  background: "none",
                }}
              >
                {newKey.api_key}
              </code>
              <Button
                variant="ghost"
                size="small"
                onClick={() => handleCopyKey(newKey.api_key)}
                style={{ flexShrink: 0, padding: "8px", minWidth: "auto" }}
              >
                {copiedKeyId === "new-key" ? (
                  <FiCheck
                    style={{ color: "var(--success-green)" }}
                    size={18}
                  />
                ) : (
                  <FiCopy size={18} />
                )}
              </Button>
            </div>
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                gap: "8px",
                fontSize: "14px",
                color: "var(--text-primary)",
                marginBottom: "24px",
              }}
            >
              <p style={{ margin: 0 }}>
                <strong
                  style={{ color: "var(--text-secondary)", marginRight: "8px" }}
                >
                  Name:
                </strong>{" "}
                {newKey.name}
              </p>
              <p style={{ margin: 0 }}>
                <strong
                  style={{ color: "var(--text-secondary)", marginRight: "8px" }}
                >
                  Key ID:
                </strong>{" "}
                {newKey.uid}
              </p>
              {newKey.expires_at && (
                <p style={{ margin: 0 }}>
                  <strong
                    style={{
                      color: "var(--text-secondary)",
                      marginRight: "8px",
                    }}
                  >
                    Expires:
                  </strong>{" "}
                  {formatDate(newKey.expires_at)}
                </p>
              )}
            </div>
            <div className="form-actions">
              <Button variant="primary" onClick={() => setNewKey(null)}>
                I've copied the key
              </Button>
            </div>
          </>
        )}
      </Modal>

      {/* Create Form */}
      {showCreateForm && (
        <Card className="create-key-card">
          <div className="create-key-header">
            <h2>Generate New API Key</h2>
            <Button
              variant="ghost"
              size="small"
              onClick={() => {
                setShowCreateForm(false);
                setNewKeyName("");
                setExpiresInDays("");
                setError("");
              }}
            >
              <FiX size={18} />
            </Button>
          </div>
          <form onSubmit={handleCreate} className="create-key-form">
            <Input
              label="Key Name"
              type="text"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              placeholder="e.g., Production API Key"
              required
              className="create-key-input"
            />
            <Input
              label="Expires In (Days)"
              type="number"
              value={expiresInDays}
              onChange={(e) => setExpiresInDays(e.target.value)}
              placeholder="Leave empty for no expiration"
              min="0"
              className="create-key-input"
            />
            <p className="create-key-hint">
              Leave expiration empty for keys that never expire. You can only
              have one active API key at a time.
            </p>
            {error && (
              <div
                style={{
                  color: "var(--error-red)",
                  fontSize: "14px",
                  padding: "12px",
                  backgroundColor: "#fee2e2",
                  borderRadius: "6px",
                  border: "1px solid var(--error-red)",
                }}
              >
                {error}
              </div>
            )}
            <div className="form-actions">
              <Button
                type="button"
                variant="ghost"
                onClick={() => {
                  setShowCreateForm(false);
                  setNewKeyName("");
                  setExpiresInDays("");
                  setError("");
                }}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={creating || !newKeyName.trim()}
              >
                {creating ? "Generating..." : "Generate API Key"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      {/* API Keys List */}
      <Card>
        <div className="section-header">
          <div>
            <h2 className="section-title">Your API Keys</h2>
            {apiKeys.filter((k) => k.is_active).length > 0 && (
              <p
                style={{
                  fontSize: "14px",
                  color: "var(--text-secondary)",
                  margin: "8px 0 0 0",
                }}
              >
                You can only have one active API key. Revoke the current one to
                create a new one.
              </p>
            )}
          </div>
        </div>
        {apiKeys.length === 0 ? (
          <EmptyState
            icon={FiKey}
            title="No API keys found"
            description="Generate an API key to enable programmatic access to your projects."
            actionLabel="Generate Your First API Key"
            onAction={() => setShowCreateForm(true)}
          />
        ) : (
          <Table>
            <thead>
              <tr>
                <th>NAME</th>
                <th>KEY ID</th>
                <th>PREFIX</th>
                <th>STATUS</th>
                <th>CREATED</th>
                <th>LAST USED</th>
                <th>EXPIRES</th>
                <th>ACTIONS</th>
              </tr>
            </thead>
            <tbody>
              {apiKeys.map((key) => (
                <tr key={key.key_id}>
                  <td>
                    <strong>{key.name}</strong>
                  </td>
                  <td>
                    <code className="code-text">{key.key_id}</code>
                    <Button
                      variant="ghost"
                      size="small"
                      onClick={() => handleCopyKeyId(key.key_id)}
                      style={{
                        padding: "2px 4px",
                        marginLeft: "8px",
                        minWidth: "auto",
                      }}
                      title="Copy Key ID"
                    >
                      {copiedKeyId === key.key_id ? (
                        <FiCheck
                          size={14}
                          style={{ color: "var(--success-green)" }}
                        />
                      ) : (
                        <FiCopy size={14} />
                      )}
                    </Button>
                  </td>
                  <td>
                    <code className="code-text">{key.key_prefix}...</code>
                  </td>
                  <td>
                    {key.is_active ? (
                      <Badge variant="success">Active</Badge>
                    ) : key.revoked_at ? (
                      <Badge variant="error">Revoked</Badge>
                    ) : isExpired(key.expires_at) ? (
                      <Badge variant="error">Expired</Badge>
                    ) : (
                      <Badge variant="default">Inactive</Badge>
                    )}
                  </td>
                  <td>{formatDate(key.created_at)}</td>
                  <td>{formatDate(key.last_used_at)}</td>
                  <td>
                    {key.expires_at ? (
                      isExpired(key.expires_at) ? (
                        <span
                          style={{ color: "var(--error-red)", fontWeight: 500 }}
                        >
                          Expired
                        </span>
                      ) : (
                        formatDate(key.expires_at)
                      )
                    ) : (
                      <span
                        style={{
                          color: "var(--text-secondary)",
                          fontStyle: "italic",
                        }}
                      >
                        Never
                      </span>
                    )}
                  </td>
                  <td>
                    {key.is_active && (
                      <Button
                        variant="ghost"
                        size="small"
                        onClick={() => handleRevoke(key.key_id)}
                        style={{
                          color: "var(--error-red)",
                          padding: "4px 8px",
                        }}
                        onMouseEnter={(e) => {
                          e.currentTarget.style.backgroundColor = "#fee2e2";
                        }}
                        onMouseLeave={(e) => {
                          e.currentTarget.style.backgroundColor = "transparent";
                        }}
                      >
                        <FiTrash2 size={16} />
                        Revoke
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
      </Card>
    </div>
  );
}
