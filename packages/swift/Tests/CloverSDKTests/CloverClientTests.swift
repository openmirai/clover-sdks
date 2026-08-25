import XCTest
@testable import CloverSDK

private struct StubTransport: CloverTransport {
    let handler: @Sendable (HTTPRequest) async throws -> HTTPResponse

    func send(_ request: HTTPRequest) async throws -> HTTPResponse { try await handler(request) }
}

private actor Recorder {
    var requests: [HTTPRequest] = []
    var attempts = 0
    func append(_ request: HTTPRequest) { requests.append(request) }
    func nextAttempt() -> Int { attempts += 1; return attempts }
}

final class CloverClientTests: XCTestCase {
    func testSendAddsBearerHeadersAndEncodesPathAndRetries() async throws {
        let recorder = Recorder()
        let transport = StubTransport { request in
            await recorder.append(request)
            if await recorder.nextAttempt() == 1 { return HTTPResponse(statusCode: 503, headers: ["Retry-After": "0"]) }
            let body = #"{"id":"email-1","status":"queued","request_id":"req-1"}"#.data(using: .utf8)!
            return HTTPResponse(statusCode: 202, headers: ["X-Request-ID": "req-1"], body: body)
        }
        let client = CloverClient(
            configuration: CloverClientConfiguration(baseURL: URL(string: "https://api.example.test")!, apiKey: "re_test", maxRetries: 1),
            transport: transport,
            sleep: { _ in }
        )
        let response = try await client.schedule(emailID: "a/b", scheduledAt: "2026-08-15T00:00:00Z", idempotencyKey: "idem-1234")
        XCTAssertEqual(response.value.id, "email-1")
        XCTAssertEqual(response.metadata.requestID, "req-1")
        let requests = await recorder.requests
        XCTAssertEqual(requests.count, 2)
        XCTAssertTrue(requests[0].url.absoluteString.contains("a%2Fb"))
        XCTAssertEqual(requests[0].headers["Authorization"], "Bearer re_test")
        XCTAssertEqual(requests[0].headers["Idempotency-Key"], "idem-1234")
        XCTAssertTrue(requests[0].headers["X-Request-ID"]?.hasPrefix("req_") == true)
        XCTAssertEqual(requests[0].headers["X-Request-ID"], requests[1].headers["X-Request-ID"])
    }

    func testBatchOmitsScheduleOnlyField() async throws {
        let recorder = Recorder()
        let transport = StubTransport { request in
            await recorder.append(request)
            let body = #"{"data":[],"request_id":"req-1"}"#.data(using: .utf8)!
            return HTTPResponse(statusCode: 202, body: body)
        }
        let client = CloverClient(
            configuration: CloverClientConfiguration(baseURL: URL(string: "https://api.example.test")!, apiKey: "key"),
            transport: transport,
            sleep: { _ in }
        )
        _ = try await client.sendBatch([
            SendEmailRequest(
                from: EmailAddress(address: "sender@example.com"),
                to: [EmailAddress(address: "user@example.com")],
                subject: "Hello",
                text: "Queued",
                scheduledAt: "2026-08-15T00:00:00Z"
            )
        ], idempotencyKey: "idem-1234")
        let requests = await recorder.requests
        let body = try XCTUnwrap(requests.first?.body)
        let bodyString = try XCTUnwrap(String(data: body, encoding: .utf8))
        XCTAssertFalse(bodyString.contains("scheduled_at"))
    }

