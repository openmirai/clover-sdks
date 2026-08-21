import unittest

from clover_sdk import CloverClient, CloverError, HttpResponse


class FakeTransport:
    def __init__(self, responses):
        self.responses = list(responses)
        self.calls = []

    def __call__(self, method, url, headers, body, timeout):
        self.calls.append((method, url, dict(headers), body))
        return self.responses.pop(0)


class ClientTests(unittest.TestCase):
    def test_invalid_configuration_is_rejected(self):
        with self.assertRaises(ValueError):
            CloverClient("ftp://api.example.test", "secret")
        with self.assertRaises(ValueError):
            CloverClient("https://user:pass@api.example.test", "secret")
        with self.assertRaises(ValueError):
            CloverClient("https://api.example.test?token=leak", "secret")
        with self.assertRaises(ValueError):
            CloverClient("https://api.example.test", "  ")

    def test_idempotency_key_boundaries_are_enforced(self):
        client = CloverClient("https://api.example.test", "secret", transport=FakeTransport([]))
        for key in ("a" * 7, "a" * 129, "a bad-key", "_" + "a" * 8):
            with self.assertRaises(ValueError):
                client.send({"subject": "hello"}, key)
        accepted = CloverClient(
            "https://api.example.test",
            "secret",
            transport=FakeTransport([HttpResponse(202, {}, b"{}"), HttpResponse(202, {}, b"{}")]),
        )
        accepted.send({"subject": "hello"}, "a" * 8)
        accepted.send({"subject": "hello"}, "a" * 128)

    def test_send_headers_and_unknown_response_fields(self):
        transport = FakeTransport(
            [HttpResponse(202, {"X-Request-ID": "req_12345678"}, b'{"success":true,"data":{"id":"e1","status":"queued","extra":true},"requestId":"req_12345678"}')]
        )
        client = CloverClient("https://api.example.test", "secret", transport=transport)
        result = client.send(
            {
                "from": {"address": "a@example.com"},
                "to": [{"address": "b@example.com"}],
                "subject": "hi",
                "text": "body",
            },
            "idem-1234",
        )
        self.assertTrue(result["extra"])
        self.assertEqual(transport.calls[0][2]["Authorization"], "Bearer secret")
        self.assertEqual(transport.calls[0][2]["Idempotency-Key"], "idem-1234")

    def test_bounded_get_retry_and_problem_preservation(self):
        body = (
            b'{"type":"about:blank","title":"Busy","status":503,"code":"BUSY",'
            b'"request_id":"req_12345678","vendor":{"x":1}}'
        )
        transport = FakeTransport([HttpResponse(503, {}, body), HttpResponse(503, {}, body)])
        sleeps = []
        client = CloverClient(
            "https://api.example.test",
            "secret",
            max_retries=1,
            transport=transport,
            sleep=sleeps.append,
        )
        with self.assertRaises(CloverError) as raised:
            client.get("e1")
        self.assertEqual(raised.exception.problem["vendor"]["x"], 1)
        self.assertEqual(len(transport.calls), 2)
        self.assertEqual(sleeps, [0.1])

    def test_email_id_is_encoded_as_one_path_segment(self):
        transport = FakeTransport([HttpResponse(200, {}, b'{"id":"e/1"}')])
        client = CloverClient("https://api.example.test", "secret", transport=transport)
        client.get("e/1 ?#")
        self.assertEqual(transport.calls[0][1], "https://api.example.test/api/v1/emails/e%2F1%20%3F%23")

    def test_list_query_encodes_cursor_and_filter_slashes(self):
        transport = FakeTransport([HttpResponse(200, {}, b'{"data":[]}')])
        client = CloverClient("https://api.example.test", "secret", transport=transport)
        client.list(cursor="next/page", route="a/b")
        self.assertEqual(transport.calls[0][1], "https://api.example.test/api/v1/emails?route=a%2Fb&cursor=next%2Fpage")

    def test_batch_strips_scheduled_at(self):
        transport = FakeTransport([HttpResponse(202, {}, b"{}")])
        client = CloverClient("https://api.example.test", "secret", transport=transport)
        client.send_batch([{"subject": "hello", "scheduled_at": "2030-01-01T00:00:00Z"}], "batch-1234")
        self.assertNotIn("scheduled_at", transport.calls[0][3].decode())

    def test_oversized_response_fails_before_decode(self):
        transport = FakeTransport([HttpResponse(200, {}, b'{"data":"too long"}')])
        client = CloverClient("https://api.example.test", "secret", max_response_body_bytes=8, transport=transport)
        with self.assertRaises(CloverError) as raised:
            client.get("e1")
        self.assertEqual(raised.exception.status, 200)
        self.assertIn("exceeds the configured limit", str(raised.exception))


if __name__ == "__main__":
    unittest.main()
