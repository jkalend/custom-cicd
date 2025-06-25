FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    curl \
    git \
    docker.io \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app/logs

COPY requirements.txt .

RUN pip install --no-cache-dir -r requirements.txt

COPY agent.py .
COPY api.py .
COPY cli.py .
# COPY pipeline_example.json .

RUN chmod +x cli.py

# Create a startup script
RUN echo '#!/bin/bash\n\
# Start the API server\n\
python api.py &\n\
\n\
# Keep container running\n\
wait' > /app/start.sh && chmod +x /app/start.sh

EXPOSE 8080

ENV PORT=8080
ENV DEBUG=false
ENV PYTHONPATH=/app

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["/app/start.sh"] 
