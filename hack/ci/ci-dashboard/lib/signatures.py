"""Failure-signature normalization.

A signature collapses a concrete failure into a stable identity so that the same
failure recurring across runs maps to one key. This dedupes both trend counting
and (crucially) LLM classification, which is cached by signature.

    signature = "<workflow> / <job (matrix stripped)> / <failing step> / <normalized error line>"
"""

from __future__ import annotations

import hashlib
import re

_MATRIX = re.compile(r"\s*\(.*\)\s*$")
# GitHub Actions log line prefix, e.g. "2026-06-15T04:53:17.6890901Z "
_TS = re.compile(r"^\d{4}-\d{2}-\d{2}T[\d:.]+Z\s?")
_UUID = re.compile(r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}", re.I)
_HEX = re.compile(r"\b0x[0-9a-f]+\b", re.I)
_DUR = re.compile(r"\b\d+(\.\d+)?(ns|µs|us|ms|s|m|h)\b")
_ADDR = re.compile(r":\d{2,5}\b")
_NUM = re.compile(r"\b\d+(\.\d+)?\b")


def strip_ts(line: str) -> str:
    return _TS.sub("", line).rstrip("\r\n")


def strip_matrix(name: str) -> str:
    return _MATRIX.sub("", name or "").strip()


def normalize(line: str) -> str:
    s = strip_ts(line or "")
    s = _UUID.sub("<uuid>", s)
    s = _HEX.sub("<hex>", s)
    s = _DUR.sub("<dur>", s)
    s = _ADDR.sub(":<port>", s)
    s = _NUM.sub("<n>", s)
    s = re.sub(r"\s+", " ", s).strip()
    return s[:300]


def signature(workflow: str, job_name: str, step: str | None, error_line: str | None) -> str:
    return f"{workflow} / {strip_matrix(job_name)} / {step or '?'} / {normalize(error_line or '')}"


def sig_hash(sig: str) -> str:
    return hashlib.sha1(sig.encode()).hexdigest()[:12]
