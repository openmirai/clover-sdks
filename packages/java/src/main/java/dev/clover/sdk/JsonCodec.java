package dev.clover.sdk;

import java.math.BigDecimal;
import java.math.BigInteger;
import java.util.*;
import java.util.regex.Pattern;

/** Strict dependency-free JSON codec for map-shaped API responses and bodies. */
final class JsonCodec {
  private static final Pattern NUMBER = Pattern.compile("-?(?:0|[1-9][0-9]*)(?:\\.[0-9]+)?(?:[eE][+-]?[0-9]+)?");

  private JsonCodec() {}

  static Object parse(String input) {
    if (input == null || input.isEmpty()) throw new IllegalArgumentException("JSON input is empty");
    Parser parser = new Parser(input);
    Object value = parser.value();
    parser.whitespace();
    if (!parser.atEnd()) throw parser.error("trailing JSON token");
    return value;
  }

  @SuppressWarnings("unchecked")
  static Map<String, Object> object(String input) {
    Object value = parse(input);
    if (!(value instanceof Map<?, ?> map)) throw new IllegalArgumentException("JSON object expected");
    return (Map<String, Object>) map;
  }

  static String stringify(Object value) {
    if (value == null) return "null";
    if (value instanceof String string) return quote(string);
    if (value instanceof Number number) {
      String text = number.toString();
      if (number instanceof Double doubleValue && !Double.isFinite(doubleValue) || number instanceof Float floatValue && !Float.isFinite(floatValue) || !NUMBER.matcher(text).matches()) {
        throw new IllegalArgumentException("non-finite or invalid JSON number");
      }
      return text;
    }
    if (value instanceof Boolean bool) return bool.toString();
    if (value instanceof Map<?, ?> map) {
      StringJoiner joiner = new StringJoiner(",", "{", "}");
      for (Map.Entry<?, ?> entry : map.entrySet()) joiner.add(quote(String.valueOf(entry.getKey())) + ":" + stringify(entry.getValue()));
      return joiner.toString();
    }
    if (value instanceof Iterable<?> iterable) {
      StringJoiner joiner = new StringJoiner(",", "[", "]");
      for (Object item : iterable) joiner.add(stringify(item));
      return joiner.toString();
    }
    return quote(value.toString());
  }

  private static String quote(String value) {
    StringBuilder result = new StringBuilder(value.length() + 2).append('"');
    for (int index = 0; index < value.length();) {
      char character = value.charAt(index);
      if (Character.isHighSurrogate(character)) {
        if (index + 1 >= value.length() || !Character.isLowSurrogate(value.charAt(index + 1))) throw new IllegalArgumentException("unpaired surrogate in JSON string");
        result.appendCodePoint(Character.toCodePoint(character, value.charAt(index + 1)));
        index += 2;
        continue;
      }
      if (Character.isLowSurrogate(character)) throw new IllegalArgumentException("unpaired surrogate in JSON string");
      switch (character) {
        case '"' -> result.append("\\\"");
        case '\\' -> result.append("\\\\");
        case '\b' -> result.append("\\b");
        case '\f' -> result.append("\\f");
        case '\n' -> result.append("\\n");
        case '\r' -> result.append("\\r");
        case '\t' -> result.append("\\t");
        default -> { if (character < 0x20) result.append(String.format("\\u%04x", (int) character)); else result.append(character); }
      }
      index++;
    }
    return result.append('"').toString();
  }

  private static final class Parser {
    private final String input;
    private int index;

    Parser(String input) { this.input = input; }

    Object value() {
      whitespace();
      if (atEnd()) throw error("value expected");
      return switch (input.charAt(index)) {
        case '{' -> objectValue();
        case '[' -> arrayValue();
        case '"' -> stringValue();
        case 't' -> literal("true", Boolean.TRUE);
        case 'f' -> literal("false", Boolean.FALSE);
        case 'n' -> literal("null", null);
        default -> number();
      };
    }

    private Map<String, Object> objectValue() {
      expect('{');
      Map<String, Object> result = new LinkedHashMap<>();
      whitespace();
      if (consume('}')) return result;
      while (true) {
        whitespace();
        if (atEnd() || input.charAt(index) != '"') throw error("object key expected");
        String key = stringValue();
        whitespace();
        expect(':');
        result.put(key, value());
        whitespace();
        if (consume('}')) return result;
        expect(',');
        whitespace();
        if (!atEnd() && input.charAt(index) == '}') throw error("trailing comma");
      }
    }

