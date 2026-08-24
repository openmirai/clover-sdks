#!/usr/bin/env python3
"""Check new Go SDK/CLI platform calls against backend Swagger routes.

The SDK intentionally exposes a subset of the backend.  This check therefore
asserts that every route used by the new platform clients exists with the same
HTTP verb, while allowing path parameters to use the backend's parameter name.
It is deliberately dependency-free so it can run before package installs.
"""

from __future__ import annotations

import ast
import json
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
BACKEND_SWAGGER = ROOT.parent / "clover" / "backend" / "cmd" / "api" / "docs" / "swagger.json"
SOURCES = (
    ROOT / "apps" / "cli" / "client_platform.go",
    ROOT / "packages" / "go" / "resources_platform.go",
)
METHODS = {"GET", "POST", "PUT", "PATCH", "DELETE"}


def split_args(value: str) -> list[str]:
    result: list[str] = []
    start = 0
    depth = 0
    quote: str | None = None
    escaped = False
    for index, char in enumerate(value):
        if quote:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == quote:
                quote = None
            continue
        if char in "'\"":
            quote = char
        elif char == "(":
            depth += 1
        elif char == ")":
            depth -= 1
        elif char == "," and depth == 0:
            result.append(value[start:index].strip())
            start = index + 1
    result.append(value[start:].strip())
    return result


def call_args(text: str, start: int) -> tuple[list[str], int]:
    opening = text.find("(", start)
    if opening < 0:
        raise ValueError("call has no opening parenthesis")
    depth = 0
    quote: str | None = None
    escaped = False
    for index in range(opening, len(text)):
        char = text[index]
        if quote:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == quote:
                quote = None
            continue
        if char in "'\"":
            quote = char
        elif char == "(":
            depth += 1
        elif char == ")":
            depth -= 1
            if depth == 0:
                return split_args(text[opening + 1 : index]), index + 1
    raise ValueError("unterminated call")


def split_plus(value: str) -> list[str]:
    result: list[str] = []
    start = 0
    depth = 0
    quote: str | None = None
    escaped = False
    for index, char in enumerate(value):
        if quote:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == quote:
                quote = None
            continue
        if char in "'\"":
            quote = char
        elif char == "(":
            depth += 1
        elif char == ")":
            depth -= 1
        elif char == "+" and depth == 0:
            result.append(value[start:index].strip())
            start = index + 1
    result.append(value[start:].strip())
    return result


def strip_wrapper(expression: str) -> str:
    expression = expression.strip()
    while True:
        match = re.match(r"^(?:withQuery|withValues|platformPathWithQuery)\((.*)\)$", expression)
        if not match:
            return expression
        expression = split_args(match.group(1))[0]


def path_skeleton(expression: str, variables: dict[str, str] | None = None) -> str:
    variables = variables or {}
    expression = strip_wrapper(expression)
    if expression in variables:
        return variables[expression]
    match = re.match(r"^scope\.path\((.*)\)$", expression)
    if match:
        suffix = split_args(match.group(1))[0]
        return "/api/v1/platform/accounts/{}/environments/{}" + path_skeleton(suffix, variables)
    match = re.match(r"^platformAccountPath\((.*)\)$", expression)
    if match:
        args = split_args(match.group(1))
        return "/api/v1/platform/accounts/{}" + path_skeleton(args[1], variables)
    if expression.startswith("(") and expression.endswith(")"):
        return path_skeleton(expression[1:-1], variables)
    parts = split_plus(expression)
    if len(parts) > 1:
        return "".join(path_skeleton(part, variables) for part in parts)
    try:
        value = ast.literal_eval(expression)
    except (SyntaxError, ValueError):
        value = None
    if isinstance(value, str):
        return value
    if re.match(r"^(?:segment|url\.PathEscape)\(.*\)$", expression):
        return "{}"
    if re.match(r"^[A-Za-z_]\w*$", expression):
        return "{}"
    return expression


def normalized_segments(path: str) -> tuple[str, ...]:
    return tuple(segment for segment in path.split("/") if segment)


