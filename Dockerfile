FROM golang:alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . . 

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /images-app ./cmd/app/

FROM alpine AS build-release-stage

WORKDIR /

COPY --from=build-stage /images-app /images-app

COPY --from=build-stage /app/images ./images

COPY --from=build-stage /app/.env .

EXPOSE 8090

ENTRYPOINT ["./images-app", "--config=.env", "--env=local"]

