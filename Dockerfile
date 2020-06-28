FROM golang:1.14 as go
RUN mkdir -p /app
WORKDIR /app
COPY go.mod /app/
COPY imports.go /app/
RUN go mod vendor
COPY ./ /app/
RUN CGO_ENABLED=0 go build

FROM scratch

COPY --from=go /app/iam4apis /app/

COPY --from=go /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app

ENV POSTGRES_URL="postgresql://postgres:postgres@sql/iam4apis"
ENV ADMIN_USER="pujo.j@sfeir.com"
EXPOSE 4300

CMD ["/app/iam4apis"]