# Dockerfile
FROM golang:latest
RUN mkdir -p /go/src/app
WORKDIR /go/src/app

# this will ideally be built by the ONBUILD below ;)
CMD ["go-wrapper", "run"]

COPY . /go/src/app
#RUN go-wrapper download
#RUN go-wrapper install
EXPOSE 5000

