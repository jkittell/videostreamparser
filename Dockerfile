FROM golang:alpine as builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o /videostreamparser

FROM busybox
COPY --from=builder /videostreamparser /home
EXPOSE 80
WORKDIR /home
ENTRYPOINT [ "./videostreamparser" ]