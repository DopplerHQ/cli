FROM debian:12-slim as build
RUN apt-get update \
    && apt-get install -y \
    tini \
    && rm -rf /var/lib/apt/lists/*

FROM gcr.io/distroless/static-debian12:latest-amd64
COPY --from=build /usr/bin/tini-static /usr/bin/tini-static

COPY doppler /bin/doppler
ENTRYPOINT ["/usr/bin/tini-static", "--", "/bin/doppler"]
