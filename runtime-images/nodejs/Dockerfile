# FaaS Node.js Runtime Base Image
FROM node:20-alpine

# Set working directory
WORKDIR /app

# Create function directory
RUN mkdir -p /app/function

# Copy runtime wrapper
COPY wrapper.sh /app/wrapper.sh
RUN chmod +x /app/wrapper.sh

# Set entrypoint
ENTRYPOINT ["/app/wrapper.sh"]
