FROM alpine as deployment

# install bash via apk
RUN apk update && apk add --no-cache bash gcc musl-dev openssl bash ca-certificates curl postgresql-client

RUN curl -sSf https://atlasgo.sh | sh

COPY ./hack/db/atlas-apply.sh ./atlas-apply.sh
COPY ./sql/migrations ./sql/migrations

RUN chmod +x ./atlas-apply.sh

# Run the entrypoint script
CMD ["./atlas-apply.sh"]
