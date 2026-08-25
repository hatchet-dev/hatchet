# frozen_string_literal: true

# Generates the Ruby SDK reference (fumadocs .mdx pages) from YARD docstrings
# in src/lib/ and RBS signatures in src/sig/.
#
# Output layout mirrors the Python SDK reference:
#   frontend/docs/content/docs/reference/ruby/
#     client.mdx, context.mdx, runnables.mdx
#     feature-clients/<name>.mdx (one per client in lib/hatchet/features/)
#     feature-clients/meta.json
#
# Ownership: this script owns every .mdx file and feature-clients/meta.json in
# the output directory. ruby/meta.json is MERGED, not overwritten: existing
# entry order and "---Separator---" strings are preserved, newly emitted
# top-level pages are appended, and entries whose page no longer exists are
# dropped. The top-level reference/meta.json is never touched.
#
# Output is deterministic: files are parsed in sorted order and every listing
# either follows an explicit order defined here or the (stable) source order.

require "json"
require "yard"

SDK_ROOT = File.expand_path("..", __dir__)
REPO_ROOT = File.expand_path("../..", SDK_ROOT)
OUT_DIR = File.join(REPO_ROOT, "frontend", "docs", "content", "docs", "reference", "ruby")
FEATURES_DIR = File.join(SDK_ROOT, "src", "lib", "hatchet", "features")
SIG_DIR = File.join(SDK_ROOT, "src", "sig", "hatchet")

# ---------------------------------------------------------------------------
# RBS signatures: best-effort type lookup used when a YARD @param/@return tag
# has no type. Keyed as "Hatchet::Features::Runs#get" => { "param" => "String" }.
# ---------------------------------------------------------------------------
module RbsTypes
  @params = {}
  @returns = {}

  class << self
    def load!
      require "rbs"
      Dir[File.join(SIG_DIR, "**", "*.rbs")].sort.each { |f| load_file(f) }
    rescue LoadError, StandardError
      # RBS enrichment is optional; YARD tags carry types for nearly everything.
    end

    def param_type(method_path, param)
      @params.dig(method_path, param)
    end

    def return_type(method_path)
      @returns[method_path]
    end

    private

    def load_file(path)
      parsed = RBS::Parser.parse_signature(File.read(path))
      decls = parsed.is_a?(Array) ? parsed.last : parsed
      decls.each { |decl| walk(decl, []) }
    rescue StandardError
      nil
    end

    def walk(decl, namespace)
      name = decl.respond_to?(:name) ? decl.name.to_s : nil
      case decl
      when RBS::AST::Declarations::Module
        decl.members.each { |m| walk(m, namespace + [name]) }
      when RBS::AST::Declarations::Class
        decl.members.each { |m| record(m, (namespace + [name]).join("::")) }
      end
    end

    def record(member, class_path)
      return unless member.is_a?(RBS::AST::Members::MethodDefinition)

      overload = member.overloads.first or return
      fn = overload.method_type.type
      key = "#{class_path}##{member.name}"
      params = {}
      (fn.required_positionals + fn.optional_positionals).each do |p|
        params[p.name.to_s] = display(p.type) if p.name
      end
      fn.required_keywords.merge(fn.optional_keywords).each do |kname, p|
        params[kname.to_s] = display(p.type)
      end
      @params[key] = params
      @returns[key] = display(overload.method_type.type.return_type)
    rescue StandardError
      nil
    end

    def display(type)
      s = type.to_s
      return nil if s == "untyped" || s == "void"

      s = s.gsub(/\buntyped\b/, "Object")
      s = s.gsub("[", "<").gsub("]", ">")
      s = s.gsub(/([A-Za-z0-9_:>]+)\?/, '\1 | nil')
      s
    end
  end
end

