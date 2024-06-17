FROM node:current-alpine AS base
RUN mkdir -p /app
ARG BUILD_VERSION=dev
ENV BUILD_VERSION=$BUILD_VERSION
ARG GIT_REV=HEAD
ENV GIT_REV=$GIT_REV
EXPOSE 8080

FROM base AS development
ENV NODE_ENV=development
COPY package*.json /
RUN npm install
WORKDIR /app
CMD [ "node", "/node_modules/nodemon/bin/nodemon.js" ]

FROM base AS production
ENV NODE_ENV=production
RUN chown node:node /app
RUN mkdir -p /app/node_modules
RUN chown node:node /app/node_modules
USER node
WORKDIR /app
COPY --chown=node:node package*.json .
RUN npm install
COPY --chown=node:node . .
CMD [ "node", "app.js" ]