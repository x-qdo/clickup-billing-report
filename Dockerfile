# --- Go Build Stage ---
FROM public.ecr.aws/docker/library/golang:1.24-bullseye AS go-builder
WORKDIR /clickup-billing-report
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

ARG GOPRIVATE
ENV GOPRIVATE=${GOPRIVATE}

# Copy Go module files first for caching
COPY go.mod go.sum ./
RUN GOPATH=/tmp GOPROXY=${GOPROXY} GOPRIVATE=${GOPRIVATE} go mod download

# Copy the rest of the Go source code and build
# This includes the main.go and any other Go packages.
COPY . .
RUN go build -tags lambda.norpc -o main cmd/api_handler/main.go

# --- Frontend Build Stage ---
FROM public.ecr.aws/docker/library/node:20-bullseye AS fe-builder
WORKDIR /app/frontend

# Copy package.json and package-lock.json (if available) first to leverage Docker cache
COPY frontend/package.json ./
# If package-lock.json is not always present, you might need a more robust copy strategy
# or ensure it's always committed. For now, assuming it exists.
COPY frontend/package-lock.json ./

# Install npm dependencies
RUN npm install

# Copy the rest of the frontend application code
COPY frontend/ ./

# Build the frontend application
RUN npm run build
# This stage will have the compiled assets in /app/frontend/dist

# --- Final Application Stage ---
# Use the AWS Lambda provided base image for Go
FROM public.ecr.aws/lambda/provided:al2023 AS final
ENV CONFIG_PATH "/"

# Copy Go binary from the go-builder stage
COPY --from=go-builder /clickup-billing-report/main /main

# Copy compiled frontend assets from the fe-builder stage
# The assets from /app/frontend/dist in fe-builder are copied to /dist in the final image.
# This makes them available at /dist, sibling to the /main executable.
COPY --from=fe-builder /app/frontend/dist /dist

ENTRYPOINT [ "/main"]
