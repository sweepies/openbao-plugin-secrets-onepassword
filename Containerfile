# OpenBao OCI plugin image: the static binary at the image root, named by
# ENTRYPOINT. The release workflow builds the binaries with
# `mise run release:build` first; nothing is compiled here.
FROM scratch
ARG TARGETARCH
LABEL org.opencontainers.image.source=https://github.com/sweepies/openbao-plugin-secrets-onepassword
LABEL org.opencontainers.image.licenses=MPL-2.0
COPY --chmod=0755 dist/linux_${TARGETARCH}/openbao-plugin-secrets-onepassword /openbao-plugin-secrets-onepassword
ENTRYPOINT ["/openbao-plugin-secrets-onepassword"]
