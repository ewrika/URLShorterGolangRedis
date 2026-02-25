FROM golang:alpine as builder

RUN mkdir /build

ADD . /build

WORKDIR /build

RUN go build -o main .

FROM alpine

RUN adduser -S -D -H -h /app appuser

USER appuser

WORKDIR /app/

# Копируем бинарный файл
COPY --from=builder /build/main .

# Копируем .env файл (если он существует)
COPY .env .

EXPOSE 3000

CMD ["./main"]