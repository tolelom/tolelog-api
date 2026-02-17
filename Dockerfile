FROM alpine:latest
WORKDIR /app
COPY server .
RUN mkdir -p /app/uploads/images
VOLUME ["/app/uploads"]
EXPOSE 8000
CMD ["./server"]
