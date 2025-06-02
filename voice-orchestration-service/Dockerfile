FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copiar los archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -o voice-orchestration-service .

# Imagen final
FROM gcr.io/distroless/static:nonroot

WORKDIR /app

# Copiar el binario compilado
COPY --from=builder /app/voice-orchestration-service .

# Exponer el puerto
EXPOSE 8080

# Ejecutar la aplicación
CMD ["/app/voice-orchestration-service"]
