# Build Dependencies ---------------------------
FROM golang:1.23.4-alpine AS build_deps

RUN apk add --no-cache git

WORKDIR /workspace
COPY go.mod .
COPY go.sum .

RUN go mod download

# Build the app --------------------------------
FROM build_deps AS build

COPY . .
RUN CGO_ENABLED=0 go build -o random -ldflags '-w -extldflags "-static"' .

# Package the image ----------------------------
FROM scratch

EXPOSE 777
COPY --from=build /workspace/random /usr/local/bin/random
ENTRYPOINT ["random"]