# ---------------------------------------------------------------------------
# Markdown helpers
# ---------------------------------------------------------------------------
module Md
  module_function

  # Render a GFM table with padded columns (prettier-compatible).
  def table(headers, rows)
    return "" if rows.empty?

    widths = headers.each_index.map do |i|
      ([headers[i]] + rows.map { |r| r[i].to_s }).map(&:length).max
    end
    line = ->(cells) { "| #{cells.each_with_index.map { |c, i| c.to_s.ljust(widths[i]) }.join(' | ')} |" }
    sep = "| #{widths.map { |w| '-' * w }.join(' | ')} |"
    ([line.call(headers), sep] + rows.map { |r| line.call(r) }).join("\n")
  end

  # Clean a YARD docstring into markdown prose: resolve {Ref} and +rdoc+
  # markup (outside of code spans), unwrap hard-wrapped lines within paragraphs.
  def clean(text)
    t = text.to_s.gsub(/``([^`\n]+)``/, '`\1`')
    t = t.split(/(`[^`\n]*`)/).map { |seg| seg.start_with?("`") ? seg : resolve_refs(seg) }.join
    t.split(/\n{2,}/).map { |p| ensure_period(p.strip.gsub(/\s*\n\s*/, " ")) }.reject(&:empty?).join("\n\n")
  end

  def resolve_refs(text)
    t = text.gsub(/\+([A-Za-z0-9_:?!]+)\+/, '`\1`')
    t.gsub(/\{([^}\s]+)(?:\s+([^}]+))?\}/) do
      ref = Regexp.last_match(2) || Regexp.last_match(1)
      "`#{ref.sub(/\A#/, '')}`"
    end
  end

  def first_sentence(text)
    ensure_period(clean(text).split(/(?<=\.)\s+/).first.to_s.tr("\n", " "))
  end

  def types(list)
    Array(list).map(&:to_s).join(" \\| ")
  end

  def cell(text)
    ensure_period(clean(text).tr("\n", " ").gsub("|", "\\|"))
  end

  def ensure_period(text)
    return text if text.empty? || text.end_with?(".", "!", "?", ":", ")")

    "#{text}."
  end
end

