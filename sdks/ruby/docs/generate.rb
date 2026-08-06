#!/usr/bin/env ruby
# frozen_string_literal: true

# Generates MDX reference docs for the Ruby SDK into
# frontend/docs/pages/reference/ruby, mirroring the Python (mkdocs) and
# TypeScript (typedoc) SDK doc generators. Uses YARD via the yard-markdown
# plugin, then post-processes the markdown into MDX-safe pages with Nextra
# _meta.js navigation.
#
# Usage: cd sdks/ruby && ruby docs/generate.rb

require "fileutils"

SCRIPT_DIR = File.expand_path(__dir__)
SDK_SRC_DIR = File.expand_path("../src", SCRIPT_DIR)
REPO_ROOT = File.expand_path("../../..", SCRIPT_DIR)
OUTPUT_DIR = File.join(REPO_ROOT, "frontend/docs/pages/reference/ruby")
TMP_ROOT = "/tmp/hatchet-ruby/docs"
TMP_GEN_PATH = File.join(TMP_ROOT, "gen")
GEM_HOME = File.join(TMP_ROOT, "gems")
YARD_MARKDOWN_VERSION = "0.7.3"

def sh!(cmd)
  system({ "GEM_HOME" => GEM_HOME, "GEM_PATH" => GEM_HOME }, cmd) || abort("command failed: #{cmd}")
end

def kebab(name)
  name
    .gsub(/([A-Z]+)([A-Z][a-z])/, '\1-\2')
    .gsub(/([a-z\d])([A-Z])/, '\1-\2')
    .downcase
end

YARD_TAG_LABELS = {
  "param" => "Parameter",
  "return" => "Returns",
  "yield" => "Yields",
  "yieldparam" => "Yield parameter",
  "yieldreturn" => "Yield returns",
  "raise" => "Raises",
  "see" => "See",
  "note" => "Note",
  "option" => "Option",
  "deprecated" => "Deprecated",
}.freeze

def humanize_yard_tags(line)
  line
    .gsub(/\*\*@example (.*?)\*\*/) { "**Example: #{Regexp.last_match(1)}**" }
    .gsub(/\*\*@(\w+)\*\*/) { "**#{YARD_TAG_LABELS[Regexp.last_match(1)] || Regexp.last_match(1).capitalize}**" }
end

