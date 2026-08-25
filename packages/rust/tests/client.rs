use clover_sdk::{CloverClient, JsonValue, RawResponse, Transport};
use std::collections::BTreeMap;
use std::sync::{Arc, Mutex};

struct Fake {
    calls: Arc<Mutex<usize>>,
}

struct SingleResponse {
    response: Option<RawResponse>,
}
struct OneShot(Mutex<SingleResponse>);
impl Transport for OneShot {
    fn send(
        &self,
        _method: &str,
        _url: &str,
        _headers: &BTreeMap<String, String>,
        _body: Option<&str>,
    ) -> Result<RawResponse, String> {
        Ok(self
            .0
            .lock()
            .unwrap()
            .response
            .take()
            .expect("one response"))
    }
}

type CapturedRequest = (String, String, BTreeMap<String, String>);
struct Capture {
    requests: Arc<Mutex<Vec<CapturedRequest>>>,
}
impl Transport for Capture {
    fn send(
        &self,
        method: &str,
        url: &str,
        headers: &BTreeMap<String, String>,
        _body: Option<&str>,
    ) -> Result<RawResponse, String> {
        self.requests
            .lock()
            .unwrap()
            .push((method.into(), url.into(), headers.clone()));
        Ok(RawResponse {
            status: 200,
            headers: BTreeMap::new(),
            body: "{}".into(),
        })
    }
}
impl Transport for Fake {
    fn send(
        &self,
        _method: &str,
        _url: &str,
        _headers: &BTreeMap<String, String>,
        _body: Option<&str>,
    ) -> Result<RawResponse, String> {
        let mut calls = self.calls.lock().unwrap();
        *calls += 1;
        if *calls == 1 {
            Ok(RawResponse { status: 503, headers: BTreeMap::new(), body: r#"{"type":"about:blank","title":"Busy","status":503,"code":"BUSY","request_id":"req_12345678","vendor":{"x":1}}"#.into() })
        } else {
            Ok(RawResponse {
                status: 202,
                headers: BTreeMap::new(),
                body: r#"{"id":"e1","status":"queued","extra":true}"#.into(),
            })
        }
    }
}

#[test]
fn send_retries_and_preserves_unknown_fields() {
    let calls = Arc::new(Mutex::new(0));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(Fake {
            calls: calls.clone(),
        }),
        Box::new(|_| {}),
    );
    let result = client
        .send(
            JsonValue::object([(String::from("subject"), "hello".into())]),
            "idem-1234",
        )
        .unwrap();
    assert!(
        matches!(result, JsonValue::Object(ref object) if object.get("extra") == Some(&JsonValue::Bool(true)))
    );
    assert_eq!(*calls.lock().unwrap(), 2);
}

#[test]
fn strict_json_rejects_trailing_input_and_preserves_unicode() {
    let transport = OneShot(Mutex::new(SingleResponse {
        response: Some(RawResponse {
            status: 200,
            headers: BTreeMap::new(),
            body: r#"{"text":"\b\f\n\r\t\uD834\uDD1E","unknown":{"kept":true}} trailing"#.into(),
        }),
    }));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(transport),
        Box::new(|_| {}),
    );
    let error = client
        .get("e1")
        .expect_err("trailing JSON must be rejected");
    assert_eq!(error.status, 200);
    assert_eq!(error.message, "Clover returned malformed JSON");

    let transport = OneShot(Mutex::new(SingleResponse {
        response: Some(RawResponse {
            status: 200,
            headers: BTreeMap::new(),
            body: r#"{"text":"\b\f\n\r\t\uD834\uDD1E","unknown":{"kept":true}}"#.into(),
        }),
    }));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(transport),
        Box::new(|_| {}),
    );
    let result = client.get("e1").expect("valid JSON");
    let JsonValue::Object(object) = result else {
        panic!("object expected")
    };
    assert_eq!(
        object.get("unknown").and_then(|value| match value {
            JsonValue::Object(value) => value.get("kept"),
            _ => None,
        }),
        Some(&JsonValue::Bool(true))
    );
    assert_eq!(
        JsonValue::String("\u{0000}\u{0001}\u{001f}\u{0008}\u{000c}\n\r\t".into()).to_json(),
        r#""\u0000\u0001\u001f\b\f\n\r\t""#
    );
    assert!(JsonValue::Number("01".into()).try_to_json().is_err());

    let transport = OneShot(Mutex::new(SingleResponse {
        response: Some(RawResponse {
            status: 200,
            headers: BTreeMap::new(),
            body: r#"{"bad":"\uD800"}"#.into(),
        }),
    }));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(transport),
        Box::new(|_| {}),
    );
    assert_eq!(
        client
            .get("e1")
            .expect_err("lone surrogate must be rejected")
            .message,
        "Clover returned malformed JSON"
    );
}