# ---------------------------------------------------------------------------
# Renderer for YARD method / class objects
# ---------------------------------------------------------------------------
class Renderer
  # Methods that are implementation plumbing, not public API surface.
  EXCLUDE = {
    "Hatchet::Client" => %w[
      initialize rest_client channel dispatcher_grpc admin_grpc event_grpc
      workflow_run_listener admin
    ],
    "Hatchet::Workflow" => %w[initialize to_proto id=],
    "Hatchet::Task" => %w[initialize to_proto call fn],
    "Hatchet::Context" => %w[initialize],
    "Hatchet::DurableContext" => %w[
      initialize eviction_manager action_key durable_event_listener
      invocation_count engine_version
    ],
    "Hatchet::WorkflowRunRef" => %w[initialize],
    "Hatchet::TaskRunRef" => %w[initialize],
  }.tap { |h| h.default = %w[initialize].freeze }.freeze

  # Explicit ordering for curated pages; anything not listed is appended in
  # source order. Feature clients use plain source order.
  PREFERRED_ORDER = {
    "Hatchet::Client" => %w[worker workflow task durable_task batch_task],
    "Hatchet::Workflow" => %w[
      task durable_task batch_task on_failure_task on_success_task
      run run_no_wait run_many run_many_no_wait create_bulk_run_item
      schedule create_cron id
    ],
    "Hatchet::Task" => %w[
      run run_no_wait run_many run_many_no_wait create_bulk_run_item mock_run id
    ],
    "Hatchet::Context" => %w[
      task_output was_skipped? log cancel cancelled? refresh_timeout
      release_slot put_stream task_run_errors get_task_run_error worker
    ],
    "Hatchet::DurableContext" => %w[sleep_for wait_for],
  }.freeze

  def initialize(class_obj)
    @obj = class_obj
  end

  def functions
    order(methods_of(@obj).reject { |m| m.is_attribute? })
  end

  def attributes
    methods_of(@obj)
      .select(&:is_attribute?)
      .reject { |m| m.name.to_s.end_with?("=") }
      .sort_by { |m| m.name.to_s }
  end

  def class_intro
    parts = [Md.clean(@obj.docstring)]
    parts += examples(@obj)
    parts.reject(&:empty?).join("\n\n")
  end

  def bases_line
    sup = @obj.superclass&.path
    return nil if sup.nil? || sup == "Object"

    "Bases: `#{sup}`"
  end

  def summary_table(meths = functions)
    rows = meths.map { |m| ["`#{m.name}`", Md.cell(Md.first_sentence(m.docstring))] }
    Md.table(%w[Name Description], rows)
  end

  def attribute_section(meth)
    ret = meth.tag(:return)
    prose = Md.clean(meth.docstring)
    prose = Md.clean(ret&.text) if prose.empty?
    out = ["#### `#{meth.name}`", prose]
    if ret && !Array(ret.types).empty?
      out << "Returns:"
      out << Md.table(%w[Type Description], [["`#{Md.types(ret.types)}`", Md.cell(ret.text)]])
    end
    out.reject { |s| s.to_s.empty? }.join("\n\n")
  end

  def method_section(meth, heading_level: 4)
    out = ["#{'#' * heading_level} `#{meth.name}`", Md.clean(meth.docstring)]
    out += examples(meth)

    rows = param_rows(meth)
    unless rows.empty?
      out << "Parameters:"
      out << Md.table(["Name", "Type", "Description", "Default"], rows)
    end

    ret_rows = meth.tags(:return).reject { |t| Array(t.types) == ["void"] }.map do |t|
      ["`#{Md.types(t.types)}`", Md.cell(t.text)]
    end
    unless ret_rows.empty?
      out << "Returns:"
      out << Md.table(%w[Type Description], ret_rows)
    end

    # YARD synthesizes empty @raise tags from raise statements; only keep
    # deliberately documented ones.
    raise_rows = meth.tags(:raise)
                     .reject { |t| t.text.to_s.strip.empty? }
                     .map { |t| ["`#{Md.types(t.types)}`", Md.cell(t.text)] }
    unless raise_rows.empty?
      out << "Raises:"
      out << Md.table(%w[Type Description], raise_rows)
    end

    out.reject { |s| s.to_s.empty? }.join("\n\n")
  end

  # Any public method not in the curated order and not excluded is still
  # documented (appended alphabetically by #order); log it so additions to the
  # SDK surface are visible in generator output without requiring an edit here.
  def log_uncurated
    preferred = PREFERRED_ORDER[@obj.path]
    return unless preferred

    functions.reject { |m| preferred.include?(m.name.to_s) }.each do |m|
      puts "auto-included: #{@obj.path}##{m.name} (not in curated order; appended alphabetically)"
    end
  end

  # Render a full "## Class" block: intro, methods table, attributes, functions.
  def class_block(title: @obj.name.to_s)
    log_uncurated
    meths = functions
    attrs = attributes
    out = ["## #{title}"]
    out << bases_line if bases_line
    out << class_intro
    unless meths.empty?
      out << "### Methods"
      out << summary_table(meths)
    end
    unless attrs.empty?
      out << "### Attributes"
      out += attrs.map { |a| attribute_section(a) }
    end
    unless meths.empty?
      out << "### Functions"
      out += meths.map { |m| method_section(m) }
    end
    out.reject { |s| s.to_s.empty? }.join("\n\n")
  end

  private

  def methods_of(obj)
    excluded = EXCLUDE[obj.path]
    obj.meths(inherited: false, scope: :instance)
       .select { |m| m.visibility == :public }
       .reject { |m| excluded.include?(m.name.to_s) }
  end

  def order(meths)
    preferred = PREFERRED_ORDER[@obj.path]
    return meths unless preferred

    listed, rest = meths.partition { |m| preferred.include?(m.name.to_s) }
    listed.sort_by { |m| preferred.index(m.name.to_s) } + rest.sort_by { |m| m.name.to_s }
  end

  def examples(obj)
    obj.tags(:example).map { |t| "```ruby\n#{t.text.strip}\n```" }
  end

  def param_rows(meth)
    tags = meth.tags(:param)
    meth.parameters.flat_map do |(pname, default)|
      next [] if pname.start_with?("&")

      key = pname.sub(/:\z/, "").sub(/\A\*+/, "")
      tag = tags.find { |t| t.name == key }
      splat = pname.start_with?("*")

      # A documented options hash (**opts with @option tags) is expanded into
      # one row per option instead of a single opaque "**opts" row.
      if splat && !meth.tags(:option).empty?
        next option_rows(meth)
      end

      type = tag && !Array(tag.types).empty? ? Md.types(tag.types) : nil
      type ||= RbsTypes.param_type("#{meth.namespace.path}##{meth.name}", key)
      type ||= "Hash" if splat

      default_cell =
        if splat
          "`{}`"
        elsif default.nil?
          "_required_"
        else
          "`#{default}`"
        end

      [["`#{splat ? pname : key}`", type ? "`#{type}`" : "", Md.cell(tag&.text), default_cell]]
    end
  end

  def option_rows(meth)
    meth.tags(:option).map do |t|
      pair = t.pair
      default = pair.respond_to?(:defaults) && pair.defaults&.first
      [
        "`#{pair.name.to_s.delete_prefix(':')}`",
        "`#{Md.types(pair.types)}`",
        Md.cell(pair.text),
        default ? "`#{default}`" : "`nil`",
      ]
    end
  end
end

# ---------------------------------------------------------------------------
# Page assembly
# ---------------------------------------------------------------------------
def frontmatter(title)
  "---\ntitle: \"#{title}\"\n---"
