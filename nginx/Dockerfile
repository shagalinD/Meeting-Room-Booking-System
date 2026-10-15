FROM nginx:alpine

RUN apk add --no-cache openssl

COPY entrypoint.sh /usr/local/bin/generate-self-signed-cert
RUN sed -i 's/\r$//' /usr/local/bin/generate-self-signed-cert \
    && chmod 755 /usr/local/bin/generate-self-signed-cert

ENTRYPOINT ["/usr/local/bin/generate-self-signed-cert"]
CMD ["nginx", "-g", "daemon off;"]
