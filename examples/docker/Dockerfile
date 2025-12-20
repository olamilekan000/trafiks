FROM python:3.11-slim

WORKDIR /app

COPY server.py /app/server.py
RUN chmod +x /app/server.py

EXPOSE 8080

CMD ["python3", "server.py"]
