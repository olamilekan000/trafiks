#!/usr/bin/env python3
import http.server
import socketserver
import json
from datetime import datetime

class MyHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        # Set cache headers
        self.send_header("Cache-Control", "public, max-age=300, s-maxage=60")
        self.send_header("ETag", "test-etag-123")
        self.send_header("Last-Modified", datetime.utcnow().strftime("%a, %d %b %Y %H:%M:%S GMT"))
        # Echo back request headers as response headers
        for header_name, header_value in self.headers.items():
            self.send_header(f"X-Echo-{header_name}", header_value)
        self.end_headers()
        # Parse query parameters
        query_params = {}
        if "?" in self.path:
            query_string = self.path.split("?")[1]
            for param in query_string.split("&"):
                if "=" in param:
                    key, value = param.split("=", 1)
                    query_params[key] = value
        
        response = {
            "service": "test-api",
            "status": "running",
            "timestamp": datetime.now().isoformat(),
            "path": self.path.split("?")[0],
            "query_params": query_params,
            "message": "Hello from Docker-discovered service!",
            "request_headers": dict(self.headers)
        }
        self.wfile.write(json.dumps(response, indent=2).encode())

if __name__ == "__main__":
    PORT = 8080
    with socketserver.TCPServer(("", PORT), MyHandler) as httpd:
        print(f"Server running on port {PORT}")
        httpd.serve_forever()
