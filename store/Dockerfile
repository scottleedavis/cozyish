# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:latest

# Add Maintainer Info
LABEL maintainer="Scott Lee Davis <scottleedavis@gmail.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

RUN date > /root/tmp_variable
RUN SESSION_KEY=$(cat /root/tmp_variable);

# Build the Go app
RUN go build -o main .


# Command to run the executable
CMD sleep 30 && ./main
