# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:latest as builder

# install
RUN go get github.com/pdfcpu/pdfcpu/cmd/...
WORKDIR $GOPATH/src/github.com/pdfcpu/pdfcpu/cmd/pdfcpu
RUN CGO_ENABLED=0 GOOS=linux go build -a -o pdfcpu .

######## Start a new stage from scratch #######

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /go/src/github.com/pdfcpu/pdfcpu/cmd/pdfcpu .

# Command to run the executable
CMD ["./pdfcpu"]
