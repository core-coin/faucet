FROM node:lts-alpine as frontend

WORKDIR /frontend-build

COPY ./web/package*.json ./
RUN npm install

COPY ./web .
RUN npm run build

FROM golang:1.18 as backend

RUN apt-get update && apt-get install -y gcc musl-dev

WORKDIR /backend-build

ARG GIT_ENERGY_USERNAME
ARG GIT_ENERGY_PASSWORD
RUN echo "machine git.energy\nlogin $GIT_ENERGY_USERNAME\npassword $GIT_ENERGY_PASSWORD\n" > ~/.netrc
RUN go env -w GOPRIVATE=.energy

COPY go.* ./
RUN go mod download

COPY . .
COPY --from=frontend /frontend-build/public ./web/public

RUN go build -o faucet -ldflags "-s -w"

FROM alpine

FROM golang:1.18

#RUN apk add --no-cache ca-certificates bash

COPY --from=backend /backend-build/faucet /app/faucet

EXPOSE 8080

CMD /app/faucet \
    -logger.level "$LOGGER_LEVEL" \
    -wallet.privkey "$PRIVATE_KEY" \
    -wallet.provider "$WEB3_PROVIDER" \
    -postgres.url "$POSTGRES_URL" \
    -kyc.request.url "$KYC_REQUEST_URL" \
    -callback.url "$CALLBACK_URL" \
    -registry.address "$REGISTRY_ADDRESS"