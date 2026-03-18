FROM alpine:latest
RUN apk add --no-cache curl
WORKDIR /app
COPY server .
RUN mkdir -p /app/uploads/images
VOLUME ["/app/uploads"]
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8000/api/v1/health || exit 1
CMD ["./server"]
