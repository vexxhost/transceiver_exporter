FROM scratch
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/transceiver_exporter /bin/transceiver_exporter
EXPOSE 9459
ENTRYPOINT ["/bin/transceiver_exporter"]
