import Foundation

#if canImport(FoundationNetworking)
import FoundationNetworking
#endif

/// A JSON value used for extension fields and attachment metadata.
public enum JSONValue: Codable, Equatable, Sendable {
    case string(String)
    case number(Double)
    case boolean(Bool)
    case object([String: JSONValue])
    case array([JSONValue])
    case null

    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() { self = .null; return }
        if let value = try? container.decode(Bool.self) { self = .boolean(value); return }
        if let value = try? container.decode(Int.self) { self = .number(Double(value)); return }
        if let value = try? container.decode(Double.self) { self = .number(value); return }
        if let value = try? container.decode(String.self) { self = .string(value); return }
        if let value = try? container.decode([String: JSONValue].self) { self = .object(value); return }
        self = .array(try container.decode([JSONValue].self))
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case let .string(value): try container.encode(value)
        case let .number(value): try container.encode(value)
        case let .boolean(value): try container.encode(value)
        case let .object(value): try container.encode(value)
        case let .array(value): try container.encode(value)
        case .null: try container.encodeNil()
        }
    }
}

public struct EmailAddress: Codable, Equatable, Sendable {
    public let address: String
    public let name: String?

    public init(address: String, name: String? = nil) {
        self.address = address
        self.name = name
    }
}

public struct SendEmailRequest: Codable, Equatable, Sendable {
    public let from: EmailAddress
    public let to: [EmailAddress]
    public let cc: [EmailAddress]?
    public let bcc: [EmailAddress]?
    public let replyTo: [EmailAddress]?
    public let subject: String
    public let html: String?
    public let text: String?
    public let attachments: [JSONValue]?
    public let headers: [String: String]?
    public let tags: [String: String]?
    public let scheduledAt: String?

    public init(
        from: EmailAddress,
        to: [EmailAddress],
        cc: [EmailAddress]? = nil,
        bcc: [EmailAddress]? = nil,
        replyTo: [EmailAddress]? = nil,
        subject: String,
        html: String? = nil,
        text: String? = nil,
        attachments: [JSONValue]? = nil,
        headers: [String: String]? = nil,
        tags: [String: String]? = nil,
        scheduledAt: String? = nil
    ) {
        self.from = from
        self.to = to
        self.cc = cc
        self.bcc = bcc
        self.replyTo = replyTo
        self.subject = subject
        self.html = html
        self.text = text
        self.attachments = attachments
        self.headers = headers
        self.tags = tags
        self.scheduledAt = scheduledAt
    }

    enum CodingKeys: String, CodingKey, CaseIterable {
        case from, to, cc, bcc, subject, html, text, attachments, headers, tags
        case replyTo = "reply_to"
        case scheduledAt = "scheduled_at"
    }
}

private struct BatchEmailItem: Encodable {
    let from: EmailAddress
    let to: [EmailAddress]
    let cc: [EmailAddress]?
    let bcc: [EmailAddress]?
    let replyTo: [EmailAddress]?
    let subject: String
    let html: String?
    let text: String?
    let attachments: [JSONValue]?
    let headers: [String: String]?
    let tags: [String: String]?

    init(_ request: SendEmailRequest) {
        from = request.from; to = request.to; cc = request.cc; bcc = request.bcc; replyTo = request.replyTo
        subject = request.subject; html = request.html; text = request.text; attachments = request.attachments
        headers = request.headers; tags = request.tags
    }

    enum CodingKeys: String, CodingKey {
        case from, to, cc, bcc, subject, html, text, attachments, headers, tags
        case replyTo = "reply_to"
    }
}

public struct EmailSummary: Codable, Equatable, Sendable {
    public let id: String
    public let status: String
    public let scheduledAt: String?
    public let requestID: String?
    public let extra: [String: JSONValue]

    enum CodingKeys: String, CodingKey, CaseIterable {
        case id, status
        case scheduledAt = "scheduled_at"
        case requestID = "request_id"
    }

