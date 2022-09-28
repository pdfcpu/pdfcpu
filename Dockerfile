# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:latest as builder

# install
RUN go install github.com/pdfcpu/pdfcpu/cmd/pdfcpu@latest

######## Start a new stage from scratch #######

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /go/bin .

# Command to run the executable
CMD ["./pdfcpu"]
