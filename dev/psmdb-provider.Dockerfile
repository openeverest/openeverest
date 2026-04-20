FROM alpine AS dev
WORKDIR /home/provider
RUN chown 65534:65534 /home/provider
COPY --chown=65534:65534 ./bin/provider ./provider
USER 65534:65534
ENTRYPOINT ["/home/provider/provider"]
