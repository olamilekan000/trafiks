import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import {
  FiArrowLeft,
  FiAlertOctagon,
  FiPlus,
  FiTrash2,
  FiCopy,
  FiCheck,
  FiX,
  FiEdit2,
  FiToggleLeft,
  FiToggleRight,
  FiClock,
  FiSend,
  FiRefreshCw,
  FiEye,
} from "react-icons/fi";
import toast from "react-hot-toast";
import { webhookService } from "../services/webhookService";
import {
  Card,
  Button,
  Input,
  Badge,
  Loading,
  Modal,
  Table,
  Pagination,
  EmptyState,
  PageHeader,
} from "../components/common";
import "./Webhooks.css";

// Available webhook events
const WEBHOOK_EVENTS = {
  "upstream.unreachable": "Upstream Unreachable",
  "upstream.timeout": "Upstream Timeout",
  "request.failed": "Request Failed",
  "cache.miss": "Cache Miss",
  "project.activated": "Project Activated",
  "project.deactivated": "Project Deactivated",
  "secretkey.created": "Secret Key Created",
  "secretkey.deactivated": "Secret Key Deactivated",
  "error_rate.high": "Error Rate High",
};

export default function Webhooks() {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [webhooks, setWebhooks] = useState([]);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [creating, setCreating] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [copiedWebhookId, setCopiedWebhookId] = useState(null);
  const [newWebhook, setNewWebhook] = useState(null);

  // Deliveries state
  const [deliveries, setDeliveries] = useState([]);
  const [deliveriesLoading, setDeliveriesLoading] = useState(false);
  const [selectedDelivery, setSelectedDelivery] = useState(null);
  const [deliveriesPagination, setDeliveriesPagination] = useState({
    page: 1,
    limit: 20,
    total: 0,
    totalPages: 0,
  });

  // Form state
  const [webhookURL, setWebhookURL] = useState("");
  const [enabledEvents, setEnabledEvents] = useState({});
  const [isActive, setIsActive] = useState(true);

  useEffect(() => {
    loadWebhooks();
  }, []);

  const loadWebhooks = async () => {
    try {
      setLoading(true);
      const response = await webhookService.list();
      const webhooksData = response.data?.webhooks || response?.webhooks || [];
      setWebhooks(webhooksData);
      // If webhook exists, load its deliveries
      if (webhooksData.length > 0 && webhooksData[0].uid) {
        loadDeliveries(webhooksData[0].uid, 1);
      }
    } catch (error) {
      console.error("Failed to load webhooks:", error);
      toast.error("Failed to load webhooks. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setWebhookURL("");
    setEnabledEvents({});
    setIsActive(true);
    setError("");
    setShowCreateForm(false);
    setEditingWebhook(null);
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    setError("");

    if (!webhookURL.trim()) {
      setError("Webhook URL is required");
      return;
    }

    const selectedEvents = Object.keys(enabledEvents).filter(
      (key) => enabledEvents[key]
    );
    if (selectedEvents.length === 0) {
      setError("At least one event must be enabled");
      return;
    }

    try {
      setCreating(true);
      const eventsMap = {};
      selectedEvents.forEach((event) => {
        eventsMap[event] = true;
      });

      const response = await webhookService.create({
        url: webhookURL.trim(),
        enabled_events: eventsMap,
      });

      const webhookData = response.data || response;
      setNewWebhook(webhookData); // Store webhook with secret to show in modal
      setShowCreateForm(false);
      await loadWebhooks();
    } catch (error) {
      const errorMessage =
        error.response?.data?.message ||
        "Failed to create webhook. Please try again.";
      setError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setCreating(false);
    }
  };

  const handleUpdate = async (e) => {
    e.preventDefault();
    setError("");

    if (!webhookURL.trim()) {
      setError("Webhook URL is required");
      return;
    }

    const selectedEvents = Object.keys(enabledEvents).filter(
      (key) => enabledEvents[key]
    );
    if (selectedEvents.length === 0) {
      setError("At least one event must be enabled");
      return;
    }

    try {
      setSaving(true);
      const eventsMap = {};
      selectedEvents.forEach((event) => {
        eventsMap[event] = true;
      });

      const updateData = {
        url: webhookURL.trim(),
        enabled_events: eventsMap,
      };

      // Always include is_active when editing
      updateData.is_active = isActive;

      await webhookService.update(editingWebhook.uid, updateData);

      toast.success("Webhook updated successfully!");
      resetForm();
      await loadWebhooks();
    } catch (error) {
      const errorMessage =
        error.response?.data?.message ||
        "Failed to update webhook. Please try again.";
      setError(errorMessage);
      toast.error(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (webhookId) => {
    if (
      !window.confirm(
        "Are you sure you want to delete this webhook? This action cannot be undone."
      )
    ) {
      return;
    }

    try {
      await webhookService.delete(webhookId);
      toast.success("Webhook deleted successfully!");
      await loadWebhooks();
    } catch (error) {
      const errorMessage =
        error.response?.data?.message ||
        "Failed to delete webhook. Please try again.";
      toast.error(errorMessage);
    }
  };

  const handleEdit = (webhook) => {
    setEditingWebhook(webhook);
    setWebhookURL(webhook.url);
    setEnabledEvents(webhook.enabled_events || {});
    setIsActive(webhook.is_active !== false);
    setShowCreateForm(true);
  };

  const toggleEvent = (eventType) => {
    setEnabledEvents((prev) => ({
      ...prev,
      [eventType]: !prev[eventType],
    }));
  };

  const handleCopyWebhookId = async (webhookId) => {
    try {
      await navigator.clipboard.writeText(webhookId);
      setCopiedWebhookId(webhookId);
      toast.success("Webhook ID copied to clipboard!");
      setTimeout(() => setCopiedWebhookId(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
      toast.error("Failed to copy webhook ID. Please try again.");
    }
  };

  const handleCopySecret = async (text, type = "secret") => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedWebhookId(type);
      toast.success(
        type === "url"
          ? "Webhook URL copied to clipboard!"
          : "Webhook secret copied to clipboard!"
      );
      setTimeout(() => setCopiedWebhookId(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
      toast.error("Failed to copy. Please try again.");
    }
  };

  const loadDeliveries = async (webhookId = null, page = 1) => {
    try {
      setDeliveriesLoading(true);
      const params = {
        page,
        limit: deliveriesPagination.limit,
      };

      const response = await webhookService.listDeliveries(webhookId, params);

      const deliveriesData = response?.deliveries || [];
      setDeliveries(deliveriesData || []);

      if (response?.pagination) {
        setDeliveriesPagination({
          page: response.pagination.page,
          limit: response.pagination.limit,
          total: response.pagination.total,
          totalPages: response.pagination.total_pages,
        });
      }
    } catch (error) {
      console.error("Failed to load deliveries:", error);
      toast.error("Failed to load webhook deliveries. Please try again.");
    } finally {
      setDeliveriesLoading(false);
    }
  };

  const getStatusBadge = (status) => {
    const statusConfig = {
      pending: { variant: "default", label: "Pending" },
      processing: { variant: "default", label: "Processing" },
      success: { variant: "success", label: "Success" },
      failed: { variant: "error", label: "Failed" },
    };

    const config = statusConfig[status] || statusConfig.pending;
    return <Badge variant={config.variant}>{config.label}</Badge>;
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

  if (loading && webhooks.length === 0) {
    return (
      <div className="webhooks-loading">
        <Loading size="large" />
      </div>
    );
  }

  return (
    <div className="webhooks-page page-fade-in">
      <PageHeader
        title={
          <>
            <Button
              variant="ghost"
              onClick={() => navigate("/dashboard/profile")}
              style={{ marginRight: "16px" }}
            >
              <FiArrowLeft size={20} />
            </Button>
            Webhooks
          </>
        }
        actions={
          !showCreateForm && webhooks.length === 0 ? (
            <Button
              variant="primary"
              onClick={() => {
                resetForm();
                setShowCreateForm(true);
              }}
            >
              <FiPlus style={{ marginRight: "8px" }} />
              Create Webhook
            </Button>
          ) : null
        }
      />
      {/* Create/Edit Form */}
      {showCreateForm && (
        <Card className="create-webhook-card">
          <div className="create-webhook-header">
            <h2>{editingWebhook ? "Edit Webhook" : "Create New Webhook"}</h2>
            <Button variant="ghost" size="small" onClick={resetForm}>
              <FiX size={18} />
            </Button>
          </div>
          <form
            onSubmit={editingWebhook ? handleUpdate : handleCreate}
            className="create-webhook-form"
          >
            <Input
              label="Webhook URL"
              type="url"
              value={webhookURL}
              onChange={(e) => setWebhookURL(e.target.value)}
              placeholder="https://example.com/webhook"
              required
              className="create-webhook-input"
            />
            {!editingWebhook && (
              <p className="create-webhook-hint">
                A secure secret will be automatically generated for this
                webhook. You'll be able to copy it after creation.
              </p>
            )}

            <div className="events-section">
              <label className="events-label">Enabled Events</label>
              <p className="events-hint">
                Select which events should trigger this webhook
              </p>
              <div className="events-grid">
                {Object.entries(WEBHOOK_EVENTS).map(([eventType, label]) => (
                  <label key={eventType} className="event-checkbox">
                    <input
                      type="checkbox"
                      checked={enabledEvents[eventType] || false}
                      onChange={() => toggleEvent(eventType)}
                    />
                    <span>{label}</span>
                  </label>
                ))}
              </div>
            </div>

            {editingWebhook && (
              <div className="active-toggle">
                <label className="toggle-label">
                  <span>Active</span>
                  <button
                    type="button"
                    onClick={() => setIsActive(!isActive)}
                    className="toggle-button"
                  >
                    {isActive ? (
                      <FiToggleRight size={24} color="var(--primary-blue)" />
                    ) : (
                      <FiToggleLeft size={24} color="var(--text-secondary)" />
                    )}
                  </button>
                </label>
              </div>
            )}

            {error && <div className="create-webhook-error">{error}</div>}
            <div className="create-webhook-actions">
              <Button type="button" variant="ghost" onClick={resetForm}>
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={
                  (editingWebhook ? saving : creating) ||
                  !webhookURL.trim() ||
                  Object.keys(enabledEvents).filter((k) => enabledEvents[k])
                    .length === 0
                }
              >
                {editingWebhook
                  ? saving
                    ? "Saving..."
                    : "Save Changes"
                  : creating
                  ? "Creating..."
                  : "Create Webhook"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      {/* New Webhook Secret Modal */}
      <Modal
        isOpen={!!(newWebhook && newWebhook.secret)}
        onClose={() => setNewWebhook(null)}
        title="Webhook Created"
        size="medium"
      >
        <div className="webhook-created-content">
          <div className="new-key-warning">
            <span>⚠️</span>
            <span>
              Make sure to copy your webhook secret now. You won't be able to
              see it again!
            </span>
          </div>

          <div className="new-key-section">
            <label className="new-key-label">Webhook Secret</label>
            <div className="new-key-display">
              <code className="new-key-value">{newWebhook?.secret}</code>
              <Button
                variant="ghost"
                size="small"
                onClick={() => handleCopySecret(newWebhook?.secret, "secret")}
                className="copy-key-button"
                title="Copy secret"
              >
                {copiedWebhookId === "secret" ? (
                  <FiCheck
                    style={{ color: "var(--success-green)" }}
                    size={18}
                  />
                ) : (
                  <FiCopy size={18} />
                )}
              </Button>
            </div>
          </div>

          <div className="new-key-info">
            <div className="new-key-info-item">
              <label>URL</label>
              <div className="new-key-info-value">
                <code>{newWebhook?.url}</code>
                <Button
                  variant="ghost"
                  size="small"
                  onClick={() => handleCopySecret(newWebhook?.url, "url")}
                  className="copy-key-button"
                  title="Copy URL"
                >
                  {copiedWebhookId === "url" ? (
                    <FiCheck
                      style={{ color: "var(--success-green)" }}
                      size={16}
                    />
                  ) : (
                    <FiCopy size={16} />
                  )}
                </Button>
              </div>
            </div>

            <div className="new-key-info-item">
              <label>Webhook ID</label>
              <code>{newWebhook?.uid}</code>
            </div>

            <div className="new-key-info-item">
              <label>Created</label>
              <span>{formatDate(newWebhook?.created_at)}</span>
            </div>
          </div>

          <div className="modal-footer">
            <Button variant="primary" onClick={() => setNewWebhook(null)}>
              I've copied the secret
            </Button>
          </div>
        </div>
      </Modal>

      {/* Webhook Card */}
      {!showCreateForm && webhooks.length > 0 && (
        <>
          <Card className="webhook-detail-card">
            <div className="webhook-detail-header">
              <div>
                <h2>Webhook Configuration</h2>
                <p className="webhook-detail-subtitle">
                  Manage your webhook settings
                </p>
              </div>
              <div className="webhook-detail-actions">
                <Button
                  variant="ghost"
                  size="small"
                  onClick={() => handleEdit(webhooks[0])}
                  title="Edit Webhook"
                >
                  <FiEdit2 size={18} />
                  Edit
                </Button>
                <Button
                  variant="ghost"
                  size="small"
                  onClick={() => handleDelete(webhooks[0].uid)}
                  className="delete-button"
                  title="Delete Webhook"
                >
                  <FiTrash2 size={18} />
                  Delete
                </Button>
              </div>
            </div>

            <div className="webhook-detail-content">
              <div className="webhook-detail-item">
                <label>Webhook URL</label>
                <code className="webhook-url-display">{webhooks[0].url}</code>
              </div>

              <div className="webhook-detail-item">
                <label>Status</label>
                <div className="status-toggle-container">
                  {webhooks[0].is_active ? (
                    <Badge variant="success">Active</Badge>
                  ) : (
                    <Badge variant="error">Inactive</Badge>
                  )}
                  <button
                    type="button"
                    onClick={async (e) => {
                      e.preventDefault();
                      e.stopPropagation();
                      try {
                        const newActiveState = !webhooks[0].is_active;
                        await webhookService.update(webhooks[0].uid, {
                          is_active: newActiveState,
                        });
                        toast.success(
                          `Webhook ${
                            newActiveState ? "activated" : "deactivated"
                          } successfully!`
                        );
                        await loadWebhooks();
                      } catch (error) {
                        const errorMessage =
                          error.response?.data?.message ||
                          "Failed to update webhook status. Please try again.";
                        toast.error(errorMessage);
                      }
                    }}
                    className="webhook-status-toggle"
                    title={
                      webhooks[0].is_active
                        ? "Click to deactivate"
                        : "Click to activate"
                    }
                  >
                    {webhooks[0].is_active ? (
                      <FiToggleRight size={36} color="var(--primary-blue)" />
                    ) : (
                      <FiToggleLeft size={36} color="var(--text-secondary)" />
                    )}
                  </button>
                </div>
              </div>

              <div className="webhook-detail-item">
                <label>Enabled Events</label>
                <div className="enabled-events-display">
                  {Object.keys(webhooks[0].enabled_events || {})
                    .filter((key) => webhooks[0].enabled_events[key])
                    .map((eventType) => (
                      <Badge
                        key={eventType}
                        variant="default"
                        className="event-badge"
                      >
                        {WEBHOOK_EVENTS[eventType] || eventType}
                      </Badge>
                    ))}
                </div>
              </div>

              <div className="webhook-detail-item">
                <label>Created</label>
                <span>{formatDate(webhooks[0].created_at)}</span>
              </div>
            </div>
          </Card>

          {/* Deliveries Section */}
          <Card className="deliveries-card">
            <div className="deliveries-header">
              <h2>Webhook Deliveries</h2>
              <Button
                variant="ghost"
                size="small"
                onClick={() =>
                  loadDeliveries(webhooks[0].uid, deliveriesPagination.page)
                }
                title="Refresh"
              >
                <FiRefreshCw size={18} />
              </Button>
            </div>

            {deliveriesLoading ? (
              <div className="loading-center">
                <Loading size="medium" />
              </div>
            ) : deliveries.length === 0 ? (
              <EmptyState
                icon={FiSend}
                title="No deliveries found"
                description="Deliveries will appear here once webhook events are triggered."
              />
            ) : (
              <>
                <Table>
                  <thead>
                    <tr>
                      <th>Event Type</th>
                      <th>Status</th>
                      <th>Retries</th>
                      <th>Created</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {deliveries.map((delivery) => (
                      <tr key={delivery.uid}>
                        <td>
                          <code className="code-text">
                            {delivery.event_type}
                          </code>
                        </td>
                        <td>{getStatusBadge(delivery.status)}</td>
                        <td>
                          {delivery.retry_count > 0 ? (
                            <Badge variant="default">
                              {delivery.retry_count}
                            </Badge>
                          ) : (
                            <span style={{ color: "var(--text-secondary)" }}>
                              -
                            </span>
                          )}
                        </td>
                        <td>{formatDate(delivery.created_at)}</td>
                        <td>
                          <Button
                            variant="ghost"
                            size="small"
                            onClick={() => setSelectedDelivery(delivery)}
                            title="View Details"
                          >
                            <FiEye size={16} />
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </Table>

                <Pagination
                  currentPage={deliveriesPagination.page}
                  totalPages={deliveriesPagination.totalPages}
                  total={deliveriesPagination.total}
                  onPageChange={(page) => loadDeliveries(webhooks[0].uid, page)}
                />
              </>
            )}
          </Card>
        </>
      )}

      {/* Empty State */}
      {!showCreateForm && webhooks.length === 0 && (
        <Card>
          <EmptyState
            icon={FiAlertOctagon}
            title="No webhook configured"
            description="Create a webhook to receive real-time notifications about events across all your projects."
            actionLabel="Create Webhook"
            onAction={() => {
              resetForm();
              setShowCreateForm(true);
            }}
          />
        </Card>
      )}

      {/* Delivery Details Modal */}
      <Modal
        isOpen={!!selectedDelivery}
        onClose={() => setSelectedDelivery(null)}
        title="Delivery Details"
        size="large"
        footer={
          <Button variant="primary" onClick={() => setSelectedDelivery(null)}>
            Close
          </Button>
        }
      >
        {selectedDelivery && (
          <>
            <div className="delivery-detail-section">
              <h3>Basic Information</h3>
              <div className="grid-auto-fit">
                <div className="detail-item">
                  <label className="detail-item-label">UID</label>
                  <code className="code-text">{selectedDelivery.uid}</code>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Event Type</label>
                  <code className="code-text">
                    {selectedDelivery.event_type}
                  </code>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Status</label>
                  {getStatusBadge(selectedDelivery.status)}
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Retry Count</label>
                  <span className="detail-item-value">
                    {selectedDelivery.retry_count}
                  </span>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Created At</label>
                  <span className="detail-item-value">
                    {formatDate(selectedDelivery.created_at)}
                  </span>
                </div>
                {selectedDelivery.delivered_at && (
                  <div className="detail-item">
                    <label className="detail-item-label">Delivered At</label>
                    <span className="detail-item-value">
                      {formatDate(selectedDelivery.delivered_at)}
                    </span>
                  </div>
                )}
                {selectedDelivery.next_retry_at && (
                  <div className="detail-item">
                    <label className="detail-item-label">Next Retry At</label>
                    <span className="detail-item-value">
                      {formatDate(selectedDelivery.next_retry_at)}
                    </span>
                  </div>
                )}
                {selectedDelivery.http_status_code && (
                  <div className="detail-item">
                    <label className="detail-item-label">
                      HTTP Status Code
                    </label>
                    <Badge
                      variant={
                        selectedDelivery.http_status_code >= 200 &&
                        selectedDelivery.http_status_code < 300
                          ? "success"
                          : "error"
                      }
                    >
                      {selectedDelivery.http_status_code}
                    </Badge>
                  </div>
                )}
              </div>
            </div>

            {selectedDelivery.payload && (
              <div className="delivery-detail-section">
                <h3>Payload</h3>
                <pre className="code-block">
                  {JSON.stringify(selectedDelivery.payload, null, 2)}
                </pre>
              </div>
            )}

            {selectedDelivery.response_body && (
              <div className="delivery-detail-section">
                <h3>Response Body</h3>
                <pre className="code-block">
                  {selectedDelivery.response_body}
                </pre>
              </div>
            )}

            {selectedDelivery.error_message && (
              <div className="delivery-detail-section">
                <h3>Error Message</h3>
                <div className="delivery-error">
                  {selectedDelivery.error_message}
                </div>
              </div>
            )}

            {selectedDelivery.idempotency_key && (
              <div className="delivery-detail-section">
                <h3>Idempotency Key</h3>
                <code className="code-text">
                  {selectedDelivery.idempotency_key}
                </code>
              </div>
            )}
          </>
        )}
      </Modal>
    </div>
  );
}
