FROM golang:1.16-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o bin/bot .

FROM alpine
WORKDIR /app
COPY --from=build /src/bin/bot .

ENTRYPOINT ["/app/bot"]