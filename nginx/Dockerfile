FROM nginxinc/nginx-unprivileged:1.29-alpine3.23

USER root
RUN apk --no-cache upgrade
USER 101

COPY --chown=101:0 lb.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:8081/lb-health" || exit 1
