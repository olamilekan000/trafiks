import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  FiArrowLeft,
  FiX,
  FiCheck,
  FiXCircle,
  FiSend,
  FiFileText,
  FiDownload,
  FiServer,
  FiClock,
  FiGlobe,
  FiCopy,
  FiRefreshCw,
} from "react-icons/fi";
import toast from "react-hot-toast";
import { requestLogService } from "../services/requestLogService";
import { projectService } from "../services/projectService";
import { copyToClipboard } from "../utils/clipboard";
import {
  Card,
  Button,
  Badge,
  Loading,
  Modal,
  Table,
  Pagination,
  EmptyState,
  PageHeader,
} from "../components/common";
import "./RequestLogs.css";

export default function RequestLogs() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const [project, setProject] = useState(null);
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedLog, setSelectedLog] = useState(null);
  const [replaying, setReplaying] = useState(false);
  const [replayResult, setReplayResult] = useState(null);
  const [pagination, setPagination] = useState({
    page: 1,
    limit: 20,
    total: 0,
    totalPages: 0,
  });

  // Filters
  const [filters, setFilters] = useState({
    method: "",
    status_code: "",
    cache_hit: "",
    path: "",
    start_time: "",
    end_time: "",
  });

  useEffect(() => {
    loadProject();
    loadLogs();
  }, [projectId, pagination.page, filters]);

  const loadProject = async () => {
    try {
      const response = await projectService.get(projectId);
      setProject(response.data);
    } catch (error) {
      console.error("Failed to load project:", error);
    }
  };

  const loadLogs = async () => {
    try {
      setLoading(true);
      const activeFilters = {};
      if (filters.method) activeFilters.method = filters.method;
      if (filters.status_code)
        activeFilters.status_code = parseInt(filters.status_code);
      if (filters.cache_hit !== "")
        activeFilters.cache_hit = filters.cache_hit === "true";
      if (filters.path) activeFilters.path = filters.path;
      // Convert datetime-local to ISO format
      if (filters.start_time) {
        activeFilters.start_time = new Date(filters.start_time).toISOString();
      }
      if (filters.end_time) {
        activeFilters.end_time = new Date(filters.end_time).toISOString();
      }

      const response = await requestLogService.list(
        projectId,
        pagination.page,
        pagination.limit,
        activeFilters
      );
      setLogs(response.request_logs || []);
      setPagination((prev) => ({
        ...prev,
        total: response.pagination?.total || 0,
        totalPages: response.pagination?.total_pages || 0,
      }));
    } catch (error) {
      console.error("Failed to load request logs:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (key, value) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
    setPagination((prev) => ({ ...prev, page: 1 })); // Reset to first page on filter change
  };

  const clearFilters = () => {
    setFilters({
      method: "",
      status_code: "",
      cache_hit: "",
      path: "",
      start_time: "",
      end_time: "",
    });
    setPagination((prev) => ({ ...prev, page: 1 }));
  };

  const handlePageChange = (newPage) => {
    setPagination((prev) => ({ ...prev, page: newPage }));
  };

  const getStatusBadgeVariant = (statusCode) => {
    if (statusCode >= 200 && statusCode < 300) return "success";
    if (statusCode >= 400 && statusCode < 500) return "error";
    if (statusCode >= 500) return "error";
    return "default";
  };

  const formatDate = (dateString) => {
    return new Date(dateString).toLocaleString();
  };

  const formatBytes = (bytes) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
  };

  const generateCurlCommand = (log) => {
    if (!log) return "";

    const baseUrl = log.proxy_url;
    const method = log.method;
    let curl = `curl -X ${method}`;

    if (log.request_headers && typeof log.request_headers === "object") {
      Object.entries(log.request_headers).forEach(([key, value]) => {
        const escapedValue = String(value).replace(/"/g, '\\"');
        curl += ` \\\n  -H "${key}: ${escapedValue}"`;
      });
    }

    if (["POST", "PUT", "PATCH"].includes(method) && log.request_body) {
      const body = log.request_body.trim();
      if (body.startsWith("{") || body.startsWith("[")) {
        curl += ` \\\n  -d '${body.replace(/'/g, "'\\''")}'`;
      } else {
        curl += ` \\\n  -d "${body.replace(/"/g, '\\"')}"`;
      }
    }

    let fullUrl = baseUrl;
    if (log.path && log.path !== "/") {
      if (fullUrl.endsWith("/")) {
        fullUrl = fullUrl.slice(0, -1);
      }
      const path = log.path.startsWith("/") ? log.path : `/${log.path}`;
      fullUrl += path;
    }
    if (log.query_string) {
      fullUrl += `?${log.query_string}`;
    }

    // Ensure URL has protocol
    if (
      fullUrl &&
      !fullUrl.startsWith("http://") &&
      !fullUrl.startsWith("https://")
    ) {
      fullUrl = `https://${fullUrl}`;
    }

    curl += ` \\\n  "${fullUrl}"`;

    return curl;
  };

  const handleExportCurl = async () => {
    if (!selectedLog) return;

    try {
      const curlCommand = generateCurlCommand(selectedLog);
      await copyToClipboard(curlCommand);
      toast.success("cURL command copied to clipboard!");
    } catch (err) {
      console.error("Failed to copy:", err);
      toast.error("Failed to copy to clipboard");
    }
  };

  // Replay request
  const handleReplayRequest = async () => {
    if (!selectedLog || !projectId) return;

    setReplaying(true);
    setReplayResult(null);

    try {
      const response = await requestLogService.replay(
        projectId,
        selectedLog.uid
      );
      setReplayResult(response);
      toast.success("Request replayed successfully!");
    } catch (error) {
      const errorMessage =
        error.response?.data?.message || "Failed to replay request";
      setReplayResult({ error: errorMessage });
      toast.error(errorMessage);
    } finally {
      setReplaying(false);
    }
  };

  if (loading && !logs.length) {
    return (
      <div className="request-logs-page page-fade-in">
        <div className="loading-full">
          <Loading size="large" />
        </div>
      </div>
    );
  }

  return (
    <div className="request-logs-page page-fade-in">
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
            Request Logs
          </>
        }
      />

      {/* Filters */}
      <Card>
        <div className="section-header">
          <h3 className="section-title">Filters</h3>
          <Button variant="ghost" size="small" onClick={clearFilters}>
            Clear All
          </Button>
        </div>
        <div className="grid-auto-fit" style={{ gap: "16px" }}>
          <div className="filter-group">
            <label>Method</label>
            <select
              value={filters.method}
              onChange={(e) => handleFilterChange("method", e.target.value)}
              className="filter-select"
            >
              <option value="">All</option>
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
              <option value="PATCH">PATCH</option>
            </select>
          </div>
          <div className="filter-group">
            <label>Status Code</label>
            <input
              type="number"
              value={filters.status_code}
              onChange={(e) =>
                handleFilterChange("status_code", e.target.value)
              }
              placeholder="e.g., 200"
              className="filter-input"
            />
          </div>
          <div className="filter-group">
            <label>Cache Hit</label>
            <select
              value={filters.cache_hit}
              onChange={(e) => handleFilterChange("cache_hit", e.target.value)}
              className="filter-select"
            >
              <option value="">All</option>
              <option value="true">Yes</option>
              <option value="false">No</option>
            </select>
          </div>
          <div className="filter-group">
            <label>Path</label>
            <input
              type="text"
              value={filters.path}
              onChange={(e) => handleFilterChange("path", e.target.value)}
              placeholder="e.g., /api"
              className="filter-input"
            />
          </div>
          <div className="filter-group">
            <label>Start Time</label>
            <input
              type="datetime-local"
              value={filters.start_time}
              onChange={(e) => handleFilterChange("start_time", e.target.value)}
              className="filter-input"
            />
          </div>
          <div className="filter-group">
            <label>End Time</label>
            <input
              type="datetime-local"
              value={filters.end_time}
              onChange={(e) => handleFilterChange("end_time", e.target.value)}
              className="filter-input"
            />
          </div>
        </div>
      </Card>

      {/* Logs Table */}
      <Card>
        {logs.length === 0 ? (
          <EmptyState
            icon={FiFileText}
            title="No request logs found"
            description="Request logs will appear here once your service receives requests."
          />
        ) : (
          <>
            <Table>
              <thead>
                <tr>
                  <th>Status</th>
                  <th>Method</th>
                  <th>Path</th>
                  <th>Response Time</th>
                  <th>Cache</th>
                  <th>Size</th>
                  <th>Timestamp</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((log) => (
                  <tr key={log.uid}>
                    <td>
                      <Badge variant={getStatusBadgeVariant(log.status_code)}>
                        {log.status_code}
                      </Badge>
                    </td>
                    <td>
                      <span
                        className={`method-badge method-${log.method.toLowerCase()}`}
                      >
                        {log.method}
                      </span>
                    </td>
                    <td>
                      <div className="path-cell">
                        <span className="path-value">{log.path}</span>
                        {log.query_string && (
                          <span className="query-string">
                            ?{log.query_string}
                          </span>
                        )}
                      </div>
                    </td>
                    <td>{log.response_time} ms</td>
                    <td>
                      {log.cache_hit ? (
                        <Badge variant="success">Hit</Badge>
                      ) : (
                        <Badge variant="default">Miss</Badge>
                      )}
                    </td>
                    <td>{formatBytes(log.response_size || 0)}</td>
                    <td>{formatDate(log.requested_at)}</td>
                    <td>
                      <Button
                        variant="primary"
                        size="small"
                        onClick={() => setSelectedLog(log)}
                        className="view-details-button"
                      >
                        View Details
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>

            <Pagination
              currentPage={pagination.page}
              totalPages={pagination.totalPages}
              total={pagination.total}
              onPageChange={handlePageChange}
            />
          </>
        )}
      </Card>

      {/* Detail Modal */}
      <Modal
        isOpen={!!selectedLog}
        onClose={() => {
          setSelectedLog(null);
          setReplayResult(null);
          setReplaying(false);
        }}
        title="Request Details"
        size="xlarge"
        showCloseButton={false}
      >
        {selectedLog && (
          <>
            <div style={{ display: "flex", gap: "8px", marginBottom: "16px" }}>
              <Button
                variant="ghost"
                size="small"
                onClick={handleExportCurl}
                title="Export as cURL"
              >
                <FiCopy size={16} style={{ marginRight: "4px" }} />
                Export cURL
              </Button>
              <Button
                variant="primary"
                size="small"
                onClick={handleReplayRequest}
                disabled={replaying}
                title="Replay this request"
              >
                <FiRefreshCw
                  size={16}
                  style={{
                    marginRight: "4px",
                    animation: replaying ? "spin 1s linear infinite" : "none",
                  }}
                />
                {replaying ? "Replaying..." : "Replay"}
              </Button>
            </div>
            <div className="detail-section">
              <h3>
                <FiSend className="section-icon" />
                Request
              </h3>
              <div className="grid-auto-fit">
                <div className="detail-item">
                  <label className="detail-item-label">Method</label>
                  <span
                    className={`method-badge method-${selectedLog.method.toLowerCase()}`}
                  >
                    {selectedLog.method}
                  </span>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Path</label>
                  <span className="detail-item-value">{selectedLog.path}</span>
                </div>
                {selectedLog.query_string && (
                  <div className="detail-item">
                    <label className="detail-item-label">Query</label>
                    <code className="code-text">
                      {selectedLog.query_string}
                    </code>
                  </div>
                )}
                <div className="detail-item detail-item-highlight">
                  <label className="detail-item-label">
                    <FiGlobe style={{ marginRight: "4px" }} />
                    Target URL
                  </label>
                  <code className="code-text">{selectedLog.target_url}</code>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">IP Address</label>
                  <span className="detail-item-value">
                    {selectedLog.ip_address}
                  </span>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">User Agent</label>
                  <span
                    className="detail-item-value"
                    style={{ fontSize: "12px", color: "var(--text-secondary)" }}
                  >
                    {selectedLog.user_agent}
                  </span>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Request Size</label>
                  <span className="detail-item-value">
                    {formatBytes(selectedLog.request_size || 0)}
                  </span>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Timestamp</label>
                  <span className="detail-item-value">
                    {formatDate(selectedLog.requested_at)}
                  </span>
                </div>
              </div>
            </div>

            {selectedLog.request_headers &&
              Object.keys(selectedLog.request_headers).length > 0 && (
                <div className="detail-section">
                  <h3>
                    <FiFileText className="section-icon" />
                    Request Headers
                  </h3>
                  <pre className="code-block">
                    {JSON.stringify(selectedLog.request_headers, null, 2)}
                  </pre>
                </div>
              )}

            {selectedLog.request_body && (
              <div className="detail-section">
                <h3>
                  <FiFileText className="section-icon" />
                  Request Body
                </h3>
                <pre className="code-block">{selectedLog.request_body}</pre>
              </div>
            )}

            <div className="detail-section">
              <h3>
                <FiDownload className="section-icon" />
                Response
              </h3>
              <div className="grid-auto-fit">
                <div className="detail-item">
                  <label className="detail-item-label">Status Code</label>
                  <Badge
                    variant={getStatusBadgeVariant(selectedLog.status_code)}
                  >
                    {selectedLog.status_code}
                  </Badge>
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Response Time</label>
                  <span className="detail-item-value">
                    {selectedLog.response_time} ms
                  </span>
                </div>
                <div className="detail-item detail-item-cache">
                  <label className="detail-item-label">
                    <FiClock style={{ marginRight: "4px" }} />
                    Cache Hit
                  </label>
                  {selectedLog.cache_hit ? (
                    <Badge
                      variant="success"
                      className="cache-badge cache-hit-badge"
                    >
                      <FiCheck style={{ marginRight: "6px" }} />
                      Cache Hit
                    </Badge>
                  ) : (
                    <Badge
                      variant="default"
                      className="cache-badge cache-miss-badge"
                    >
                      <FiXCircle style={{ marginRight: "6px" }} />
                      Cache Miss
                    </Badge>
                  )}
                </div>
                <div className="detail-item">
                  <label className="detail-item-label">Response Size</label>
                  <span className="detail-item-value">
                    {formatBytes(selectedLog.response_size || 0)}
                  </span>
                </div>
              </div>
            </div>

            {selectedLog.response_headers &&
              Object.keys(selectedLog.response_headers).length > 0 && (
                <div className="detail-section">
                  <h3>
                    <FiFileText className="section-icon" />
                    Response Headers
                  </h3>
                  <pre className="code-block">
                    {JSON.stringify(selectedLog.response_headers, null, 2)}
                  </pre>
                </div>
              )}

            {selectedLog.response_body && (
              <div className="detail-section">
                <h3>
                  <FiServer className="section-icon" />
                  Response Body
                </h3>
                <pre className="code-block">{selectedLog.response_body}</pre>
              </div>
            )}

            {/* Replay Result */}
            {replayResult && (
              <div className="detail-section">
                <h3>
                  <FiRefreshCw className="section-icon" />
                  Replay Result
                </h3>
                {replayResult.error ? (
                  <div className="replay-error">
                    <Badge variant="error">Error</Badge>
                    <p>{replayResult.error}</p>
                  </div>
                ) : (
                  <div className="replay-success">
                    <div className="replay-result-header">
                      <Badge variant="success">Success</Badge>
                      <span>Status: {replayResult.status_code || "N/A"}</span>
                      <span>
                        Response Time: {replayResult.response_time || "N/A"}ms
                      </span>
                    </div>
                    {replayResult.response_body && (
                      <pre className="code-block">
                        {replayResult.response_body}
                      </pre>
                    )}
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </Modal>
    </div>
  );
}
