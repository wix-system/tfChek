FROM golang:latest

# Add Maintainer Info
LABEL maintainer="Maksym Shkolnyi <maksymsh@wix.com>"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o tfChek .

#Add user
RUN addgroup --system deployer && adduser --system --ingroup deployer --uid 5500 deployer
USER deployer

# Expose port 8080 to the outside world
EXPOSE 8085

# Command to run the executable
#CMD ["./tfChek"]

ENTRYPOINT [ "./tfChek.sh" ]