YARD_LINK_RE = /\{(#?[A-Za-z_][A-Za-z0-9_:#.?!]*)\}/

def escape_mdx(line)
  humanize_yard_tags(line).split(/(`[^`]*`)/).map do |segment|
    if segment.start_with?("`")
      segment
    else
      segment
        .gsub(YARD_LINK_RE, '`\1`')
        .gsub(/(?<!\\)[<{}]/) { |c| "\\#{c}" }
    end
  end.join
end

PARAM_RE = /^- \*\*@param\*\* `([^`]*)` \[([^\]]*)\](.*)$/

def param_tables(lines)
  out = []
  in_fence = false
  i = 0

  while i < lines.length
    line = lines[i]

    if line.lstrip.start_with?("```")
      in_fence = !in_fence
    end

    unless !in_fence && line.match?(PARAM_RE)
      out << line
      i += 1
      next
    end

    rows = []
    while i < lines.length && (m = lines[i].match(PARAM_RE))
      desc = m[3].strip
      i += 1
      while i < lines.length && lines[i] =~ /^\s+\S/ && !lines[i].lstrip.start_with?("-", "#", "```")
        desc += " #{lines[i].strip}"
        i += 1
      end
      rows << [m[1], m[2], desc.gsub("|", "\\|")]
    end

    out << "\n" unless out.last.nil? || out.last.strip.empty?
    out << "**Parameters:**\n" << "\n"
    out << "| Name | Type | Description |\n" << "| --- | --- | --- |\n"
    rows.each { |name, type, desc| out << "| `#{name}` | `#{type}` | #{desc} |\n" }
    out << "\n"
  end

  out
end

# Promote member headings to H2 so they appear in the sidebar (the Nextra
# theme only lists depth-2 headings), with the short name as the heading and
# the full signature below it.
def promote_member_heading(line)
  m = line.match(/^### `([^(`]+)(\([^`]*\))?`(.*)$/)
  return line unless m

  suffix = m[3]
  note = nil
  if suffix =~ /\s*\[(RW?)\]/
    note = Regexp.last_match(1) == "RW" ? "_Read-write attribute_\n\n" : "_Read-only attribute_\n\n"
    suffix = suffix.sub(/\s*\[RW?\]/, "")
  end

  heading = "## `#{m[1]}`#{suffix}\n"
  signature = m[2] ? "\n`#{m[1]}#{m[2]}`\n\n" : "\n"
  "#{heading}#{signature}#{note || ''}"
end

def to_mdx(content)
  in_fence = false
  param_tables(content.each_line.to_a).map do |line|
    if line.lstrip.start_with?("```")
      in_fence = !in_fence
      next line
    end
    next line if in_fence
    next nil if line.strip == "Not documented."

    promote_member_heading(escape_mdx(line.gsub(%r{\s*<a id="[^"]*"></a>}, "")))
  end.compact.join
end

def generate_markdown
  FileUtils.rm_rf(TMP_GEN_PATH)

  unless File.exist?(File.join(GEM_HOME, "bin", "yardoc"))
    sh!("gem install yard-markdown --version #{YARD_MARKDOWN_VERSION} --no-document --install-dir #{GEM_HOME}")
  end

  Dir.chdir(SDK_SRC_DIR) do
    sh!(
      "#{GEM_HOME}/bin/yardoc --plugin yard-markdown -f markdown " \
      "-o #{TMP_GEN_PATH} -b #{TMP_ROOT}/.yardoc --no-progress " \
      "--exclude lib/hatchet/contracts --exclude lib/hatchet/clients/rest " \
      "'lib/hatchet/**/*.rb'"
    )
  end
end

def write_mdx_pages
  FileUtils.rm_rf(OUTPUT_DIR)

  Dir.glob(File.join(TMP_GEN_PATH, "**/*.md")).sort.map do |src|
    parts = src.delete_prefix("#{TMP_GEN_PATH}/").delete_suffix(".md").split("/")
    title = parts.last

    is_root_index = parts == ["Hatchet"]
    if is_root_index
      parts = ["index"]
    else
      parts.shift if parts.first == "Hatchet"
      parts << "index" if Dir.exist?(src.delete_suffix(".md"))
    end

    out_path = File.join(OUTPUT_DIR, *parts.map { |p| kebab(p) }) + ".mdx"
    FileUtils.mkdir_p(File.dirname(out_path))

    if is_root_index
      generated_sections = File.read(src).sub(/\A.*?(?=^## )/m, "")
      header = File.read(File.join(SCRIPT_DIR, "index_header.mdx"))
      File.write(out_path, header + "\n" + to_mdx(generated_sections))
    else
      File.write(out_path, to_mdx(File.read(src)))
    end

    { dir: File.dirname(out_path), key: File.basename(out_path, ".mdx"), title: title }
  end
end

def meta_entry(key, title)
  <<~ENTRY
    "#{key}": {
        title: "#{title}",
        theme: {
          toc: true,
        },
      },
  ENTRY
end

def write_meta_js(docs)
  dirs = docs.map { |d| d[:dir] }.uniq

  dirs.each do |dir|
    entries = docs
      .select { |d| d[:dir] == dir }
      .map { |d| [d[:key], d[:key] == "index" ? "Overview" : d[:title]] }

    dirs
      .select { |other| File.dirname(other) == dir }
      .each { |sub| entries << [File.basename(sub), File.basename(sub).split("-").map(&:capitalize).join(" ")] }

    core_order = %w[workflow task worker context durable-context config connection features clients]
    entries.sort_by! do |key, _|
      if key == "index"
        [0, 0, key]
      elsif (rank = core_order.index(key))
        [1, rank, key]
      elsif key.end_with?("-error")
        [3, 0, key]
      else
        [2, 0, key]
      end
    end

    body = entries.map { |key, title| meta_entry(key, title) }.join("  ").rstrip
    File.write(File.join(dir, "_meta.js"), "export default {\n  #{body}\n};\n")
  end
end

generate_markdown
docs = write_mdx_pages
write_meta_js(docs)
FileUtils.rm_rf(TMP_GEN_PATH)
puts "Wrote #{docs.size} pages to #{OUTPUT_DIR}"
