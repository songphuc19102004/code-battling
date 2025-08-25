FROM songphuc191004/compilers:latest AS builder

ENV PATH="/usr/local/go-1.24.6/bin:/usr/local/python-3.13.6/bin:${PATH}"

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/server ./cmd/main


FROM songphuc191004/compilers:latest AS product

RUN useradd -u 1000 -m -r codebatt&& \
    echo "codebatt ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers

WORKDIR /app

COPY --from=builder /bin/server ./

RUN chmod +x ./server

USER codebatt

EXPOSE 8081

CMD ["./server"]
