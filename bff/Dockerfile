FROM golang:1.25-alpine AS builder

WORKDIR /app

# copy dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy source code
COPY . .

# build the application
# create the executable "server"
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

FROM gcr.io/distroless/static-debian11

# copy "server"
COPY --from=stage /app/server /

# Listen the port "8080" for Yobitsugi
EXPOSE 8080

# Execute "server"
ENTRYPOINT [ "/server" ]

