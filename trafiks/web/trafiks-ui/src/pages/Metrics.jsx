import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { FiArrowLeft, FiBarChart2, FiXCircle, FiZap } from "react-icons/fi";
import { metricsService } from "../services/metricsService";
import { requestLogService } from "../services/requestLogService";
import { projectService } from "../services/projectService";
import { useMetricsStream } from "../hooks/useMetricsStream";
import {
  Card,
  Badge,
  Button,
  Loading,
  IconContainer,
  PageHeader,
  Table,
} from "../components/common";
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import "./Metrics.css";

export default function Metrics() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const [metrics, setMetrics] = useState(null);
  const [recentLogs, setRecentLogs] = useState([]);
  const [project, setProject] = useState(null);
  const [loading, setLoading] = useState(true);
  const [timeRange, setTimeRange] = useState("24h");
  const [groupBy, setGroupBy] = useState("1min");

  // Handle real-time metrics updates
  const handleMetricsEvent = useCallback((event) => {
    if (event.type === "request_logged") {
      // Update metrics incrementally
      setMetrics((prev) => {
        if (!prev) return prev;

        const data = prev.data || prev;
        const newData = { ...data };

        // Increment counters
        newData.total_requests = (newData.total_requests || 0) + 1;
        if (event.data.is_error) {
          newData.total_errors = (newData.total_errors || 0) + 1;
        }
        if (event.data.cache_hit) {
          newData.total_cache_hits = (newData.total_cache_hits || 0) + 1;
        }

        // Update average response time (simple moving average)
        const currentAvg = newData.average_response_time || 0;
        const requestCount = newData.total_requests || 1;
        const newResponseTime = event.data.response_time || 0;
        newData.average_response_time =
          (currentAvg * (requestCount - 1) + newResponseTime) / requestCount;

        // Update time series (add to current hour)
        const now = new Date();
        const currentHour = new Date(
          now.getFullYear(),
          now.getMonth(),
          now.getDate(),
          now.getHours()
        );
        const timeSeries = [...(newData.requests_over_time || [])];

        // Find or create current hour data point
        let currentPoint = timeSeries.find(
          (point) => new Date(point.time).getTime() === currentHour.getTime()
        );

        if (!currentPoint) {
          currentPoint = {
            time: currentHour.toISOString(),
            count: 0,
            errors: 0,
            cache_hits: 0,
            average_response_time: 0,
          };
          timeSeries.push(currentPoint);
          // Keep only last 24 hours
          if (timeSeries.length > 24) {
            timeSeries.shift();
          }
        }

        currentPoint.count = (currentPoint.count || 0) + 1;
        if (event.data.is_error) {
          currentPoint.errors = (currentPoint.errors || 0) + 1;
        }
        if (event.data.cache_hit) {
          currentPoint.cache_hits = (currentPoint.cache_hits || 0) + 1;
        }

        // Update average response time for this hour
        const hourAvg = currentPoint.average_response_time || 0;
        const hourCount = currentPoint.count || 1;
        currentPoint.average_response_time =
          (hourAvg * (hourCount - 1) + newResponseTime) / hourCount;

        newData.requests_over_time = timeSeries;

        return { ...prev, data: newData };
      });

      // Reload recent logs periodically (every 5 events)
      if (Math.random() < 0.2) {
        loadRecentLogs();
      }
    } else if (event.type === "metrics_snapshot") {
      // Update with full snapshot
      setMetrics((prev) => ({
        ...prev,
        data: {
          ...(prev?.data || prev || {}),
          ...event.data,
        },
      }));
    }
  }, []);

  // Set up SSE connection
  const { connected: streamConnected } = useMetricsStream(
    projectId,
    handleMetricsEvent
  );

  useEffect(() => {
    loadProject();
    loadMetrics();
    loadRecentLogs();
  }, [projectId, timeRange, groupBy]);

  const loadProject = async () => {
    try {
      const response = await projectService.get(projectId);
      const projectData = response.data || response;
      setProject(projectData);
    } catch (error) {
      console.error("Failed to load project:", error);
    }
  };

  const loadMetrics = async () => {
    try {
      setLoading(true);
      const endTime = new Date();
      const startTime = new Date();

      switch (timeRange) {
        case "24h":
          startTime.setHours(startTime.getHours() - 24);
          break;
        case "7d":
          startTime.setDate(startTime.getDate() - 7);
          break;
        case "30d":
          startTime.setDate(startTime.getDate() - 30);
          break;
        default:
          startTime.setHours(startTime.getHours() - 24);
      }

      const response = await metricsService.getMetrics(projectId, {
        start_time: startTime.toISOString(),
        end_time: endTime.toISOString(),
        group_by: groupBy,
      });
      // Response is wrapped in { message, data, success }
      setMetrics(response || {});
    } catch (error) {
      console.error("Failed to load metrics:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadRecentLogs = async () => {
    try {
      const response = await requestLogService.list(projectId, 1, 10);
      // Response is wrapped in { message, data, success }
      setRecentLogs(response?.request_logs || response?.logs || []);
    } catch (error) {
      console.error("Failed to load recent logs:", error);
    }
  };

  const formatTime = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
    });
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  };

  const getStatusBadgeVariant = (statusCode) => {
    if (statusCode >= 200 && statusCode < 300) return "success";
    if (statusCode >= 400 && statusCode < 500) return "error";
    if (statusCode >= 500) return "error";
    return "default";
  };

  const getStatusText = (statusCode) => {
    if (statusCode >= 200 && statusCode < 300) return "Success";
    if (statusCode >= 400 && statusCode < 500) return "Client Error";
    if (statusCode >= 500) return "Server Error";
    return `${statusCode}`;
  };

  if (loading && !metrics) {
    return (
      <div className="metrics-page page-fade-in">
        <div className="loading-full">
          <Loading size="large" />
        </div>
      </div>
    );
  }

  return (
    <div className="metrics-page page-fade-in">
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
            Request Metrics
          </>
        }
        actions={
          <div style={{ display: "flex", gap: "12px" }}>
            <select
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value)}
              style={{
                padding: "8px 12px",
                fontSize: "14px",
                border: "1px solid var(--border-gray)",
                borderRadius: "6px",
                backgroundColor: "white",
                color: "var(--text-primary)",
                cursor: "pointer",
              }}
              onFocus={(e) => {
                e.target.style.outline = "none";
                e.target.style.borderColor = "var(--primary-blue)";
                e.target.style.boxShadow = "0 0 0 3px rgba(59, 130, 246, 0.1)";
              }}
              onBlur={(e) => {
                e.target.style.borderColor = "var(--border-gray)";
                e.target.style.boxShadow = "none";
              }}
            >
              <option value="24h">Last 24 Hours</option>
              <option value="7d">Last 7 Days</option>
              <option value="30d">Last 30 Days</option>
            </select>
            <select
              value={groupBy}
              onChange={(e) => setGroupBy(e.target.value)}
              style={{
                padding: "8px 12px",
                fontSize: "14px",
                border: "1px solid var(--border-gray)",
                borderRadius: "6px",
                backgroundColor: "white",
                color: "var(--text-primary)",
                cursor: "pointer",
              }}
              onFocus={(e) => {
                e.target.style.outline = "none";
                e.target.style.borderColor = "var(--primary-blue)";
                e.target.style.boxShadow = "0 0 0 3px rgba(59, 130, 246, 0.1)";
              }}
              onBlur={(e) => {
                e.target.style.borderColor = "var(--border-gray)";
                e.target.style.boxShadow = "none";
              }}
            >
              <option value="1min">1 Minute</option>
              <option value="5min">5 Minutes</option>
              <option value="30min">30 Minutes</option>
              <option value="1h">1 Hour</option>
              <option value="day">By Day</option>
              <option value="week">By Week</option>
              <option value="month">By Month</option>
            </select>
          </div>
        }
      />

      {/* Summary Cards */}
      <div className="metrics-summary">
        <Card className="summary-card">
          <IconContainer size="large" className="summary-icon">
            <FiBarChart2 size={24} />
          </IconContainer>
          <div className="summary-content">
            <div className="summary-value">
              {(metrics?.data || metrics)?.total_requests || 0}
            </div>
            <div className="summary-label">Total Requests</div>
          </div>
        </Card>
        <Card className="summary-card">
          <IconContainer size="large" className="summary-icon">
            <FiXCircle size={24} />
          </IconContainer>
          <div className="summary-content">
            <div className="summary-value error">
              {(metrics?.data || metrics)?.total_errors || 0}
            </div>
            <div className="summary-label">Errors</div>
          </div>
        </Card>
        <Card className="summary-card">
          <IconContainer size="large" className="summary-icon">
            <FiZap size={24} />
          </IconContainer>
          <div className="summary-content">
            <div className="summary-value success">
              {(metrics?.data || metrics)?.total_cache_hits || 0}
            </div>
            <div className="summary-label">Cache Hits</div>
          </div>
        </Card>
      </div>

      {/* Charts */}
      <div className="metrics-charts">
        <Card className="chart-card">
          <h2 className="section-title">Requests Over Time</h2>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart
              data={(metrics?.data || metrics)?.requests_over_time || []}
            >
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="time"
                tickFormatter={(value) => {
                  const date = new Date(value);
                  return date.toLocaleDateString("en-US", {
                    month: "short",
                    day: "numeric",
                  });
                }}
              />
              <YAxis yAxisId="left" />
              <YAxis yAxisId="right" orientation="right" />
              <Tooltip
                labelFormatter={(value) => {
                  const date = new Date(value);
                  return date.toLocaleString();
                }}
                formatter={(value, name) => {
                  if (name === "Response Time (ms)") {
                    return [`${Math.round(value)}ms`, name];
                  }
                  return [value, name];
                }}
              />
              <Legend />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="count"
                stroke="#3b82f6"
                strokeWidth={2}
                name="Requests"
              />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="errors"
                stroke="#ef4444"
                strokeWidth={2}
                name="Errors"
              />
              <Line
                yAxisId="left"
                type="monotone"
                dataKey="cache_hits"
                stroke="#f59e0b"
                strokeWidth={2}
                name="Cache Hits"
              />
              <Line
                yAxisId="right"
                type="monotone"
                dataKey="average_response_time"
                stroke="#10b981"
                strokeWidth={2}
                strokeDasharray="5 5"
                name="Avg. Response Time (ms)"
              />
            </LineChart>
          </ResponsiveContainer>
        </Card>

        <div className="charts-row">
          <Card className="chart-card">
            <h2 className="section-title">Status Codes</h2>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={(metrics?.data || metrics)?.status_codes || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="status_code" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="count" fill="#3b82f6" />
              </BarChart>
            </ResponsiveContainer>
          </Card>

          <Card className="chart-card">
            <h2 className="section-title">HTTP Methods</h2>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={(metrics?.data || metrics)?.methods || []}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="method" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="count" fill="#10b981" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </div>
      </div>

      {/* Recent Requests Table */}
      <Card>
        <div className="section-header">
          <h2 className="section-title">Recent Requests</h2>
        </div>
        <Table>
          <thead>
            <tr>
              <th>STATUS</th>
              <th>METHOD</th>
              <th>PATH</th>
              <th>RESPONSE TIME</th>
              <th>TIME</th>
            </tr>
          </thead>
          <tbody>
            {recentLogs.length === 0 ? (
              <tr>
                <td
                  colSpan="5"
                  style={{
                    textAlign: "center",
                    color: "var(--text-secondary)",
                    padding: "32px",
                  }}
                >
                  No requests found
                </td>
              </tr>
            ) : (
              recentLogs.map((log) => (
                <tr key={log.uid}>
                  <td>
                    <Badge variant={getStatusBadgeVariant(log.status_code)}>
                      {log.status_code} {getStatusText(log.status_code)}
                    </Badge>
                  </td>
                  <td>
                    <code className="code-text">{log.method}</code>
                  </td>
                  <td>
                    <code className="code-text">{log.path}</code>
                  </td>
                  <td>{log.response_time}ms</td>
                  <td>
                    {formatDate(log.requested_at)}{" "}
                    {formatTime(log.requested_at)}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </Table>
      </Card>
    </div>
  );
}
