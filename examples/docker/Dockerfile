FROM python:3.11-slim

WORKDIR /app

RUN echo '#!/usr/bin/env python3\n\
import http.server\n\
import socketserver\n\
import json\n\
from datetime import datetime\n\
\n\
class MyHandler(http.server.SimpleHTTPRequestHandler):\n\
    def do_GET(self):\n\
        self.send_response(200)\n\
        self.send_header("Content-type", "application/json")\n\
        self.end_headers()\n\
        response = {\n\
            "service": "test-api",\n\
            "status": "running",\n\
            "timestamp": datetime.now().isoformat(),\n\
            "path": self.path,\n\
            "message": "Hello from Docker-discovered service!"\n\
        }\n\
        self.wfile.write(json.dumps(response, indent=2).encode())\n\
\n\
if __name__ == "__main__":\n\
    PORT = 8080\n\
    with socketserver.TCPServer(("", PORT), MyHandler) as httpd:\n\
        print(f"Server running on port {PORT}")\n\
        httpd.serve_forever()\n\
' > server.py && chmod +x server.py

EXPOSE 8080

CMD ["python3", "server.py"]
