# frozen_string_literal: true

require "open3"
require "net/http"
require "logger"
require "timeout"

module HatchetWorkerFixture
  LOGGER = Logger.new($stdout)

  # Wait for the worker health check endpoint to respond
  #
  # @param port [Integer] Health check port
  # @param max_attempts [Integer] Maximum number of attempts
  # @return [Boolean] true if healthy
  # @raise [RuntimeError] if worker fails to start
  def self.wait_for_worker_health(port:, max_attempts: 25)
    attempts = 0

    loop do
      if attempts > max_attempts
        raise "Worker failed to start within #{max_attempts} seconds"
      end

      begin
        uri = URI("http://localhost:#{port}/health")
        response = Net::HTTP.get_response(uri)
        return true if response.code == "200"
      rescue StandardError
        # Worker not ready yet
      end

      sleep 1
      attempts += 1
    end
  end

  # Start a worker subprocess and wait for it to be healthy
  #
  # @param command [Array<String>] Command to run
  # @param healthcheck_port [Integer] Port for health checks
  # @yield [pid] Yields the process PID
  # @return [void]
  def self.with_worker(command, healthcheck_port: 8001)
    LOGGER.info("Starting background worker: #{command.join(' ')}")

    ENV["HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT"] = healthcheck_port.to_s

    stdin, stdout, stderr, wait_thr = Open3.popen3(*command)
    pid = wait_thr.pid

    # Log output in background threads
    Thread.new do
      stdout.each_line { |line| puts line.chomp }
    rescue IOError
      # Stream closed
    end

    Thread.new do
      stderr.each_line { |line| $stderr.puts line.chomp }
    rescue IOError
      # Stream closed
    end

    wait_for_worker_health(port: healthcheck_port)

    yield pid
  ensure
    LOGGER.info("Cleaning up background worker (PID: #{pid})")

    if pid
      begin
        # Kill process group to get children too
        Process.kill("TERM", -Process.getpgid(pid))
      rescue Errno::ESRCH, Errno::EPERM
        # Process already gone
      end

      begin
        Timeout.timeout(5) { Process.wait(pid) }
      rescue Timeout::Error
        begin
          Process.kill("KILL", pid)
          Process.wait(pid)
        rescue Errno::ESRCH, Errno::ECHILD
          # Already gone
        end
      rescue Errno::ECHILD
        # Already reaped
      end
    end

    [stdin, stdout, stderr].each do |io|
      io&.close rescue nil
    end
  end
end
