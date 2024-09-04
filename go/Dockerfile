# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY go.mod go.sum ./

# Copy the source code into the container
COPY . .

# Build the Go app using the Makefile
RUN apk add --no-cache make
RUN make api


FROM node:22-alpine as frontend
WORKDIR /app

# Copy the dependencies
COPY ./chat/package.json ./chat/package-lock.json ./chat/
RUN npm --prefix ./chat i


# Copy the source code
COPY ./chat ./chat

# Build the Web app
RUN apk add --no-cache make
RUN make chat


# Stage 2: Create a minimal image with just the binary and the static files
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the builder stage
COPY --from=builder /app/bin/api .
COPY --from=frontend /app/chat/dist ./chat/dist

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./api"]