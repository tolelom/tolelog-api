FROM alpine:latest
WORKDIR /app
COPY server .
EXPOSE 8000
CMD ["./server"]
