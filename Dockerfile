FROM node:lts-alpine as frontend

WORKDIR /frontend-build

COPY ./web/package*.json ./
RUN npm install

COPY ./web .
RUN npm run build

FROM golang:1.16 as backend

RUN apt-get update && apt-get install -y gcc musl-dev

WORKDIR /backend-build

COPY go.* ./
RUN go mod download

COPY . .
COPY --from=frontend /frontend-build/public ./web/public

RUN go build -o faucet -ldflags "-s -w"

FROM alpine

FROM golang:1.16

#RUN apk add --no-cache ca-certificates bash

COPY --from=backend /backend-build/faucet /app/faucet

EXPOSE 8080

ENTRYPOINT ["/app/faucet"]