    public init(id: String, status: String, scheduledAt: String? = nil, requestID: String? = nil, extra: [String: JSONValue] = [:]) {
        self.id = id; self.status = status; self.scheduledAt = scheduledAt; self.requestID = requestID; self.extra = extra
    }

    public init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        id = try values.decode(String.self, forKey: .id)
        status = try values.decode(String.self, forKey: .status)
        scheduledAt = try values.decodeIfPresent(String.self, forKey: .scheduledAt)
        requestID = try values.decodeIfPresent(String.self, forKey: .requestID)
        let all = try decoder.singleValueContainer().decode([String: JSONValue].self)
        let known = Set(CodingKeys.allCases.map(\.stringValue))
        extra = all.filter { !known.contains($0.key) }
    }

    public func encode(to encoder: Encoder) throws {
        var value = extra
        value["id"] = .string(id); value["status"] = .string(status)
        if let scheduledAt { value["scheduled_at"] = .string(scheduledAt) }
        if let requestID { value["request_id"] = .string(requestID) }
        try value.encode(to: encoder)
    }
}

public typealias EmailAccepted = EmailSummary

public struct EmailBatchAccepted: Codable, Equatable, Sendable {
    public let data: [EmailSummary]
    public let requestID: String?
    public let extra: [String: JSONValue]

    enum CodingKeys: String, CodingKey, CaseIterable {
        case data
        case requestID = "request_id"
    }

    public init(data: [EmailSummary], requestID: String? = nil, extra: [String: JSONValue] = [:]) {
        self.data = data; self.requestID = requestID; self.extra = extra
    }

    public init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        data = try values.decode([EmailSummary].self, forKey: .data)
        requestID = try values.decodeIfPresent(String.self, forKey: .requestID)
        let all = try decoder.singleValueContainer().decode([String: JSONValue].self)
        let known = Set(CodingKeys.allCases.map(\.stringValue))
        extra = all.filter { !known.contains($0.key) }
    }

    public func encode(to encoder: Encoder) throws {
        var value = extra
        value["data"] = .array(data.map { summary in
            var encoded = summary.extra
            encoded["id"] = .string(summary.id); encoded["status"] = .string(summary.status)
            if let scheduledAt = summary.scheduledAt { encoded["scheduled_at"] = .string(scheduledAt) }
            if let requestID = summary.requestID { encoded["request_id"] = .string(requestID) }
            return .object(encoded)
        })
        if let requestID { value["request_id"] = .string(requestID) }
        try value.encode(to: encoder)
    }
}

/// The response shape returned by list and detail operations. Unknown fields
/// are retained in `extra` so SDK upgrades do not discard server data.
public struct EmailPage: Codable, Equatable, Sendable {
    public let data: [JSONValue]
    public let nextCursor: String?
    public let extra: [String: JSONValue]

    enum CodingKeys: String, CodingKey { case data, nextCursor = "next_cursor" }

    public init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        data = try values.decodeIfPresent([JSONValue].self, forKey: .data) ?? []
        nextCursor = try values.decodeIfPresent(String.self, forKey: .nextCursor)
        let all = try decoder.singleValueContainer().decode([String: JSONValue].self)
        extra = all.filter { !["data", "next_cursor"].contains($0.key) }
    }

    public func encode(to encoder: Encoder) throws {
        var value = extra
        value["data"] = .array(data)
        if let nextCursor { value["next_cursor"] = .string(nextCursor) }
        try value.encode(to: encoder)
    }
}

public struct EmailDetail: Codable, Equatable, Sendable {
    public let value: [String: JSONValue]

    public init(from decoder: Decoder) throws {
        value = try decoder.singleValueContainer().decode([String: JSONValue].self)
    }

    public func encode(to encoder: Encoder) throws { try value.encode(to: encoder) }
}

public struct ListEmailsOptions: Sendable, Equatable {
    public var cursor: String?
    public var limit: Int?
    public var status: String?
    public var domainID: String?
    public var apiKeyID: String?
    public var requestID: String?
    public var createdAfter: String?
    public var createdBefore: String?
    public var additional: [String: String]