#[test]
fn replay_header_value_is_case_insensitive() {
    let transport = OneShot(Mutex::new(SingleResponse {
        response: Some(RawResponse {
            status: 400,
            headers: BTreeMap::from([("Idempotency-Replayed".into(), "TrUe".into())]),
            body: r#"{"type":"about:blank","title":"Bad request"}"#.into(),
        }),
    }));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(transport),
        Box::new(|_| {}),
    );
    let error = client.get("e1").expect_err("problem response expected");
    assert!(error.metadata.replayed);
}

#[test]
#[should_panic(expected = "api_key")]
fn rejects_empty_api_key() {
    let _ = CloverClient::with_transport(
        "https://api.example.test",
        "",
        Box::new(Fake {
            calls: Arc::new(Mutex::new(0)),
        }),
        Box::new(|_| {}),
    );
}

#[test]
#[should_panic(expected = "absolute http(s) URL")]
fn rejects_invalid_base_url() {
    let _ = CloverClient::with_transport(
        "ftp://api.example.test",
        "secret",
        Box::new(Fake {
            calls: Arc::new(Mutex::new(0)),
        }),
        Box::new(|_| {}),
    );
}

#[test]
#[should_panic(expected = "absolute http(s) URL")]
fn rejects_base_url_userinfo_or_query() {
    let _ = CloverClient::with_transport(
        "https://user:pass@api.example.test?token=leak",
        "secret",
        Box::new(Fake {
            calls: Arc::new(Mutex::new(0)),
        }),
        Box::new(|_| {}),
    );
}

#[test]
fn default_transport_accepts_https_configuration() {
    let _ = CloverClient::new("https://api.example.test", "secret");
}

#[test]
fn idempotency_key_boundaries_are_enforced() {
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(Fake {
            calls: Arc::new(Mutex::new(0)),
        }),
        Box::new(|_| {}),
    );
    for key in [
        "a".repeat(7),
        "a".repeat(129),
        "a bad-key".into(),
        format!("_{}", "a".repeat(8)),
    ] {
        assert!(client
            .send(
                JsonValue::object([(String::from("subject"), "hello".into())]),
                &key
            )
            .is_err());
    }
    assert!(client
        .send(
            JsonValue::object([(String::from("subject"), "hello".into())]),
            &"a".repeat(8)
        )
        .is_ok());
    assert!(client
        .send(
            JsonValue::object([(String::from("subject"), "hello".into())]),
            &"a".repeat(128)
        )
        .is_ok());
}

#[test]
fn batch_strips_scheduled_at() {
    let transport = OneShot(Mutex::new(SingleResponse {
        response: Some(RawResponse {
            status: 202,
            headers: BTreeMap::new(),
            body: "{}".into(),
        }),
    }));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(transport),
        Box::new(|_| {}),
    );
    let item = JsonValue::object([
        (String::from("subject"), "hello".into()),
        (String::from("scheduled_at"), "2030-01-01".into()),
    ]);
    assert!(client.send_batch(vec![item], "batch-1234").is_ok());
}

#[test]
fn all_routes_use_one_api_prefix_and_originate_request_ids() {
    let requests = Arc::new(Mutex::new(Vec::new()));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(Capture {
            requests: requests.clone(),
        }),
        Box::new(|_| {}),
    );
    client.send_batch(Vec::new(), "batch-1234").unwrap();
    client
        .schedule("email/1", "2030-01-01T00:00:00Z", "schedule-1234")
        .unwrap();
    client.cancel("email/1", "cancel-1234").unwrap();
    client.get("email/1").unwrap();

    let requests = requests.lock().unwrap();
    let urls = requests
        .iter()
        .map(|(_, url, _)| url.as_str())
        .collect::<Vec<_>>();
    assert_eq!(
        urls,
        vec![
            "https://api.example.test/api/v1/emails/batch",
            "https://api.example.test/api/v1/emails/email%2F1/schedule",
            "https://api.example.test/api/v1/emails/email%2F1/cancel",
            "https://api.example.test/api/v1/emails/email%2F1",
        ]
    );
    for (_, _, headers) in requests.iter() {
        let request_id = headers.get("x-request-id").expect("request id header");
        assert!(request_id.starts_with("req_"));
        assert!(request_id.len() >= 12);
    }
}

#[test]
fn oversized_response_fails_before_json_decode() {
    let transport = OneShot(Mutex::new(SingleResponse {
        response: Some(RawResponse {
            status: 200,
            headers: BTreeMap::new(),
            body: "x".repeat(clover_sdk::DEFAULT_MAX_RESPONSE_BODY_BYTES + 1),
        }),
    }));
    let client = CloverClient::with_transport(
        "https://api.example.test",
        "secret",
        Box::new(transport),
        Box::new(|_| {}),
    );
    let error = client.get("e1").expect_err("oversized body must fail");
    assert_eq!(error.status, 200);
    assert_eq!(
        error.message,
        "Clover response body exceeds the configured limit"
    );
}