end

def write_page(relpath, sections)
  path = File.join(OUT_DIR, relpath)
  FileUtils.mkdir_p(File.dirname(path))
  File.write(path, sections.reject { |s| s.to_s.empty? }.join("\n\n") + "\n")
  puts "wrote #{relpath}"
end

# Feature client discovery: one page per file in lib/hatchet/features/, named
# after the file (which matches the accessor name on Hatchet::Client).
def feature_pages
  Dir[File.join(FEATURES_DIR, "*.rb")].sort.map do |file|
    basename = File.basename(file, ".rb")
    klass = YARD::Registry.all(:class).find do |c|
      c.namespace.path == "Hatchet::Features" &&
        c.name.to_s.downcase.delete("_") == basename.delete("_") &&
        c.file&.end_with?("features/#{basename}.rb")
    end
    raise "No feature client class found for #{file}" unless klass

    [basename, klass]
  end
end

def feature_title(klass)
  name = klass.name.to_s
  name.match?(/\A[A-Z0-9]+\z/) ? name : name.gsub(/(?<=[a-z0-9])(?=[A-Z])/, " ")
end

def generate_feature_client_pages(pages)
  pages.each do |basename, klass|
    r = Renderer.new(klass)
    title = feature_title(klass)
    intro = [
      r.class_intro,
      "It is available on the main client as `hatchet.#{basename}`.",
    ].reject(&:empty?).join("\n\n")

    write_page(
      File.join("feature-clients", "#{basename}.mdx"),
      [
        frontmatter(title),
        "# #{title} Client",
        intro,
        "Methods:",
        r.summary_table,
        "### Functions",
        *r.functions.map { |m| r.method_section(m) },
      ],
    )
  end

  meta = { "pages" => pages.map(&:first), "title" => "Feature Clients" }
  File.write(File.join(OUT_DIR, "feature-clients", "meta.json"), JSON.pretty_generate(meta) + "\n")
  puts "wrote feature-clients/meta.json"
end

def generate_client_page(pages)
  klass = YARD::Registry.at("Hatchet::Client")
  r = Renderer.new(klass)

  all = klass.meths(inherited: false, scope: :instance).select { |m| m.visibility == :public }
  main_names = Renderer::PREFERRED_ORDER["Hatchet::Client"]
  main = r.functions.select { |m| main_names.include?(m.name.to_s) }

  # Everything public that isn't a main method or excluded plumbing is an
  # "attribute": the feature clients plus config/logger/tenant_id.
  attrs = all.reject do |m|
    main_names.include?(m.name.to_s) ||
      Renderer::EXCLUDE["Hatchet::Client"].include?(m.name.to_s) ||
      m.name.to_s.end_with?("=")
  end.sort_by { |m| m.name.to_s }

  attr_sections = attrs.map do |m|
    ret = m.tag(:return)
    prose = Md.clean(m.docstring)
    prose = Md.clean(ret&.text) if prose.empty?
    feature = pages.find { |_, k| Array(ret&.types).include?(k.path) }
    prose += " See the [#{feature_title(feature[1])} client](./feature-clients/#{feature[0]})." if feature
    "#### `#{m.name}`\n\n#{prose}"
  end

  ctor = klass.meths(inherited: false).find { |m| m.name == :initialize }
  option_rows = ctor.tags(:option).map do |t|
    pair = t.pair
    ["`#{pair.name.to_s.delete_prefix(':')}`", "`#{Md.types(pair.types)}`", Md.cell(pair.text)]
  end

  write_page(
    "client.mdx",
    [
      frontmatter("Client"),
      "# Hatchet Ruby SDK Reference",
      "This is the Ruby SDK reference, documenting methods available for interacting with Hatchet resources. Check out the [user guide](/v1) for an introduction for getting your first tasks running.",
      "## The Hatchet Ruby Client",
      r.class_intro,
      "The constructor accepts keyword options. Anything not passed explicitly is read from `HATCHET_CLIENT_*` environment variables.",
      "Options:",
      Md.table(%w[Name Type Description], option_rows),
      "Methods:",
      r.summary_table(main),
      "### Attributes",
      *attr_sections,
      "### Functions",
      *main.map { |m| r.method_section(m) },
    ],
  )
end

