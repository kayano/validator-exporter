FROM gcr.io/distroless/static-debian12:nonroot

COPY validator-exporter /usr/bin/validator-exporter

# metrics server
EXPOSE 8008

ENTRYPOINT [ "/usr/bin/validator-exporter" ]

CMD [ "--help" ]