    public init(
        cursor: String? = nil,
        limit: Int? = nil,
        status: String? = nil,
        domainID: String? = nil,
        apiKeyID: String? = nil,
        requestID: String? = nil,
        createdAfter: String? = nil,
        createdBefore: String? = nil,
        additional: [String: String] = [:]
    ) {
        self.cursor = cursor; self.limit = limit; self.status = status
        self.domainID = domainID; self.apiKeyID = apiKeyID; self.requestID = requestID
        self.createdAfter = createdAfter; self.createdBefore = createdBefore
        self.additional = additional
    }

    var queryItems: [String: String] {
        var result = additional
        if let cursor { result["cursor"] = cursor }
        if let limit { result["limit"] = String(limit) }
        if let status { result["status"] = status }
        if let domainID { result["domain_id"] = domainID }
        if let apiKeyID { result["api_key_id"] = apiKeyID }
        if let requestID { result["request_id"] = requestID }
        if let createdAfter { result["created_after"] = createdAfter }
        if let createdBefore { result["created_before"] = createdBefore }
        return result
    }
}

public struct ResponseMetadata: Sendable, Equatable {
    public let statusCode: Int
    public let requestID: String?
    public let retryAfter: TimeInterval?
    public let rateLimitRemaining: Int?
    public let replayed: Bool

    public init(statusCode: Int, requestID: String? = nil, retryAfter: TimeInterval? = nil, rateLimitRemaining: Int? = nil, replayed: Bool = false) {
        self.statusCode = statusCode; self.requestID = requestID; self.retryAfter = retryAfter
        self.rateLimitRemaining = rateLimitRemaining; self.replayed = replayed
    }
}

public struct CloverResponse<Value: Codable & Sendable>: Sendable {
    public let value: Value
    public let metadata: ResponseMetadata

    public init(value: Value, metadata: ResponseMetadata) { self.value = value; self.metadata = metadata }
}

public struct ProblemDocument: Codable, Error, Equatable, Sendable {
    public let type: String?
    public let title: String?
    public let status: Int?
    public let code: String?
    public let detail: String?
    public let requestID: String?
    public let fieldErrors: [String: [String]]?
    public let extra: [String: JSONValue]

    enum CodingKeys: String, CodingKey, CaseIterable {
        case type, title, status, code, detail
        case requestID = "request_id"
        case fieldErrors = "field_errors"
    }

    public init(from decoder: Decoder) throws {
        let values = try decoder.container(keyedBy: CodingKeys.self)
        type = try values.decodeIfPresent(String.self, forKey: .type)
        title = try values.decodeIfPresent(String.self, forKey: .title)
        status = try values.decodeIfPresent(Int.self, forKey: .status)
        code = try values.decodeIfPresent(String.self, forKey: .code)
        detail = try values.decodeIfPresent(String.self, forKey: .detail)
        requestID = try values.decodeIfPresent(String.self, forKey: .requestID)
        fieldErrors = try values.decodeIfPresent([String: [String]].self, forKey: .fieldErrors)
        let all = try decoder.singleValueContainer().decode([String: JSONValue].self)
        let known = Set(CodingKeys.allCases.map(\.stringValue))
        extra = all.filter { !known.contains($0.key) }
    }

    public func encode(to encoder: Encoder) throws {
        var value = extra
        if let type { value["type"] = .string(type) }; if let title { value["title"] = .string(title) }
        if let status { value["status"] = .number(Double(status)) }; if let code { value["code"] = .string(code) }
        if let detail { value["detail"] = .string(detail) }; if let requestID { value["request_id"] = .string(requestID) }
        if let fieldErrors { value["field_errors"] = .object(fieldErrors.mapValues { .array($0.map(JSONValue.string)) }) }
        try value.encode(to: encoder)
    }
}

public struct CloverError: Error, LocalizedError, Sendable {
    public let statusCode: Int
    public let problem: ProblemDocument?
    public let metadata: ResponseMetadata
    public let message: String

