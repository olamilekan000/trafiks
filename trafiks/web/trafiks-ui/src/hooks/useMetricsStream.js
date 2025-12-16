import { useEffect, useRef, useState } from "react";
import api from "../services/api";

/**
 * Custom hook for SSE metrics streaming
 * @param {string} projectId - The project ID to stream metrics for
 * @param {function} onEvent - Callback function when an event is received
 * @returns {object} - { connected, error, reconnect }
 */
export function useMetricsStream(projectId, onEvent) {
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(null);
  const eventSourceRef = useRef(null);
  const reconnectTimeoutRef = useRef(null);
  const reconnectAttemptsRef = useRef(0);

  const MAX_RECONNECT_ATTEMPTS = 5;
  const RECONNECT_DELAY = 3000; // 3 seconds

  const connect = () => {
    if (!projectId) return;

    // Close existing connection if any
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // Clear any pending reconnect
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }

    try {
      // Build SSE URL
      const baseURL = api.defaults.baseURL || "";
      const url = `${baseURL}/projects/${projectId}/metrics/stream`;

      // Create EventSource
      // Cookies are sent automatically for same-origin requests
      const eventSource = new EventSource(url);

      eventSource.onopen = () => {
        setConnected(true);
        setError(null);
        reconnectAttemptsRef.current = 0;
      };

      // Handle generic messages (data events)
      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (onEvent) {
            onEvent(data);
          }
        } catch (err) {
          console.error("Error parsing SSE message:", err);
        }
      };

      // Handle connection confirmation
      eventSource.addEventListener("connected", (event) => {
        setConnected(true);
        setError(null);
      });

      // Handle keepalive pings
      eventSource.addEventListener("ping", (event) => {
        // Keepalive ping, no action needed
      });

      eventSource.onerror = (err) => {
        setConnected(false);
        eventSource.close();

        // Attempt to reconnect
        if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttemptsRef.current += 1;
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, RECONNECT_DELAY);
        } else {
          setError("Failed to connect after multiple attempts");
        }
      };

      eventSourceRef.current = eventSource;
    } catch (err) {
      setError(err.message);
      setConnected(false);
    }
  };

  const disconnect = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    setConnected(false);
  };

  const reconnect = () => {
    reconnectAttemptsRef.current = 0;
    disconnect();
    connect();
  };

  useEffect(() => {
    if (projectId) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [projectId]);

  return { connected, error, reconnect };
}
