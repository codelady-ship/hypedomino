FROM postgres:16-bookworm
ARG WALG_VERSION=v3.0.9
ARG WALG_SHA256=4f03ee4679db7f660bfd1b2b8291ac5df247cd2ec48074b60c112f4897299f24
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl \
 && curl -sSfL -o /tmp/wal-g.tar.gz "https://github.com/wal-g/wal-g/releases/download/${WALG_VERSION}/wal-g-pg-22.04-amd64.tar.gz" \
 && echo "${WALG_SHA256}  /tmp/wal-g.tar.gz" | sha256sum -c - \
 && tar -xzf /tmp/wal-g.tar.gz -C /tmp \
 && install -m 0755 /tmp/wal-g-pg-22.04-amd64 /usr/local/bin/wal-g \
 && rm -rf /tmp/wal-g* /var/lib/apt/lists/*
