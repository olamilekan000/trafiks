import { useState, useEffect, use } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  FiArrowLeft,
  FiCopy,
  FiCheck,
  FiPlus,
  FiX,
  FiLock,
} from "react-icons/fi";
import toast from "react-hot-toast";
import { serviceService } from "../services/serviceService";
import { projectService } from "../services/projectService";
import {
  Card,
  Button,
  Input,
  Loading,
  PageHeader,
  Badge,
} from "../components/common";
import "./ServiceConfig.css";

export default function ServiceConfig() {
  const { projectId } = useParams();
  const navigate = useNavigate();
  const [project, setProject] = useState(null);
  const [service, setService] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [copied, setCopied] = useState(false);
  const [certificate, setCertificate] = useState(null);

  // Form state
  const [source, setSource] = useState("trafiks");
  const [scheme, setScheme] = useState("http");
  const [proxyURL, setProxyURL] = useState("");
  const [targetBackendURL, setTargetBackendURL] = useState("");
  const [cacheEnabled, setCacheEnabled] = useState(false);
  const [cacheTTL, setCacheTTL] = useState(300);

  // Docker configuration state
  const [dockerLabels, setDockerLabels] = useState([{ key: "", value: "" }]);
  const [dockerNetwork, setDockerNetwork] = useState("");
  const [dockerPort, setDockerPort] = useState("");

  // Kubernetes configuration state
  const [k8sNamespace, setK8sNamespace] = useState("");
  const [k8sServiceName, setK8sServiceName] = useState("");
  const [k8sServicePort, setK8sServicePort] = useState("");

  // Unified configuration state
  const [config, setConfig] = useState({
    headers: {
      remove: [],
      add: {},
    },
    query_params: {
      remove: [],
    },
    https_redirect: false,
  });

  useEffect(() => {
    console.log("certificate", certificate);
  }, [certificate]);

  const [headersToAdd, setHeadersToAdd] = useState([{ key: "", value: "" }]);

  useEffect(() => {
    loadData();
  }, [projectId]);

  const loadData = async () => {
    try {
      setLoading(true);
      const projectRes = await projectService.get(projectId);
      const projectData = projectRes.data || projectRes;
      setProject(projectData);

      // Try to get service for this project
      try {
        const serviceRes = await serviceService.getByProject(projectId);
        const serviceData = serviceRes.data || serviceRes;

        console.log("serviceData", serviceData);

        setService(serviceData);
        setSource(serviceData.source || "trafiks");
        setScheme(serviceData.scheme || "http");
        // ProxyURL is stored as domain only (no scheme)
        setProxyURL(serviceData.proxy_url || "");
        setTargetBackendURL(serviceData.target_backend_url || "");
        setCacheEnabled(serviceData.cache_enabled || false);
        setCacheTTL(serviceData.cache_ttl || 300);
        setCertificate(serviceData.certificate || null);

        const serviceConfig = serviceData.configuration || {};

        // Initialize unified config state
        setConfig({
          headers: {
            remove: serviceConfig.headers?.remove || [],
            add: serviceConfig.headers?.add || {},
          },
          query_params: {
            remove: serviceConfig.query_params?.remove || [],
          },
          https_redirect:
            serviceConfig.https_redirect !== undefined
              ? serviceConfig.https_redirect
              : serviceData.scheme === "https",
        });

        // Initialize headers to add as array for UI
        if (
          serviceConfig.headers?.add &&
          typeof serviceConfig.headers.add === "object" &&
          Object.keys(serviceConfig.headers.add).length > 0
        ) {
          setHeadersToAdd(
            Object.entries(serviceConfig.headers.add).map(([key, value]) => ({
              key: key || "",
              value: value || "",
            }))
          );
        } else {
          setHeadersToAdd([{ key: "", value: "" }]);
        }

        // Initialize Docker configuration
        if (serviceData.source === "docker" && serviceConfig.docker) {
          const dockerConfig = serviceConfig.docker;
          if (
            dockerConfig.labels &&
            Object.keys(dockerConfig.labels).length > 0
          ) {
            setDockerLabels(
              Object.entries(dockerConfig.labels).map(([key, value]) => ({
                key: key || "",
                value: value || "",
              }))
            );
          } else {
            setDockerLabels([{ key: "", value: "" }]);
          }
          setDockerNetwork(dockerConfig.network || "");
          setDockerPort(dockerConfig.port || "");
        } else {
          setDockerLabels([{ key: "", value: "" }]);
          setDockerNetwork("");
          setDockerPort("");
        }

        // Initialize Kubernetes configuration
        if (serviceData.source === "kubernetes" && serviceConfig.kubernetes) {
          const k8sConfig = serviceConfig.kubernetes;
          setK8sNamespace(k8sConfig.namespace || "");
          setK8sServiceName(k8sConfig.service_name || "");
          setK8sServicePort(k8sConfig.service_port || "");
        } else {
          setK8sNamespace("");
          setK8sServiceName("");
          setK8sServicePort("");
        }
      } catch (serviceError) {
        // Service doesn't exist yet - that's okay, user can create it
        console.log("No service found, user can create one");
        setService(null);
        // Reset all form fields to defaults
        setSource("trafiks");
        setScheme("http");
        setProxyURL("");
        setTargetBackendURL("");
        setCacheEnabled(false);
        setCacheTTL(300);
        setConfig({
          headers: { remove: [], add: {} },
          query_params: { remove: [] },
          https_redirect: false,
        });
        setHeadersToAdd([{ key: "", value: "" }]);
        setDockerLabels([{ key: "", value: "" }]);
        setDockerNetwork("");
        setDockerPort("");
        setK8sNamespace("");
        setK8sServiceName("");
        setK8sServicePort("");
      }
    } catch (error) {
      console.error("Failed to load data:", error);
      toast.error("Failed to load service configuration. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  const handleConfigChange = (path, value) => {
    setConfig((prev) => {
      const newConfig = JSON.parse(JSON.stringify(prev)); // Deep clone
      const keys = path.split(".");
      let current = newConfig;

      for (let i = 0; i < keys.length - 1; i++) {
        if (!current[keys[i]]) {
          current[keys[i]] = {};
        }
        current = current[keys[i]];
      }

      current[keys[keys.length - 1]] = value;
      return newConfig;
    });
  };

  const handleCommaSeparatedChange = (path, value) => {
    const array = value
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
    handleConfigChange(path, array);
  };

  // Get comma-separated string from array
  const getCommaSeparated = (array) => {
    return Array.isArray(array) ? array.join(", ") : "";
  };

  const handleSave = async () => {
    try {
      setSaving(true);

      // Start with existing configuration to preserve any other fields
      const existingConfig = service?.configuration || {};
      const configuration = { ...existingConfig, ...config };

      // Process headers to add from key-value pairs (convert array to object)
      const headersToAddMap = {};
      headersToAdd.forEach(({ key, value }) => {
        const trimmedKey = key.trim();
        const trimmedValue = value.trim();
        if (trimmedKey && trimmedValue) {
          headersToAddMap[trimmedKey] = trimmedValue;
        }
      });

      // Update headers config
      if (
        Object.keys(headersToAddMap).length > 0 ||
        config.headers.remove.length > 0
      ) {
        configuration.headers = {
          remove: config.headers.remove,
          add: headersToAddMap,
        };
      }

      // Update query params config
      if (config.query_params.remove.length > 0) {
        configuration.query_params = {
          remove: config.query_params.remove,
        };
      }

      // Set HTTPS redirect based on scheme
      if (scheme === "https") {
        configuration.https_redirect = config.https_redirect;
      } else {
        // Remove https_redirect if switching to HTTP
        delete configuration.https_redirect;
      }

      // Set Docker configuration if source is docker
      if (source === "docker") {
        const labelsMap = {};
        dockerLabels.forEach(({ key, value }) => {
          const trimmedKey = key.trim();
          const trimmedValue = value.trim();
          if (trimmedKey && trimmedValue) {
            labelsMap[trimmedKey] = trimmedValue;
          }
        });

        if (Object.keys(labelsMap).length > 0 || dockerNetwork || dockerPort) {
          configuration.docker = {};
          if (Object.keys(labelsMap).length > 0) {
            configuration.docker.labels = labelsMap;
          }
          if (dockerNetwork.trim()) {
            configuration.docker.network = dockerNetwork.trim();
          }
          if (dockerPort.trim()) {
            configuration.docker.port = dockerPort.trim();
          }
        }
      }

      // Set Kubernetes configuration if source is kubernetes
      if (source === "kubernetes") {
        if (k8sNamespace || k8sServiceName || k8sServicePort) {
          configuration.kubernetes = {};
          if (k8sNamespace.trim()) {
            configuration.kubernetes.namespace = k8sNamespace.trim();
          }
          if (k8sServiceName.trim()) {
            configuration.kubernetes.service_name = k8sServiceName.trim();
          }
          if (k8sServicePort.trim()) {
            configuration.kubernetes.service_port = k8sServicePort.trim();
          }
        }
      }

      // Clean up empty config objects
      if (Object.keys(configuration.headers || {}).length === 0) {
        delete configuration.headers;
      }
      if (Object.keys(configuration.query_params || {}).length === 0) {
        delete configuration.query_params;
      }

      const configToSend =
        Object.keys(configuration).length > 0 ? configuration : undefined;

      if (service && service.uid) {
        // Update existing service - use pointers for optional fields
        const updateData = {
          Source: source,
          Scheme: scheme,
          ProxyURL: proxyURL,
          CacheEnabled: cacheEnabled,
          CacheTTL: cacheTTL,
        };
        // Only include TargetBackendURL if source is trafiks
        if (source === "trafiks" && targetBackendURL) {
          updateData.TargetBackendURL = targetBackendURL;
        }
        if (configToSend) {
          updateData.Configuration = configToSend;
        }
        await serviceService.update(projectId, service.uid, updateData);
      } else {
        // Create new service
        const createData = {
          Source: source,
          Scheme: scheme,
          ProxyURL: proxyURL,
          CacheEnabled: cacheEnabled,
          CacheTTL: cacheTTL,
        };
        // Only include TargetBackendURL if source is trafiks
        if (source === "trafiks") {
          createData.TargetBackendURL = targetBackendURL;
        }
        if (configToSend) {
          createData.Configuration = configToSend;
        }
        await serviceService.create(projectId, createData);
      }

      toast.success("Service configuration saved successfully!");
      await loadData();
    } catch (error) {
      console.error("Failed to save service:", error);
      const errorMessage =
        error.response?.data?.message ||
        "Failed to save service configuration. Please try again.";
      toast.error(errorMessage);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="service-config-page page-fade-in">
        <div className="loading-full">
          <Loading size="large" />
        </div>
      </div>
    );
  }

  return (
    <div className="service-config-page page-fade-in">
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
            Service Configuration
          </>
        }
      />
      {proxyURL && (
        <div className="proxy-url-banner">
          <div className="proxy-url-content">
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: "8px",
                flex: 1,
              }}
            >
              <div className="proxy-url-label">Proxy URL</div>
              {scheme === "https" && (
                <Badge
                  variant="success"
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "4px",
                    fontSize: "11px",
                    padding: "4px 8px",
                    backgroundColor: "rgba(16, 185, 129, 0.2)",
                    color: "rgba(255, 255, 255, 0.95)",
                    border: "1px solid rgba(16, 185, 129, 0.4)",
                  }}
                >
                  <FiLock size={12} />
                  HTTPS
                </Badge>
              )}
            </div>
            <code className="proxy-url-value">{`${scheme}://${proxyURL}`}</code>
            <Button
              variant="ghost"
              size="small"
              onClick={async () => {
                try {
                  const fullUrl = `${scheme}://${proxyURL}`;
                  await navigator.clipboard.writeText(fullUrl);
                  setCopied(true);
                  toast.success("Proxy URL copied to clipboard!");
                  setTimeout(() => setCopied(false), 2000);
                } catch (err) {
                  console.error("Failed to copy:", err);
                  toast.error("Failed to copy URL. Please try again.");
                }
              }}
              title="Copy to clipboard"
              className="proxy-url-copy-button"
            >
              {copied ? (
                <FiCheck size={18} style={{ color: "var(--success-green)" }} />
              ) : (
                <FiCopy size={18} />
              )}
            </Button>
          </div>
        </div>
      )}

      <Card>
        <h2 className="section-title">Backend Configuration</h2>
        <div className="form-section">
          <div className="input-group" style={{ maxWidth: "600px" }}>
            <label className="input-label">
              Source <span className="input-required">*</span>
            </label>
            <select
              value={source}
              onChange={(e) => {
                setSource(e.target.value);
                // Clear TargetBackendURL when switching away from trafiks
                if (e.target.value !== "trafiks") {
                  setTargetBackendURL("");
                }
                // Clear Kubernetes config when switching away from kubernetes
                if (e.target.value !== "kubernetes") {
                  setK8sNamespace("");
                  setK8sServiceName("");
                  setK8sServicePort("");
                }
                // Clear Docker config when switching away from docker
                if (e.target.value !== "docker") {
                  setDockerLabels([{ key: "", value: "" }]);
                  setDockerNetwork("");
                  setDockerPort("");
                }
              }}
              className="input-field"
              disabled={loading || saving || source === "kubernetes"}
            >
              <option value="trafiks">Trafiks (Manual URL)</option>
              <option value="docker">Docker</option>
              <option value="kubernetes" disabled>
                Kubernetes (Operator Managed)
              </option>
            </select>
            <div
              style={{
                marginTop: "8px",
                fontSize: "14px",
                color: "var(--text-secondary)",
              }}
            >
              {source === "trafiks" && "Provide a direct backend URL"}
              {source === "docker" && (
                <>
                  <div>Use Docker labels to discover services.</div>
                  <div
                    style={{
                      marginTop: "4px",
                      fontSize: "12px",
                      color: "var(--warning-text, #f59e0b)",
                    }}
                  >
                    Note: Your backend containers must run on the same Docker
                    network as Trafiks (for example,{" "}
                    <code>trafiks_network</code>) so Trafiks can reach them.
                  </div>
                </>
              )}
              {source === "kubernetes" && (
                <>
                  <div>This service is managed by the Kubernetes operator.</div>
                  <div
                    style={{
                      marginTop: "4px",
                      fontSize: "12px",
                      color: "var(--warning-text, #f59e0b)",
                    }}
                  >
                    Kubernetes source services can only be created and updated
                    via Kubernetes Custom Resources (TrafiksProxy). This view is
                    read-only.
                  </div>
                </>
              )}
            </div>
          </div>
          <div
            className="input-group"
            style={{ maxWidth: "600px", marginTop: "16px" }}
          >
            <label className="input-label">
              Scheme <span className="input-required">*</span>
            </label>
            <select
              value={scheme}
              onChange={(e) => {
                setScheme(e.target.value);
                // Reset HTTPS redirect when switching schemes
                if (e.target.value === "https") {
                  handleConfigChange("https_redirect", true); // Default to true for HTTPS
                } else {
                  handleConfigChange("https_redirect", false); // Not applicable for HTTP
                }
              }}
              className="input-field"
              disabled={loading || saving || source === "kubernetes"}
            >
              <option value="http">HTTP</option>
              <option value="https">HTTPS</option>
            </select>

            {scheme === "https" && (
              <>
                <div
                  style={{
                    marginTop: "8px",
                    fontSize: "12px",
                    color: "var(--text-secondary)",
                  }}
                >
                  Self-signed certificate will be automatically generated. Your
                  browser will show a security warning (this is normal for
                  self-signed certificates).
                </div>
                <div
                  style={{
                    marginTop: "16px",
                    padding: "16px",
                    backgroundColor: "#f0f9ff",
                    border: "1px solid #bae6fd",
                    borderRadius: "6px",
                  }}
                >
                  <div
                    className="checkbox-group"
                    style={{ alignItems: "flex-start" }}
                  >
                    <input
                      type="checkbox"
                      id="https-redirect"
                      checked={config.https_redirect || false}
                      onChange={(e) =>
                        handleConfigChange("https_redirect", e.target.checked)
                      }
                      className="checkbox-input"
                      disabled={loading || saving}
                      style={{ marginTop: "2px", flexShrink: 0 }}
                    />
                    <div style={{ flex: 1, marginLeft: "8px" }}>
                      <label
                        htmlFor="https-redirect"
                        className="checkbox-label"
                        style={{
                          fontWeight: 500,
                          margin: 0,
                          cursor: "pointer",
                        }}
                      >
                        Redirect HTTP requests to HTTPS
                      </label>
                      <div
                        style={{
                          marginTop: "8px",
                          fontSize: "12px",
                          color: "var(--text-secondary)",
                          lineHeight: "1.5",
                        }}
                      >
                        {config.https_redirect
                          ? "HTTP requests will be automatically redirected to HTTPS (301 redirect)."
                          : "HTTP requests will be rejected with an error (403 Forbidden)."}
                      </div>
                    </div>
                  </div>
                </div>
              </>
            )}
          </div>
          <div style={{ maxWidth: "600px", marginTop: "16px" }}>
            <Input
              label="Proxy URL"
              value={proxyURL}
              onChange={(e) => setProxyURL(e.target.value)}
              placeholder="my-api.trafikscloud.cloud"
              required
            />
            <div
              style={{
                marginTop: "4px",
                fontSize: "12px",
                color: "var(--text-secondary)",
              }}
            >
              Enter domain only (no http:// or https://)
            </div>
          </div>
          {source === "trafiks" && (
            <Input
              label="Target Backend URL"
              value={targetBackendURL}
              onChange={(e) => setTargetBackendURL(e.target.value)}
              placeholder="https://api.example.com"
              required
              style={{ maxWidth: "600px", marginTop: "16px" }}
            />
          )}
          {source === "docker" && (
            <Card style={{ marginTop: "24px" }}>
              <h2 className="section-title">Docker Configuration</h2>
              <div className="form-section">
                <div style={{ marginBottom: "24px" }}>
                  <label className="input-label">
                    Docker Labels <span className="input-required">*</span>
                  </label>
                  <div
                    style={{
                      fontSize: "12px",
                      color: "var(--text-secondary)",
                      marginBottom: "8px",
                    }}
                  >
                    Labels used to discover Docker containers. At least one
                    label is required.
                  </div>
                  {dockerLabels.map((label, index) => (
                    <div
                      key={index}
                      style={{
                        display: "flex",
                        gap: "8px",
                        marginBottom: "8px",
                        alignItems: "flex-start",
                      }}
                    >
                      <Input
                        placeholder="Label key (e.g., trafiks.service)"
                        value={label.key}
                        onChange={(e) => {
                          const newLabels = [...dockerLabels];
                          newLabels[index].key = e.target.value;
                          setDockerLabels(newLabels);
                        }}
                        style={{ flex: 1 }}
                      />
                      <Input
                        placeholder="Label value (e.g., my-api)"
                        value={label.value}
                        onChange={(e) => {
                          const newLabels = [...dockerLabels];
                          newLabels[index].value = e.target.value;
                          setDockerLabels(newLabels);
                        }}
                        style={{ flex: 1 }}
                      />
                      {dockerLabels.length > 1 && (
                        <Button
                          variant="secondary"
                          onClick={() => {
                            setDockerLabels(
                              dockerLabels.filter((_, i) => i !== index)
                            );
                          }}
                          style={{ padding: "8px 12px" }}
                        >
                          <FiX />
                        </Button>
                      )}
                    </div>
                  ))}
                  <Button
                    variant="secondary"
                    onClick={() => {
                      setDockerLabels([
                        ...dockerLabels,
                        { key: "", value: "" },
                      ]);
                    }}
                    style={{ marginTop: "8px" }}
                  >
                    <FiPlus style={{ marginRight: "4px" }} />
                    Add Label
                  </Button>
                </div>
                <div style={{ marginBottom: "24px" }}>
                  <Input
                    label="Container Port"
                    value={dockerPort}
                    onChange={(e) => setDockerPort(e.target.value)}
                    placeholder="8080"
                    required
                    style={{ maxWidth: "600px" }}
                  />
                  <div
                    style={{
                      marginTop: "4px",
                      fontSize: "12px",
                      color: "var(--text-secondary)",
                    }}
                  >
                    The port exposed by the container (e.g., 8080, 3000)
                  </div>
                </div>
                <div>
                  <Input
                    label="Docker Network (Optional)"
                    value={dockerNetwork}
                    onChange={(e) => setDockerNetwork(e.target.value)}
                    placeholder="bridge"
                    style={{ maxWidth: "600px" }}
                  />
                  <div
                    style={{
                      marginTop: "4px",
                      fontSize: "12px",
                      color: "var(--text-secondary)",
                    }}
                  >
                    Optional: Docker network name. If not specified, Trafiks
                    will auto-detect based on the environment.
                  </div>
                </div>
              </div>
            </Card>
          )}
          {source === "kubernetes" && (
            <Card style={{ marginTop: "24px" }}>
              <h2 className="section-title">Kubernetes Configuration</h2>
              <div className="form-section">
                <div style={{ marginBottom: "24px" }}>
                  <Input
                    label="Namespace"
                    value={k8sNamespace}
                    onChange={(e) => setK8sNamespace(e.target.value)}
                    placeholder="default"
                    required
                    style={{ maxWidth: "600px" }}
                  />
                  <div
                    style={{
                      marginTop: "4px",
                      fontSize: "12px",
                      color: "var(--text-secondary)",
                    }}
                  >
                    The Kubernetes namespace where your service is deployed
                  </div>
                </div>
                <div style={{ marginBottom: "24px" }}>
                  <Input
                    label="Service Name"
                    value={k8sServiceName}
                    onChange={(e) => setK8sServiceName(e.target.value)}
                    placeholder="my-api-service"
                    required
                    style={{ maxWidth: "600px" }}
                  />
                  <div
                    style={{
                      marginTop: "4px",
                      fontSize: "12px",
                      color: "var(--text-secondary)",
                    }}
                  >
                    The name of the Kubernetes Service resource
                  </div>
                </div>
                <div>
                  <Input
                    label="Service Port"
                    value={k8sServicePort}
                    onChange={(e) => setK8sServicePort(e.target.value)}
                    placeholder="8080 or http"
                    required
                    style={{ maxWidth: "600px" }}
                  />
                  <div
                    style={{
                      marginTop: "4px",
                      fontSize: "12px",
                      color: "var(--text-secondary)",
                    }}
                  >
                    Port number (e.g., 8080) or port name (e.g., http) from the
                    Service definition
                  </div>
                </div>
              </div>
            </Card>
          )}
          {source !== "trafiks" &&
            source !== "docker" &&
            source !== "kubernetes" && (
              <div
                style={{
                  marginTop: "16px",
                  padding: "16px",
                  backgroundColor: "var(--bg-secondary)",
                  borderRadius: "8px",
                  maxWidth: "600px",
                }}
              >
                <div
                  style={{ fontSize: "14px", color: "var(--text-secondary)" }}
                >
                  For {source} source, you'll configure service discovery using
                  labels/selectors in the Configuration section below.
                </div>
              </div>
            )}
        </div>
      </Card>
      <Card>
        <h2 className="section-title">Cache Settings</h2>
        <div className="form-section">
          <div className="checkbox-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={cacheEnabled}
                onChange={(e) => setCacheEnabled(e.target.checked)}
                className="checkbox-input"
              />
              <span>Enable caching</span>
            </label>
          </div>
          {cacheEnabled && (
            <Input
              label="Cache TTL (seconds)"
              type="number"
              value={cacheTTL}
              onChange={(e) => setCacheTTL(parseInt(e.target.value) || 300)}
              min="0"
              max="86400"
              style={{ maxWidth: "600px" }}
            />
          )}
        </div>
      </Card>
      {scheme === "https" && certificate && (
        <Card>
          <h2 className="section-title">Certificate Details</h2>
          <div className="form-section">
            <div className="detail-item">
              <span className="detail-item-label">Type:</span>
              <span className="detail-item-value">
                {certificate.type === "selfsigned"
                  ? "Self-Signed"
                  : certificate.type === "letsencrypt"
                  ? "Let's Encrypt"
                  : "Manual"}
              </span>
            </div>
            <div className="detail-item">
              <span className="detail-item-label">Common Name:</span>
              <span className="detail-item-value">
                {certificate.common_name}
              </span>
            </div>
            {certificate.dns_names && certificate.dns_names.length > 0 && (
              <div className="detail-item">
                <span className="detail-item-label">DNS Names:</span>
                <span className="detail-item-value">
                  {certificate.dns_names.join(", ")}
                </span>
              </div>
            )}
            <div className="detail-item">
              <span className="detail-item-label">Issuer:</span>
              <span className="detail-item-value">{certificate.issuer}</span>
            </div>
            <div className="detail-item">
              <span className="detail-item-label">Valid From:</span>
              <span className="detail-item-value">
                {new Date(certificate.valid_from).toLocaleString()}
              </span>
            </div>
            <div className="detail-item">
              <span className="detail-item-label">Valid To:</span>
              <span
                className="detail-item-value"
                style={{
                  color: certificate.is_expired
                    ? "#ef4444"
                    : certificate.days_until_expiry <= 30
                    ? "#f59e0b"
                    : "inherit",
                }}
              >
                {new Date(certificate.valid_to).toLocaleString()}
                {certificate.is_expired && (
                  <Badge variant="danger" style={{ marginLeft: "8px" }}>
                    Expired
                  </Badge>
                )}
                {!certificate.is_expired &&
                  certificate.days_until_expiry <= 30 && (
                    <Badge variant="warning" style={{ marginLeft: "8px" }}>
                      Expires in {certificate.days_until_expiry} days
                    </Badge>
                  )}
              </span>
            </div>
            <div className="detail-item">
              <span className="detail-item-label">Serial Number:</span>
              <span className="detail-item-value code-text">
                {certificate.serial_number}
              </span>
            </div>
          </div>
        </Card>
      )}
      <Card>
        <h2 className="section-title">Header Modifications</h2>
        <div className="form-section">
          <div className="textarea-group">
            <label className="textarea-label">
              Headers to Remove (comma-separated)
            </label>
            <textarea
              value={getCommaSeparated(config.headers.remove)}
              onChange={(e) =>
                handleCommaSeparatedChange("headers.remove", e.target.value)
              }
              placeholder="X-Internal-Token, X-Debug-Header"
              className="config-textarea"
              rows="3"
            />
          </div>
          <div className="key-value-group">
            <label className="textarea-label">Headers to Add</label>
            <div className="key-value-list">
              {headersToAdd.map((header, index) => (
                <div key={index} className="key-value-row">
                  <Input
                    type="text"
                    value={header.key}
                    onChange={(e) => {
                      const newHeaders = [...headersToAdd];
                      newHeaders[index].key = e.target.value;
                      setHeadersToAdd(newHeaders);
                    }}
                    placeholder="Header name"
                    className="key-value-input"
                    disabled={loading || saving}
                  />
                  <Input
                    type="text"
                    value={header.value}
                    onChange={(e) => {
                      const newHeaders = [...headersToAdd];
                      newHeaders[index].value = e.target.value;
                      setHeadersToAdd(newHeaders);
                    }}
                    placeholder="Header value"
                    className="key-value-input"
                    disabled={loading || saving}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="small"
                    onClick={() => {
                      const newHeaders = headersToAdd.filter(
                        (_, i) => i !== index
                      );
                      if (newHeaders.length === 0) {
                        setHeadersToAdd([{ key: "", value: "" }]);
                      } else {
                        setHeadersToAdd(newHeaders);
                      }
                    }}
                    disabled={loading || saving}
                    className="key-value-remove-button"
                  >
                    <FiX size={16} />
                  </Button>
                </div>
              ))}
              <Button
                type="button"
                variant="ghost"
                size="small"
                onClick={() => {
                  setHeadersToAdd([...headersToAdd, { key: "", value: "" }]);
                }}
                disabled={loading || saving}
                className="key-value-add-button"
              >
                <FiPlus size={16} style={{ marginRight: "4px" }} />
                Add Header
              </Button>
            </div>
          </div>
        </div>
      </Card>
      <Card>
        <h2 className="section-title">Query Parameters</h2>
        <div className="form-section">
          <div className="textarea-group">
            <label className="textarea-label">
              Query Parameters to Remove (comma-separated)
            </label>
            <textarea
              value={getCommaSeparated(config.query_params.remove)}
              onChange={(e) =>
                handleCommaSeparatedChange(
                  "query_params.remove",
                  e.target.value
                )
              }
              placeholder="utm_source, utm_medium"
              className="config-textarea"
              rows="3"
            />
          </div>
        </div>
      </Card>
      <div className="form-actions" style={{ marginTop: "24px" }}>
        <Button
          variant="primary"
          onClick={handleSave}
          disabled={
            loading ||
            saving ||
            source === "kubernetes" ||
            !proxyURL ||
            (source === "trafiks" && !targetBackendURL) ||
            (source === "docker" &&
              (!dockerPort ||
                dockerLabels.every((l) => !l.key.trim() || !l.value.trim()))) ||
            (source === "kubernetes" &&
              (!k8sNamespace || !k8sServiceName || !k8sServicePort))
          }
          size="large"
        >
          {source === "kubernetes"
            ? "Read-Only (Operator Managed)"
            : saving
            ? "Saving..."
            : "Save Configuration"}
        </Button>
      </div>
    </div>
  );
}
