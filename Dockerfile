FROM alpine:3.7
EXPOSE 8888
RUN mkdir /db
VOLUME ["/db"]
ADD templates /templates/
ADD teambynumbers /
ENTRYPOINT ["./teambynumbers"]

