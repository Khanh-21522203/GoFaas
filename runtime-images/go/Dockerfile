# FaaS Go Runtime Base Image
FROM golang:1.22-alpine

# Install necessary tools
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Create function directory
RUN mkdir -p /app/function

# Copy runtime wrapper
COPY wrapper.sh /app/wrapper.sh
RUN chmod +x /app/wrapper.sh

# Set entrypoint
ENTRYPOINT ["/app/wrapper.sh"]