    public var errorDescription: String? { message }

    public init(statusCode: Int, problem: ProblemDocument?, metadata: ResponseMetadata, message: String? = nil) {
        self.statusCode = statusCode; self.problem = problem; self.metadata = metadata
        self.message = message ?? problem?.title ?? "Clover request failed (\(statusCode))"
    }
}

public struct HTTPRequest: Sendable {
    public let method: String
    public let url: URL
    public let headers: [String: String]
    public let body: Data?

    public init(method: String, url: URL, headers: [String: String], body: Data? = nil) {
        self.method = method; self.url = url; self.headers = headers; self.body = body
    }
}

public struct HTTPResponse: Sendable {
    public let statusCode: Int
    public let headers: [String: String]
    public let body: Data

    public init(statusCode: Int, headers: [String: String] = [:], body: Data = Data()) {
        self.statusCode = statusCode; self.headers = headers; self.body = body
    }
}

public enum CloverTransportError: Error, LocalizedError, Sendable {
    case responseBodyTooLarge(limit: Int)

    public var errorDescription: String? {
        switch self {
        case let .responseBodyTooLarge(limit):
            return "Clover response body exceeds the configured limit of \(limit) bytes"
        }
    }
}

public protocol CloverTransport: Sendable {
    func send(_ request: HTTPRequest) async throws -> HTTPResponse
}

public struct URLSessionTransport: CloverTransport {
    public let session: URLSession
    public let maxResponseBodyBytes: Int

    public init(session: URLSession = .shared, maxResponseBodyBytes: Int = CloverClientConfiguration.defaultMaxResponseBodyBytes) {
        self.session = session
        self.maxResponseBodyBytes = max(1, maxResponseBodyBytes)
    }

    public func send(_ request: HTTPRequest) async throws -> HTTPResponse {
        var urlRequest = URLRequest(url: request.url)
        urlRequest.httpMethod = request.method
        urlRequest.httpBody = request.body
        request.headers.forEach { urlRequest.setValue($1, forHTTPHeaderField: $0) }
        let (bytes, response) = try await session.bytes(for: urlRequest)
        guard let response = response as? HTTPURLResponse else { throw URLError(.badServerResponse) }
        var data = Data()
        data.reserveCapacity(min(maxResponseBodyBytes, 64 * 1024))
        for try await byte in bytes {
            guard data.count < maxResponseBodyBytes else {
                throw CloverTransportError.responseBodyTooLarge(limit: maxResponseBodyBytes)
            }
            data.append(byte)
        }
        var headers: [String: String] = [:]
        response.allHeaderFields.forEach { headers[String(describing: $0.key)] = String(describing: $0.value) }
        return HTTPResponse(statusCode: response.statusCode, headers: headers, body: data)
    }
}

public struct CloverClientConfiguration: Sendable {
    public static let defaultMaxResponseBodyBytes = 4 * 1024 * 1024

    public let baseURL: URL
    public let apiKey: String
    public let userAgent: String
    public let maxRetries: Int
    public let retryBaseDelay: TimeInterval
    public let maxResponseBodyBytes: Int

    public init(baseURL: URL, apiKey: String, userAgent: String = "clover-sdk-swift/0.1.0", maxRetries: Int = 2, retryBaseDelay: TimeInterval = 0.1, maxResponseBodyBytes: Int = CloverClientConfiguration.defaultMaxResponseBodyBytes) {
        guard let components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              ["http", "https"].contains(scheme),
              let host = components.host, !host.isEmpty,
              components.user == nil, components.password == nil,
              components.query == nil, components.fragment == nil else {
            preconditionFailure("baseURL must be an absolute http(s) URL without credentials, query, or fragment")
        }
        guard !apiKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            preconditionFailure("apiKey is required")
        }
        self.baseURL = baseURL; self.apiKey = apiKey; self.userAgent = userAgent
        self.maxRetries = min(3, max(0, maxRetries)); self.retryBaseDelay = max(0, retryBaseDelay)
        self.maxResponseBodyBytes = max(1, maxResponseBodyBytes)
    }
}