def generate_context_page
  context = Renderer.new(YARD::Registry.at("Hatchet::Context"))
  durable = Renderer.new(YARD::Registry.at("Hatchet::DurableContext"))

  write_page(
    "context.mdx",
    [
      frontmatter("Context"),
      "# Context",
      <<~INTRO.strip,
        The Hatchet Context class provides helper methods and useful data to tasks at runtime. It is passed as the second argument to all task blocks.

        There are two types of context classes you'll encounter:

        - `Hatchet::Context` - The standard context for regular tasks with methods for logging, task output retrieval, cancellation, and more.
        - `Hatchet::DurableContext` - An extended context for durable tasks that includes additional methods for durable execution like `sleep_for` and `wait_for`.
      INTRO
      context.class_block(title: "Context"),
      durable.class_block(title: "DurableContext"),
    ],
  )
end

def generate_runnables_page
  blocks = %w[Hatchet::Workflow Hatchet::Task Hatchet::WorkflowRunRef Hatchet::TaskRunRef].map do |path|
    Renderer.new(YARD::Registry.at(path)).class_block
  end

  write_page(
    "runnables.mdx",
    [
      frontmatter("Runnables"),
      "# Runnables",
      <<~INTRO.strip,
        Runnables in the Hatchet Ruby SDK are things that can be run, namely tasks and workflows. The two main types of runnables you'll encounter are:

        - `Hatchet::Workflow`, which lets you define tasks and call all of the run, schedule, etc. methods
        - `Hatchet::Task`, which is a single task returned by `hatchet.task` (standalone) or `workflow.task`, and can be run, scheduled, etc.

        Triggering methods that don't wait for a result return run references - `WorkflowRunRef` and `TaskRunRef` - which are also documented below.
      INTRO
      *blocks,
    ],
  )
end

# Merge ruby/meta.json rather than overwriting it: keep existing entry order
# and any "---Separator---" strings, drop entries whose page no longer exists,
# and append newly emitted top-level pages (sorted) that aren't listed yet.
def merge_ruby_meta
  path = File.join(OUT_DIR, "meta.json")
  meta = File.exist?(path) ? JSON.parse(File.read(path)) : { "pages" => [], "title" => "Ruby SDK" }

  emitted = Dir[File.join(OUT_DIR, "*.mdx")].map { |f| File.basename(f, ".mdx") }
  emitted << "feature-clients" if File.exist?(File.join(OUT_DIR, "feature-clients", "meta.json"))
  emitted.sort!

  kept = (meta["pages"] || []).select { |p| p.start_with?("---") || emitted.include?(p) }
  meta["pages"] = kept + (emitted - kept)

  File.write(path, JSON.pretty_generate(meta) + "\n")
  puts "wrote meta.json (merged)"
end

# Backstop: every emitted .mdx must be reachable from a meta.json pages array,
# otherwise fumadocs silently drops it from the sidebar.
def assert_all_pages_reachable
  top_pages = JSON.parse(File.read(File.join(OUT_DIR, "meta.json")))["pages"]
  fc_pages = JSON.parse(File.read(File.join(OUT_DIR, "feature-clients", "meta.json")))["pages"]

  unreachable = Dir[File.join(OUT_DIR, "**", "*.mdx")].sort.reject do |f|
    base = File.basename(f, ".mdx")
    if File.dirname(f) == OUT_DIR
      top_pages.include?(base)
    else
      top_pages.include?("feature-clients") && fc_pages.include?(base)
    end
  end
  return if unreachable.empty?

  abort "ERROR: emitted pages not reachable from any meta.json pages array: #{unreachable.join(', ')}"
end

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
require "fileutils"

YARD::Logger.instance.io = File.open(File::NULL, "w")
YARD::Registry.clear

sources = [File.join(SDK_ROOT, "src", "lib", "hatchet-sdk.rb")] +
          Dir[File.join(SDK_ROOT, "src", "lib", "hatchet", "*.rb")].sort +
          Dir[File.join(FEATURES_DIR, "*.rb")].sort
YARD.parse(sources)

RbsTypes.load!

# The generator owns every .mdx file and both meta.json files (ruby/meta.json
# via merge); remove stale output so the directory is exactly generator output.
Dir[File.join(OUT_DIR, "**", "*.mdx")].sort.each { |f| File.delete(f) }
FileUtils.mkdir_p(File.join(OUT_DIR, "feature-clients"))

pages = feature_pages
generate_client_page(pages)
generate_context_page
generate_runnables_page
generate_feature_client_pages(pages)
merge_ruby_meta
assert_all_pages_reachable

puts "done: #{Dir[File.join(OUT_DIR, '**', '*')].count { |f| File.file?(f) }} files in #{OUT_DIR}"