    private List<Object> arrayValue() {
      expect('[');
      List<Object> result = new ArrayList<>();
      whitespace();
      if (consume(']')) return result;
      while (true) {
        result.add(value());
        whitespace();
        if (consume(']')) return result;
        expect(',');
        whitespace();
        if (!atEnd() && input.charAt(index) == ']') throw error("trailing comma");
      }
    }

    private String stringValue() {
      expect('"');
      StringBuilder result = new StringBuilder();
      while (!atEnd()) {
        char character = input.charAt(index++);
        if (character == '"') return result.toString();
        if (character < 0x20) throw error("unescaped control character in string");
        if (character == '\\') {
          if (atEnd()) throw error("incomplete escape");
          char escaped = input.charAt(index++);
          switch (escaped) {
            case '"', '\\', '/' -> result.append(escaped);
            case 'b' -> result.append('\b');
            case 'f' -> result.append('\f');
            case 'n' -> result.append('\n');
            case 'r' -> result.append('\r');
            case 't' -> result.append('\t');
            case 'u' -> appendUnicodeEscape(result);
            default -> throw error("invalid escape sequence");
          }
        } else if (Character.isSurrogate(character)) {
          if (Character.isLowSurrogate(character) || index >= input.length() || !Character.isLowSurrogate(input.charAt(index))) throw error("unpaired surrogate in string");
          result.append(character).append(input.charAt(index++));
        } else result.append(character);
      }
      throw error("unterminated string");
    }

    private void appendUnicodeEscape(StringBuilder result) {
      char first = unicodeUnit();
      if (Character.isHighSurrogate(first)) {
        if (index + 1 >= input.length() || input.charAt(index) != '\\' || input.charAt(index + 1) != 'u') throw error("high surrogate must be followed by low surrogate");
        index += 2;
        char second = unicodeUnit();
        if (!Character.isLowSurrogate(second)) throw error("invalid low surrogate");
        result.append(first).append(second);
      } else if (Character.isLowSurrogate(first)) throw error("unpaired low surrogate");
      else result.append(first);
    }

    private char unicodeUnit() {
      if (index + 4 > input.length()) throw error("incomplete unicode escape");
      int value = 0;
      for (int count = 0; count < 4; count++) {
        int digit = Character.digit(input.charAt(index++), 16);
        if (digit < 0) throw error("invalid unicode escape");
        value = value * 16 + digit;
      }
      return (char) value;
    }

    private Number number() {
      int start = index;
      if (consume('-')) { if (atEnd()) throw error("invalid number"); }
      if (consume('0')) { if (!atEnd() && asciiDigit(input.charAt(index))) throw error("leading zero in number"); }
      else {
        if (atEnd() || input.charAt(index) < '1' || input.charAt(index) > '9') throw error("invalid number");
        while (!atEnd() && asciiDigit(input.charAt(index))) index++;
      }
      boolean decimal = false;
      if (consume('.')) {
        decimal = true;
        int digits = index;
        while (!atEnd() && asciiDigit(input.charAt(index))) index++;
        if (digits == index) throw error("fraction digits expected");
      }
      if (!atEnd() && (input.charAt(index) == 'e' || input.charAt(index) == 'E')) {
        decimal = true;
        index++;
        if (!atEnd() && (input.charAt(index) == '+' || input.charAt(index) == '-')) index++;
        int digits = index;
        while (!atEnd() && asciiDigit(input.charAt(index))) index++;
        if (digits == index) throw error("exponent digits expected");
      }
      String text = input.substring(start, index);
      try { return decimal ? new BigDecimal(text) : new BigInteger(text).bitLength() < 63 ? Long.valueOf(text) : new BigInteger(text); }
      catch (NumberFormatException error) { throw error("invalid number"); }
    }

    private Object literal(String text, Object value) { if (!input.startsWith(text, index)) throw error("invalid literal"); index += text.length(); return value; }
    private void expect(char expected) { if (!consume(expected)) throw error("expected '" + expected + "'"); }
    private boolean consume(char expected) { if (!atEnd() && input.charAt(index) == expected) { index++; return true; } return false; }
    private boolean asciiDigit(char value) { return value >= '0' && value <= '9'; }
    private void whitespace() { while (!atEnd() && switch (input.charAt(index)) { case ' ', '\t', '\n', '\r' -> true; default -> false; }) index++; }
    private boolean atEnd() { return index >= input.length(); }
    private IllegalArgumentException error(String message) { return new IllegalArgumentException(message + " at character " + index); }
  }
}