    func testMutationRequiresIdempotencyBeforeTransport() async {
        let transport = StubTransport { _ in XCTFail("transport must not be called"); return HTTPResponse(statusCode: 500) }
        let client = CloverClient(configuration: CloverClientConfiguration(baseURL: URL(string: "https://api.example.test")!, apiKey: "key"), transport: transport)
        do {
            _ = try await client.cancel(emailID: "email-1", idempotencyKey: " ")
            XCTFail("expected validation error")
        } catch let error as CloverError {
            XCTAssertEqual(error.statusCode, 0)
            XCTAssertTrue(error.message.contains("idempotencyKey"))
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testMutationRejectsIdempotencyKeysOutsideContractBounds() async {
        let transport = StubTransport { _ in XCTFail("transport must not be called"); return HTTPResponse(statusCode: 500) }
        let client = CloverClient(configuration: CloverClientConfiguration(baseURL: URL(string: "https://api.example.test")!, apiKey: "key"), transport: transport)

        for key in ["short-1", "starts with space", String(repeating: "a", count: 129), "starts#bad"] {
            do {
                _ = try await client.cancel(emailID: "email-1", idempotencyKey: key)
                XCTFail("expected validation error for \(key)")
            } catch let error as CloverError {
                XCTAssertEqual(error.statusCode, 0)
                XCTAssertTrue(error.message.contains("idempotencyKey"))
            } catch {
                XCTFail("unexpected error: \(error)")
            }
        }
    }

    func testProblemPreservesUnknownFieldsAndMetadata() async {
        let body = #"{"type":"https://example.test/problem","title":"Nope","status":422,"code":"invalid","request_id":"req-2","new_extension":{"flag":true}}"#.data(using: .utf8)!
        let transport = StubTransport { _ in HTTPResponse(statusCode: 422, headers: ["X-Request-ID": "req-2", "X-RateLimit-Remaining": "7"], body: body) }
        let client = CloverClient(configuration: CloverClientConfiguration(baseURL: URL(string: "https://api.example.test")!, apiKey: "key"), transport: transport)
        do {
            _ = try await client.get(emailID: "email-1")
            XCTFail("expected problem")
        } catch let error as CloverError {
            XCTAssertEqual(error.problem?.extra["new_extension"], .object(["flag": .boolean(true)]))
            XCTAssertEqual(error.metadata.rateLimitRemaining, 7)
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testNestedV2ErrorEnvelopeIsDecoded() async {
        let body = #"{"success":false,"error":{"code":2002,"type":"VALIDATION_ERROR","message":"Email is invalid","fields":{"email":"invalid"}},"timestamp":"2026-08-25T00:00:00Z","requestId":"req_12345678"}"#.data(using: .utf8)!
        let transport = StubTransport { request in
            XCTAssertTrue(request.headers["X-Request-ID"]?.hasPrefix("req_") == true)
            return HTTPResponse(statusCode: 422, headers: ["X-Request-ID": "req_12345678"], body: body)
        }
        let client = CloverClient(configuration: CloverClientConfiguration(baseURL: URL(string: "https://api.example.test")!, apiKey: "key"), transport: transport)
        do {
            _ = try await client.get(emailID: "email-1")
            XCTFail("expected V2 error")
        } catch let error as CloverError {
            XCTAssertEqual(error.message, "Email is invalid")
            XCTAssertEqual(error.problem?.code, "2002")
            XCTAssertEqual(error.problem?.type, "VALIDATION_ERROR")
            XCTAssertEqual(error.problem?.requestID, "req_12345678")
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testResponseBodyLimitPreservesMetadata() async {
        let transport = StubTransport { _ in
            HTTPResponse(statusCode: 503, headers: ["X-Request-ID": "req-large"], body: Data(repeating: 0x20, count: 32))
        }
        let client = CloverClient(
            configuration: CloverClientConfiguration(
                baseURL: URL(string: "https://api.example.test")!,
                apiKey: "key",
                maxResponseBodyBytes: 8
            ),
            transport: transport
        )
        do {
            _ = try await client.get(emailID: "email-1")
            XCTFail("expected response body limit error")
        } catch let error as CloverError {
            XCTAssertEqual(error.statusCode, 503)
            XCTAssertEqual(error.metadata.requestID, "req-large")
            XCTAssertTrue(error.message.contains("body"))
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testAcceptedPreservesUnknownFields() async throws {
        let body = #"{"id":"email-1","status":"queued","request_id":"req-1","vendor":{"flag":true}}"#.data(using: .utf8)!
        let transport = StubTransport { _ in HTTPResponse(statusCode: 202, body: body) }
        let client = CloverClient(configuration: CloverClientConfiguration(baseURL: URL(string: "https://api.example.test")!, apiKey: "key"), transport: transport)
        let response = try await client.send(
            SendEmailRequest(
                from: EmailAddress(address: "sender@example.com"),
                to: [EmailAddress(address: "user@example.com")],
                subject: "Hello",
                text: "Queued"
            ),
            idempotencyKey: "idem-1234"
        )
        XCTAssertEqual(response.value.extra["vendor"], .object(["flag": .boolean(true)]))
    }
}
