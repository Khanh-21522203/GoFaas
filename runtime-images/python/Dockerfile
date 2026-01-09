# FaaS Python Runtime Base Image
FROM python:3.11-slim

# Install common dependencies
RUN pip install --no-cache-dir requests

# Set working directory
WORKDIR /app

# Create function directory
RUN mkdir -p /app/function

# Copy runtime wrapper
COPY wrapper.sh /app/wrapper.sh
RUN chmod +x /app/wrapper.sh

# Set entrypoint
ENTRYPOINT ["/app/wrapper.sh"]