public final class CloverClient: Sendable {
    private let configuration: CloverClientConfiguration
    private let transport: any CloverTransport
    private let sleep: @Sendable (TimeInterval) async -> Void
    private static let retryable: Set<Int> = [408, 425, 429, 500, 502, 503, 504]

    public init(configuration: CloverClientConfiguration, transport: any CloverTransport = URLSessionTransport(), sleep: @escaping @Sendable (TimeInterval) async -> Void = CloverClient.defaultSleep) {
        self.configuration = configuration; self.transport = transport; self.sleep = sleep
    }

    public func send(_ request: SendEmailRequest, idempotencyKey: String) async throws -> CloverResponse<EmailAccepted> {
        try await perform("POST", path: "/v1/emails", body: request, idempotencyKey: idempotencyKey, decode: EmailAccepted.self)
    }

    public func sendBatch(_ items: [SendEmailRequest], idempotencyKey: String) async throws -> CloverResponse<EmailBatchAccepted> {
        try await perform("POST", path: "/v1/emails/batch", body: ["items": items.map(BatchEmailItem.init)], idempotencyKey: idempotencyKey, decode: EmailBatchAccepted.self)
    }

    public func schedule(emailID: String, scheduledAt: String, idempotencyKey: String) async throws -> CloverResponse<EmailAccepted> {
        try await perform("POST", path: "/v1/emails/\(Self.encodePathSegment(emailID))/schedule", body: ["scheduled_at": scheduledAt], idempotencyKey: idempotencyKey, decode: EmailAccepted.self)
    }

    public func cancel(emailID: String, idempotencyKey: String) async throws -> CloverResponse<EmailSummary> {
        try await perform("POST", path: "/v1/emails/\(Self.encodePathSegment(emailID))/cancel", idempotencyKey: idempotencyKey, decode: EmailSummary.self)
    }

    public func get(emailID: String) async throws -> CloverResponse<EmailDetail> {
        try await perform("GET", path: "/v1/emails/\(Self.encodePathSegment(emailID))", decode: EmailDetail.self)
    }

    public func list(options: ListEmailsOptions = .init()) async throws -> CloverResponse<EmailPage> {
        let query = options.queryItems.sorted { $0.key < $1.key }.map { "\(Self.encodeQueryComponent($0.key))=\(Self.encodeQueryComponent($0.value))" }.joined(separator: "&")
        let path = query.isEmpty ? "/v1/emails" : "/v1/emails?\(query)"
        return try await perform("GET", path: path, decode: EmailPage.self)
    }

    private func perform<Body: Encodable, Value: Decodable & Sendable>(_ method: String, path: String, body: Body? = nil, idempotencyKey: String? = nil, decode: Value.Type) async throws -> CloverResponse<Value> {
        let encoded = try body.map { try JSONEncoder().encode($0) }
        return try await perform(method, path: path, data: encoded, idempotencyKey: idempotencyKey, decode: decode)
    }

    private func perform<Value: Decodable & Sendable>(_ method: String, path: String, idempotencyKey: String? = nil, decode: Value.Type) async throws -> CloverResponse<Value> {
        try await perform(method, path: path, data: nil, idempotencyKey: idempotencyKey, decode: decode)
    }

