FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Declare build arguments (agar bisa diterima saat build)
ARG DB_HOST
ARG DB_NAME
ARG DB_PASSWORD
ARG DB_PORT
ARG DB_SSLMODE
ARG DB_USER
ARG JWT_SECRET
ARG PORT

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/main .

# Declare same build args here to avoid warning
ARG DB_HOST
ARG DB_NAME
ARG DB_PASSWORD
ARG DB_PORT
ARG DB_SSLMODE
ARG DB_USER
ARG JWT_SECRET
ARG PORT

# Set environment variables in the container runtime from build args
ENV DB_HOST=${DB_HOST}
ENV DB_NAME=${DB_NAME}
ENV DB_PASSWORD=${DB_PASSWORD}
ENV DB_PORT=${DB_PORT}
ENV DB_SSLMODE=${DB_SSLMODE}
ENV DB_USER=${DB_USER}
ENV JWT_SECRET=${JWT_SECRET}
ENV PORT=${PORT}

# Expose port (default 8080, bisa diubah via build arg PORT)
EXPOSE ${PORT:-8080}

# Run the application
CMD ["./main"]
