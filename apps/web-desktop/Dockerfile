# Build stage
FROM node:20-alpine AS builder

WORKDIR /workspace

# Copy the local ui-components package that web-desktop depends on
COPY packages/ui-components/ ./packages/ui-components/

# Copy web-desktop package files first for better caching
COPY apps/web-desktop/package.json ./apps/web-desktop/package.json
COPY apps/web-desktop/package-lock.json ./apps/web-desktop/package-lock.json

# Install dependencies (from within web-desktop directory)
# The file: reference to ui-components resolves via the workspace structure
WORKDIR /workspace/apps/web-desktop
RUN npm ci

# Copy the rest of the web-desktop source
WORKDIR /workspace
COPY apps/web-desktop/ ./apps/web-desktop/

# Build the Next.js app (standalone output enabled in next.config.ts)
WORKDIR /workspace/apps/web-desktop
RUN npm run build

# Runtime stage - uses standalone output for minimal image
FROM node:20-alpine AS runtime

WORKDIR /app

# Create non-root user
RUN adduser -D -g '' appuser

# Copy standalone output
COPY --from=builder /workspace/apps/web-desktop/.next/standalone ./
COPY --from=builder /workspace/apps/web-desktop/.next/static ./.next/static
COPY --from=builder /workspace/apps/web-desktop/public ./public

# Change ownership to appuser
RUN chown -R appuser:appuser /app

# Use non-root user
USER appuser

EXPOSE 3000

ENV HOSTNAME="0.0.0.0"
CMD ["node", "server.js"]