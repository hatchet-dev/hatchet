# Base Ruby environment
# ---------------------
FROM ruby:3.2 AS deployment

WORKDIR /hatchet/sdks/ruby

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

COPY sdks/ruby/src/ src/
RUN cd src && bundle lock --add-platform x86_64-linux && bundle install

COPY sdks/ruby/examples/ examples/
RUN cd examples && bundle lock --add-platform x86_64-linux && bundle install

COPY build/package/e2e-ruby-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

CMD ["/entrypoint.sh"]
