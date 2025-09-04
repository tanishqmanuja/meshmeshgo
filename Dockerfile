ARG BUILD_FROM
FROM ${BUILD_FROM}

ARG NOWTALK_SRV_VERSION


# Environment configuration
ENV \
    S6_KILL_GRACETIME=30000 \
    S6_SERVICES_GRACETIME=30000

# Install NODE JS

WORKDIR /usr/src
RUN \
    set -x \
    && apk add --no-cache \
    nodejs \
    npm \
    && apk add --no-cache --virtual .build-dependencies \
    build-base \
    git \
    linux-headers \
    python3 \
    \
    && npm rebuild --build-from-source @serialport/bindings-cpp \
    && apk del --no-cache \
    .build-dependencies

WORKDIR /

ENV PATH=/usr/src/node_modules/.bin:$PATH

COPY /app/package.json package.json
COPY /app/package-lock.json package-lock.json

RUN npm install

COPY /app /

WORKDIR /data

CMD [ "node", "/app.mjs" ]