def matches(source_path: str, server_path: str) -> bool:
    source = normalized_segments(source_path)
    server = normalized_segments(server_path)
    return len(source) == len(server) and all(
        source_segment == "{}"
        or source_segment == server_segment
        or (server_segment.startswith("{") and server_segment.endswith("}"))
        for source_segment, server_segment in zip(source, server)
    )


def cli_routes(source: Path) -> list[tuple[str, str]]:
    routes: list[tuple[str, str]] = []
    for line in source.read_text(encoding="utf-8").splitlines():
        if "platformRequest(ctx," not in line:
            continue
        args, _ = call_args(line, line.index("platformRequest(ctx,"))
        method = args[1].replace("http.Method", "").upper()
        if method not in METHODS:
            raise SystemExit(f"unsupported HTTP method in {source}: {method}")
        routes.append((method, path_skeleton(args[2])))
    return routes


def function_bodies(source: str) -> list[str]:
    starts = [match.start() for match in re.finditer(r"func \([^\n]+\) [A-Za-z_]\w*\([^\n]*\) [^{]*\{", source)]
    bodies: list[str] = []
    for start in starts:
        opening = source.find("{", start)
        depth = 0
        quote: str | None = None
        escaped = False
        for index in range(opening, len(source)):
            char = source[index]
            if quote:
                if escaped:
                    escaped = False
                elif char == "\\":
                    escaped = True
                elif char == quote:
                    quote = None
                continue
            if char == '"':
                quote = char
            elif char == "{":
                depth += 1
            elif char == "}":
                depth -= 1
                if depth == 0:
                    bodies.append(source[start : index + 1])
                    break
    return bodies


def go_routes(source: Path) -> list[tuple[str, str]]:
    routes: list[tuple[str, str]] = []
    for body in function_bodies(source.read_text(encoding="utf-8")):
        variables: dict[str, str] = {}
        for match in re.finditer(r"\b(path|p)(?:,\s*\w+)?\s*(?::=|=)\s*([^\n]+)", body):
            name, expression = match.group(1), match.group(2).strip().rstrip(";")
            if "platformEnvironmentPath" in expression or "platformAccountPath" in expression:
                arguments = split_args(expression[expression.find("(") + 1 : expression.rfind(")")])
                prefix = "/api/v1/platform/accounts/{}/environments/{}" if "platformEnvironmentPath" in expression else "/api/v1/platform/accounts/{}"
                variables[name] = prefix + path_skeleton(arguments[1], variables)
            else:
                variables[name] = path_skeleton(expression, variables)
        for match in re.finditer(r"requestTyped\w*\(", body):
            args, _ = call_args(body, match.start())
            if len(args) < 4 or "http.Method" not in args[2]:
                continue
            method = args[2].replace("http.Method", "").upper()
            if method not in METHODS:
                raise SystemExit(f"unsupported HTTP method in {source}: {method}")
            routes.append((method, path_skeleton(args[3], variables)))
    return routes


def main() -> int:
    if not BACKEND_SWAGGER.is_file():
        raise SystemExit(f"backend Swagger not found: {BACKEND_SWAGGER}")
    for source in SOURCES:
        if not source.is_file():
            raise SystemExit(f"platform client source not found: {source}")
    swagger = json.loads(BACKEND_SWAGGER.read_text(encoding="utf-8"))
    server_routes = [
        (method.upper(), path)
        for path, operations in swagger.get("paths", {}).items()
        for method in operations
        if method.upper() in METHODS
    ]
    source_routes = sorted(set(cli_routes(SOURCES[0]) + go_routes(SOURCES[1])))
    missing = [
        route
        for route in source_routes
        if not any(method == route[0] and matches(route[1], path) for method, path in server_routes)
    ]
    if missing:
        for method, path in missing:
            print(f"route missing from backend Swagger: {method} {path}")
        return 1
    print(f"platform route parity passed ({len(source_routes)} SDK/CLI route shapes)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
