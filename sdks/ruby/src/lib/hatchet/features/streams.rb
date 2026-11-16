# frozen_string_literal: true

require "grpc"
require "securerandom"
require "tempfile"

module Hatchet
  module Features
    # A message read from a durable stream topic.
    #
    # @!attribute payload
    #   @return [String] The message payload, as binary
    # @!attribute cursor
    #   @return [String] Resumes {Streams#subscribe} right after this event
    # @!attribute created_at
    #   @return [Time, nil] When the message was stored
    StreamEvent = Struct.new(:payload, :cursor, :created_at, keyword_init: true)

    # What a durable stream topic retains; counts and the latest message cover the tenant's retention.
    #
    # @!attribute latest_cursor
    #   @return [String, nil] Resumes {Streams#subscribe} after the newest retained message; nil when none is retained
    # @!attribute last_published_at
    #   @return [Time, nil] When the newest retained message was stored; nil when none is retained
    StreamTopicMetadata = Struct.new(
      :namespace, :topic, :tenant_id, :message_count, :latest_cursor, :last_published_at,
      keyword_init: true,
    )

    # Streams client for publishing to and reading from durable topics. Unlike
    # +runs.subscribe_to_stream+, topics are stored, independent of any run, and
    # readers can resume from a cursor.
    #
    # @example Publishing and reading
    #   hatchet.streams.publish("orders", '{"status":"shipped"}')
    #   hatchet.streams.subscribe("orders") { |event| puts event.payload }
    #
    # @since 0.12.0
    class Streams
      # the server's limit for a payload sent over gRPC (MaxStreamMessagePayloadBytes)
      MAX_INLINE_PAYLOAD_BYTES = (4 * 1024 * 1024) - (5 * 1024)

      # the server's limit for an uploaded payload (MaxStreamUploadedPayloadBytes)
      MAX_UPLOADED_PAYLOAD_BYTES = 64 * 1024 * 1024

      # the most publishes one PublishBatch request carries; the engine sets no limit of its own
      MAX_PUBLISH_BATCH_ENTRIES = 500

      # keeps a PublishBatch request under gRPC's 4 MiB limit with the headroom of a single max-size publish
      MAX_PUBLISH_BATCH_BYTES = (4 * 1024 * 1024) - (5 * 1024)

      # an entry's tag and length prefixes, rounded up
      PUBLISH_ENTRY_OVERHEAD_BYTES = 16

      # an upload's ref, which a batch carries in place of its payload, rounded up
      PAYLOAD_REF_BYTES = 256

      Config = Struct.new(:payload_request_attempts, :resubscribe_min_delay_seconds, :resubscribe_max_delay_seconds) do
        def initialize(payload_request_attempts: 3, resubscribe_min_delay_seconds: 0.25, resubscribe_max_delay_seconds: 10.0)
          super
        end
      end

      # the server stored nothing, so the seq is unused
      REJECTED_BEFORE_STORING = [
        GRPC::Core::StatusCodes::INVALID_ARGUMENT,
        GRPC::Core::StatusCodes::RESOURCE_EXHAUSTED,
        GRPC::Core::StatusCodes::UNAUTHENTICATED,
        GRPC::Core::StatusCodes::PERMISSION_DENIED,
        # an engine without the RPC
        GRPC::Core::StatusCodes::UNIMPLEMENTED,
      ].freeze

      # retrying a subscribe that failed with these can't succeed
      PERMANENT_SUBSCRIBE_ERRORS = [
        GRPC::Core::StatusCodes::INVALID_ARGUMENT,
        GRPC::Core::StatusCodes::PERMISSION_DENIED,
        GRPC::Core::StatusCodes::UNAUTHENTICATED,
        GRPC::Core::StatusCodes::OUT_OF_RANGE,
        GRPC::Core::StatusCodes::FAILED_PRECONDITION,
        GRPC::Core::StatusCodes::NOT_FOUND,
      ].freeze

      Producer = Struct.new(:producer_id, :seq)

      # a publish waiting to be sent; +uploaded+ is set instead of an inline payload for one too large for a gRPC message
      QueuedPublish = Struct.new(:payload, :uploaded, :send_by, :result) do
        def request_bytes
          uploaded ? PAYLOAD_REF_BYTES : payload.bytesize
        end
      end

      PublishQueue = Struct.new(:items, :wake)

      # @param rest_client [Object] The configured REST client
      # @param channel [GRPC::Core::Channel] Shared gRPC channel
      # @param config [Hatchet::Config] The Hatchet configuration
      # @param streams_config [Config] Retry and resubscribe settings
      # @return [void]
      def initialize(rest_client, channel, config, streams_config: Config.new)
        @config = config
        @streams_config = streams_config
        @streams_api = HatchetSdkRest::StreamsApi.new(rest_client)
        @stub = ::V1::V1Streams::Stub.new(config.host_port, nil, channel_override: channel)

        @mutex = Mutex.new
        # per [namespace, topic]; seq only advances after a publish succeeds
        @producers = {}
        # per [namespace, topic]; one request per topic is in flight at a time so two never share a seq
        @queues = {}
        # set once the engine answers PublishBatch as unimplemented: it predates batching
        @publish_batch_unsupported = false
      end

      # Publish a message to a topic, returning once it is stored. A client's
      # messages to a topic are delivered in the order +publish+ was called.
      # Publishes to a topic share requests: everything queued while one is in
      # flight goes out in the next, and +batch_timeout+ holds a message up to
      # that long for more to join it.
      #
      # @param topic [String] The topic to publish to
      # @param message [String] The message payload
      # @param namespace [String] The topic's namespace; the empty namespace by default
      # @param batch_timeout [Numeric, nil] Seconds the message may wait for later publishes to the same
      #   topic to share its request, bounding the latency batching adds. By default, it is sent as soon
      #   as no other request to the topic is in flight.
      # @return [void]
      # @raise [ArgumentError] If the payload is empty or too large, or the batch timeout is negative
      # @raise [GRPC::BadStatus] If the engine rejects the publish
      # @example
      #   hatchet.streams.publish("orders", '{"order_id":"ord_123"}', namespace: "customer-a")
      def publish(topic, message, namespace: "", batch_timeout: nil)
        payload = message.to_s.b

        # the engine would reject the whole request it shares, failing other publishes too
        raise ArgumentError, "payload is empty; a message must carry at least one byte" if payload.empty?

        if payload.bytesize > MAX_UPLOADED_PAYLOAD_BYTES
          raise ArgumentError, "payload exceeds maximum size of #{MAX_UPLOADED_PAYLOAD_BYTES} bytes"
        end

        timeout = batch_timeout || 0
        raise ArgumentError, "batch_timeout must be 0 or more; got #{batch_timeout}" if timeout.negative?

        # uploaded before taking its place in the topic's order, so uploads overlap
        uploaded = payload.bytesize > MAX_INLINE_PAYLOAD_BYTES ? start_upload(payload) : nil

        publish = QueuedPublish.new(payload, uploaded, monotonic_now + timeout, Thread::Queue.new)
        key = [namespace, topic]
        start_drain = false

        @mutex.synchronize do
          queue = @queues[key]
          if queue.nil?
            queue = PublishQueue.new([], ConditionVariable.new)
            @queues[key] = queue
            start_drain = true
          end
          queue.items << publish
          # a new publish may fill the request or have a shorter timeout
          queue.wake.signal
          Thread.new { drain_publishes(key, queue) } if start_drain
        end

        status, err = publish.result.pop
        raise err unless status == :ok

        nil
      end

      # Describe a topic's retained messages.
      #
      # @param topic [String] The topic to describe
      # @param namespace [String] The topic's namespace; the empty namespace by default
      # @return [StreamTopicMetadata] The topic's metadata
      # @raise [GRPC::NotFound] For a topic nothing was published to
      def topic_metadata(topic, namespace: "")
        res = @stub.get_topic_metadata(
          ::V1::GetStreamTopicMetadataRequest.new(namespace: namespace, topic: topic),
          metadata: @config.auth_metadata,
        )

        StreamTopicMetadata.new(
          namespace: res.namespace,
          topic: res.topic,
          tenant_id: res.tenant_id,
          message_count: res.message_count,
          latest_cursor: res.has_latest_cursor? ? res.latest_cursor : nil,
          last_published_at: to_time(res.last_published_at),
        )
      end

      # Make {#subscribe} calls without a cursor start after +cursor+, or the
      # topic's latest message. Readers already iterating, and any call with a
      # cursor, are unaffected.
      #
      # @param topic [String] The topic to reset
      # @param namespace [String] The topic's namespace; the empty namespace by default
      # @param cursor [String, nil] A cursor read off one of the topic's events to reset to, instead of its latest message
      # @return [void]
      # @raise [GRPC::BadStatus] If the engine rejects the reset
      def reset(topic, namespace: "", cursor: nil)
        request = { namespace: namespace, topic: topic }
        request[:cursor] = cursor unless cursor.nil?

        @stub.reset(::V1::ResetStreamTopicRequest.new(**request), metadata: @config.auth_metadata)
        nil
      end

      # Read a topic's messages after +cursor+, or from the oldest retained (or,
      # once the topic is reset, after the reset). If the connection drops, it
      # resubscribes from the last message yielded, so nothing is skipped or
      # repeated. Ends when the server hangs up an idle subscription or the
      # block breaks; raises on errors a retry can't fix, such as an expired cursor.
      #
      # @param topic [String] The topic to read from
      # @param namespace [String] The topic's namespace; the empty namespace by default
      # @param cursor [String, nil] A cursor read off a previous event, to resume right after it
      # @yieldparam event [StreamEvent] Each of the topic's events
      # @return [Enumerator<StreamEvent>, nil] An enumerator of the topic's events when no block is given
      # @raise [GRPC::BadStatus] On an error a resubscribe can't fix
      # @example
      #   hatchet.streams.subscribe("orders") do |event|
      #     puts event.payload
      #     break if event.payload == "done"
      #   end
      def subscribe(topic, namespace: "", cursor: nil, &block)
        return enum_for(:subscribe, topic, namespace: namespace, cursor: cursor) unless block

        failures = 0
        attempt = 0

        loop do
          request = { namespace: namespace, topic: topic }
          request[:cursor] = cursor unless cursor.nil?

          op = @stub.subscribe(
            ::V1::SubscribeStreamRequest.new(**request),
            metadata: @config.auth_metadata,
            return_op: true,
          )

          # an error from the caller's block is theirs, not a dropped connection
          delivering = false

          begin
            op.execute.each do |msg|
              return nil if msg.hangup

              msg.entries.each do |entry|
                payload = entry.payload_ref.empty? ? entry.payload : fetch_payload(entry.payload_ref)
                cursor = entry.cursor
                # reset only on delivery, so a payload fetch failing on every reconnect still backs off
                failures = 0
                delivering = true
                block.call(StreamEvent.new(payload: payload, cursor: entry.cursor, created_at: to_time(entry.created_at)))
                delivering = false
              end
            end
            # ending without a hangup means the engine went away, e.g. it is shutting down
          rescue StandardError => e
            raise if delivering || !should_resubscribe?(e, attempt)
          ensure
            op.cancel
          end

          sleep(resubscribe_delay(failures))
          failures += 1
          attempt += 1
        end
      end

      private

      def to_time(timestamp)
        timestamp && Time.at(timestamp.seconds, timestamp.nanos, :nsec).utc
      end

      def monotonic_now
        Process.clock_gettime(Process::CLOCK_MONOTONIC)
      end

      def grpc_code(err)
        err.is_a?(GRPC::BadStatus) ? err.code : nil
      end

      # a payload request that got no response, a 5xx or a 429 may succeed later
      def temporary_http_error?(err)
        code = err.code.to_i
        code.zero? || code >= 500 || code == 429
      end

      def should_resubscribe?(err, attempt)
        return temporary_http_error?(err) if err.is_a?(HatchetSdkRest::ApiError)

        code = grpc_code(err)
        return false if code.nil? || PERMANENT_SUBSCRIBE_ERRORS.include?(code)

        # proxies answer 404 while an engine restarts, but on a first attempt it means no streams support
        return attempt.positive? if code == GRPC::Core::StatusCodes::UNIMPLEMENTED

        true
      end

      def resubscribe_delay(failures)
        delay = [
          @streams_config.resubscribe_min_delay_seconds * (2**[failures, 16].min),
          @streams_config.resubscribe_max_delay_seconds,
        ].min
        delay * (0.5 + (rand / 2))
      end

      # retries a 5xx or network failure; a 4xx won't succeed on retry
      def with_retries
        attempt = 1
        begin
          yield
        rescue HatchetSdkRest::ApiError => e
          code = e.code.to_i
          raise if (code.positive? && code < 500) || attempt >= @streams_config.payload_request_attempts

          attempt += 1
          retry
        end
      end

      def start_upload(payload)
        thread = Thread.new { upload_payload(payload) }
        # its error fails only its own publish, raised when the drain joins it
        thread.report_on_exception = false
        thread
      end

      def upload_payload(payload)
        file = Tempfile.new("hatchet-stream-payload")
        begin
          file.binmode
          file.write(payload)
          file.flush
          with_retries do
            file.rewind
            @streams_api.v1_stream_payload_upload(@config.tenant_id, file).ref
          end
        ensure
          file.close!
        end
      end

      def fetch_payload(payload_ref)
        file = with_retries { @streams_api.v1_stream_payload_get(@config.tenant_id, payload_ref) }
        return file.b if file.is_a?(String)

        begin
          File.binread(file.path)
        ensure
          file.close
          file.unlink
        end
      end

      def drain_publishes(key, queue)
        while (batch = next_batch(key, queue))
          sent = []
          entries = []

          batch.each do |p|
            if p.uploaded.nil?
              entries << ::V1::StreamPublishEntry.new(payload: p.payload)
              sent << p
              next
            end

            # a failed upload fails only its own publish, as the rest never depended on it
            begin
              ref = p.uploaded.value
            rescue StandardError => e
              p.result << [:error, e]
              next
            end

            entries << ::V1::StreamPublishEntry.new(payload_ref: ref)
            sent << p
          end

          send_publishes(key, sent, entries) unless entries.empty?
        end
      end

      # the next request's publishes, or nil once the queue is empty and removed
      def next_batch(key, queue)
        @mutex.synchronize do
          loop do
            if queue.items.empty?
              # removed under the lock publish takes, so a later publish starts a new drain
              @queues.delete(key)
              return nil
            end

            count = fitting_publishes(queue.items)
            full = count < queue.items.length || count == MAX_PUBLISH_BATCH_ENTRIES
            wait = queue.items.map(&:send_by).min - monotonic_now

            return queue.items.shift(count) if full || wait <= 0

            queue.wake.wait(@mutex, wait)
          end
        end
      end

      # always at least one, so a max-size payload still goes out on its own
      def fitting_publishes(items)
        total = 0
        count = 0

        while count < items.length && count < MAX_PUBLISH_BATCH_ENTRIES
          nxt = items[count].request_bytes + PUBLISH_ENTRY_OVERHEAD_BYTES
          break if count.positive? && total + nxt > MAX_PUBLISH_BATCH_BYTES

          total += nxt
          count += 1
        end

        count
      end

      # sends entries as one request, or one request per entry to an engine that
      # predates PublishBatch, and settles each publish
      def send_publishes(key, sent, entries)
        unless @publish_batch_unsupported
          begin
            publish_ordered(key, entries)
            sent.each { |p| p.result << [:ok] }
            return
          rescue StandardError => e
            unless grpc_code(e) == GRPC::Core::StatusCodes::UNIMPLEMENTED
              sent.each { |p| p.result << [:error, e] }
              return
            end
            # nothing was stored, so the batch goes out again below
            @publish_batch_unsupported = true
          end
        end

        # settled one by one: a failure partway leaves the publishes before it stored
        sent.zip(entries).each do |p, entry|
          publish_ordered(key, [entry])
          p.result << [:ok]
        rescue StandardError => e
          p.result << [:error, e]
        end
      end

      def publish_ordered(key, entries, retry_gap: true)
        namespace, topic = key
        producer = @mutex.synchronize { @producers[key] ||= Producer.new(SecureRandom.uuid, 0) }

        begin
          if @publish_batch_unsupported && entries.length == 1
            @stub.publish(
              ::V1::PublishStreamMessageRequest.new(
                namespace: namespace,
                topic: topic,
                payload: entries[0].payload,
                payload_ref: entries[0].payload_ref,
                producer_id: producer.producer_id,
                producer_seq: producer.seq,
              ),
              metadata: @config.auth_metadata,
            )
          else
            @stub.publish_batch(
              ::V1::PublishStreamMessagesRequest.new(
                namespace: namespace,
                topic: topic,
                producer_id: producer.producer_id,
                first_producer_seq: producer.seq,
                entries: entries,
              ),
              metadata: @config.auth_metadata,
            )
          end
        rescue StandardError => e
          code = grpc_code(e)

          # it may still land, and a different payload under its seq would be dropped as a duplicate
          @mutex.synchronize { @producers[key] = Producer.new(SecureRandom.uuid, 0) } unless REJECTED_BEFORE_STORING.include?(code)

          # a gap stored nothing (e.g. the watermark passed cursor retention), so resend as the new producer
          return publish_ordered(key, entries, retry_gap: false) if retry_gap && code == GRPC::Core::StatusCodes::FAILED_PRECONDITION

          raise
        end

        producer.seq += entries.length
      end
    end
  end
end
