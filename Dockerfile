FROM golang:1.20

WORKDIR /app

COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /kok-bot

# Run
CMD ["/kok-bot"]