    private func perform<Value: Decodable & Sendable>(_ method: String, path: String, data: Data?, idempotencyKey: String?, decode: Value.Type) async throws -> CloverResponse<Value> {
        if method != "GET" && !(idempotencyKey.map(Self.isValidIdempotencyKey) ?? false) {
            throw CloverError(statusCode: 0, problem: nil, metadata: .init(statusCode: 0), message: "idempotencyKey must be 8-128 ASCII characters matching [A-Za-z0-9][A-Za-z0-9._:-]{7,127}")
        }
        var attempt = 0
        while true {
            let url = try Self.makeURL(base: configuration.baseURL, path: path)
            var headers = ["Accept": "application/json, application/problem+json", "Authorization": "Bearer \(configuration.apiKey)", "User-Agent": configuration.userAgent]
            if data != nil { headers["Content-Type"] = "application/json" }
            if let idempotencyKey { headers["Idempotency-Key"] = idempotencyKey }
            let response: HTTPResponse
            do { response = try await transport.send(HTTPRequest(method: method, url: url, headers: headers, body: data)) }
            catch { throw error }
            let metadata = Self.metadata(response)
            if response.body.count > configuration.maxResponseBodyBytes {
                throw CloverError(statusCode: response.statusCode, problem: nil, metadata: metadata, message: "Clover response body exceeds the configured limit")
            }
            if (200..<300).contains(response.statusCode) {
                do { return CloverResponse(value: try JSONDecoder().decode(Value.self, from: response.body), metadata: metadata) }
                catch { throw CloverError(statusCode: response.statusCode, problem: nil, metadata: metadata, message: "Clover returned an invalid response: \(error.localizedDescription)") }
            }
            if (method == "GET" || idempotencyKey != nil), Self.retryable.contains(response.statusCode), attempt < configuration.maxRetries {
                let delay = metadata.retryAfter ?? configuration.retryBaseDelay * pow(2, Double(attempt)); attempt += 1
                await sleep(delay); continue
            }
            let problem = Self.decodeProblem(response.body)
            throw CloverError(statusCode: response.statusCode, problem: problem, metadata: metadata)
        }
    }

    public static let defaultSleep: @Sendable (TimeInterval) async -> Void = { seconds in
        if seconds > 0 { try? await Task.sleep(nanoseconds: UInt64(seconds * 1_000_000_000)) }
    }

    private static func metadata(_ response: HTTPResponse) -> ResponseMetadata {
        let headers = response.headers.reduce(into: [String: String]()) { result, entry in
            result[entry.key.lowercased()] = entry.value
        }
        let retry = headers["retry-after"].flatMap { value -> TimeInterval? in
            guard let seconds = TimeInterval(value), seconds >= 0 else { return nil }
            return seconds
        }
        let remaining = headers["x-ratelimit-remaining"].flatMap { Int($0) }
        return ResponseMetadata(statusCode: response.statusCode, requestID: headers["x-request-id"], retryAfter: retry, rateLimitRemaining: remaining, replayed: headers["idempotency-replayed"]?.lowercased() == "true")
    }

    private static func isValidIdempotencyKey(_ value: String) -> Bool {
        let scalars = Array(value.unicodeScalars)
        guard (8...128).contains(scalars.count),
              let first = scalars.first,
              (first.value >= 48 && first.value <= 57) ||
                  (first.value >= 65 && first.value <= 90) ||
                  (first.value >= 97 && first.value <= 122) else { return false }
        return scalars.dropFirst().allSatisfy { scalar in
            (scalar.value >= 48 && scalar.value <= 57) ||
                (scalar.value >= 65 && scalar.value <= 90) ||
                (scalar.value >= 97 && scalar.value <= 122) ||
                "._:-".unicodeScalars.contains(scalar)
        }
    }

    private static func decodeProblem(_ body: Data) -> ProblemDocument? {
        guard let problem = try? JSONDecoder().decode(ProblemDocument.self, from: body),
              problem.type != nil, problem.title != nil, problem.status != nil, problem.code != nil else { return nil }
        return problem
    }

    private static func makeURL(base: URL, path: String) throws -> URL {
        guard let url = URL(string: base.absoluteString.trimmingCharacters(in: CharacterSet(charactersIn: "/")) + path) else { throw URLError(.badURL) }
        return url
    }

    private static func encodePathSegment(_ value: String) -> String { encode(value, allowed: CharacterSet(charactersIn: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~")) }
    private static func encodeQueryComponent(_ value: String) -> String { encode(value, allowed: CharacterSet(charactersIn: "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~")) }
    private static func encode(_ value: String, allowed: CharacterSet) -> String { value.addingPercentEncoding(withAllowedCharacters: allowed) ?? value }
}
