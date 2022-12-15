FROM node:current-alpine3.16

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/root/.local/bin

WORKDIR /docusaurus
COPY ./package.json .
VOLUME /docusaurus

run yarn install
