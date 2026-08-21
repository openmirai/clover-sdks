//! A small Clover client. The default transport uses ureq with rustls for
//! HTTPS; an injected `Transport` remains available for deterministic tests.
use std::collections::BTreeMap;
use std::fmt;
use std::io::Read;
use std::time::Duration;

#[derive(Clone, Debug, PartialEq)]
pub enum JsonValue {
    Null,
    Bool(bool),
    Number(String),
    String(String),
    Array(Vec<JsonValue>),
    Object(BTreeMap<String, JsonValue>),
}
impl From<&str> for JsonValue {
    fn from(value: &str) -> Self {
        Self::String(value.to_owned())
    }
}
impl From<String> for JsonValue {
    fn from(value: String) -> Self {
        Self::String(value)
    }
}
impl From<bool> for JsonValue {
    fn from(value: bool) -> Self {
        Self::Bool(value)
    }
}
impl From<i64> for JsonValue {
    fn from(value: i64) -> Self {
        Self::Number(value.to_string())
    }
}
impl JsonValue {
    pub fn object(entries: impl IntoIterator<Item = (impl Into<String>, JsonValue)>) -> Self {
        Self::Object(
            entries
                .into_iter()
                .map(|(key, value)| (key.into(), value))
                .collect(),
        )
    }
    pub fn array(values: impl IntoIterator<Item = JsonValue>) -> Self {
        Self::Array(values.into_iter().collect())
    }
    pub fn to_json(&self) -> String {
        self.try_to_json().expect("invalid JSON value")
    }

    pub fn try_to_json(&self) -> Result<String, String> {
        validate_numbers(self)?;
        Ok(to_serde(self).to_string())
    }
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct ResponseMeta {
    pub request_id: Option<String>,
    pub retry_after_seconds: Option<u64>,
    pub rate_limit_remaining: Option<i64>,
    pub replayed: bool,
}
#[derive(Clone, Debug)]
pub struct RawResponse {
    pub status: u16,
    pub headers: BTreeMap<String, String>,
    pub body: String,
}
pub trait Transport: Send + Sync {
    fn send(
        &self,
        method: &str,
        url: &str,
        headers: &BTreeMap<String, String>,
        body: Option<&str>,
    ) -> Result<RawResponse, String>;
}

#[derive(Debug)]
pub struct CloverError {
    pub status: u16,
    pub problem: Option<JsonValue>,
    pub metadata: ResponseMeta,
    pub message: String,
}
impl fmt::Display for CloverError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(&self.message)
    }
}
impl std::error::Error for CloverError {}

pub struct CloverClient {
    base_url: String,
    api_key: String,
    user_agent: String,
    max_retries: u8,
    retry_base_delay: Duration,
    transport: Box<dyn Transport>,
    sleeper: Box<dyn Fn(Duration) + Send + Sync>,
}
pub const DEFAULT_MAX_RESPONSE_BODY_BYTES: usize = 4 * 1024 * 1024;
#[allow(clippy::result_large_err)]
impl CloverClient {
    pub fn new(base_url: impl Into<String>, api_key: impl Into<String>) -> Self {
        let base_url = base_url.into();
        Self::with_transport(
            base_url,
            api_key,
            Box::new(UreqTransport::default()),
            Box::new(std::thread::sleep),
        )
    }
    pub fn with_transport(
        base_url: impl Into<String>,
        api_key: impl Into<String>,
        transport: Box<dyn Transport>,
        sleeper: Box<dyn Fn(Duration) + Send + Sync>,
    ) -> Self {
        let base_url = base_url.into().trim_end_matches('/').to_owned();
        let api_key = api_key.into();
        let valid_scheme = base_url.strip_prefix("http://").is_some()
            || base_url.strip_prefix("https://").is_some();
        let authority = base_url
            .split_once("://")
            .and_then(|(_, rest)| rest.split('/').next())
            .unwrap_or("");
        if !valid_scheme
            || authority.is_empty()
            || authority.contains('@')
            || base_url.contains('?')
            || base_url.contains('#')
        {
            panic!("base_url must be an absolute http(s) URL without userinfo/query/fragment")
        }
        if api_key.trim().is_empty() {
            panic!("api_key is required")
        };
        Self {
            base_url,
            api_key,
            user_agent: "clover-sdk-rust/0.1.0".into(),
            max_retries: 2,
            retry_base_delay: Duration::from_millis(100),
            transport,
            sleeper,
        }
    }
    pub fn send(
        &self,
        request: JsonValue,
        idempotency_key: &str,
    ) -> Result<JsonValue, CloverError> {
        self.request("POST", "/api/v1/emails", Some(request), Some(idempotency_key))
    }
    pub fn send_batch(
        &self,
        items: Vec<JsonValue>,
        idempotency_key: &str,
    ) -> Result<JsonValue, CloverError> {
        let sanitized: Vec<JsonValue> = items
            .into_iter()
            .map(|item| match item {
                JsonValue::Object(mut object) => {
                    object.remove("scheduled_at");
                    JsonValue::Object(object)
                }
                other => other,
            })
            .collect();
        self.request(
            "POST",
            "/api/api/v1/emails/batch",
            Some(JsonValue::object([(
                String::from("items"),
                JsonValue::array(sanitized),
            )])),
            Some(idempotency_key),
        )
    }
    pub fn schedule(
        &self,
        id: &str,
        scheduled_at: &str,
        idempotency_key: &str,
    ) -> Result<JsonValue, CloverError> {
        self.request(
            "POST",
            &format!("/api/api/v1/emails/{}/schedule", escape(id)),
            Some(JsonValue::object([(
                String::from("scheduled_at"),
                scheduled_at.into(),
            )])),
            Some(idempotency_key),
        )
    }
    pub fn cancel(&self, id: &str, idempotency_key: &str) -> Result<JsonValue, CloverError> {
        self.request(
            "POST",
            &format!("/api/api/v1/emails/{}/cancel", escape(id)),
            None,
            Some(idempotency_key),
        )
    }
    pub fn get(&self, id: &str) -> Result<JsonValue, CloverError> {
        self.request("GET", &format!("/api/api/v1/emails/{}", escape(id)), None, None)
    }
    pub fn list(&self, query: &BTreeMap<String, String>) -> Result<JsonValue, CloverError> {
        let encoded = query
            .iter()
            .map(|(key, value)| format!("{}={}", escape(key), escape(value)))
            .collect::<Vec<_>>()
            .join("&");
        self.request(
            "GET",
            &format!(
                "/api/v1/emails{}",
                if encoded.is_empty() {
                    String::new()
                } else {
                    format!("?{encoded}")
                }
            ),
            None,
            None,
        )
    }
    fn request(
        &self,
        method: &str,
        path: &str,
        payload: Option<JsonValue>,
        idempotency_key: Option<&str>,
    ) -> Result<JsonValue, CloverError> {
        if method != "GET" && !valid_idempotency_key(idempotency_key.unwrap_or("")) {
            return Err(CloverError {
                status: 0,
                problem: None,
                metadata: ResponseMeta::default(),
                message: "idempotency key must match ^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$".into(),
            });
        }
        let body = match payload.as_ref().map(JsonValue::try_to_json).transpose() {
            Ok(body) => body,
            Err(error) => {
                return Err(CloverError {
                    status: 0,
                    problem: None,
                    metadata: ResponseMeta::default(),
                    message: error,
                })
            }
        };
        let mut headers = BTreeMap::new();
        headers.insert(
            "accept".into(),
            "application/json, application/problem+json".into(),
        );
        headers.insert("authorization".into(), format!("Bearer {}", self.api_key));
        headers.insert("user-agent".into(), self.user_agent.clone());
        if body.is_some() {
            headers.insert("content-type".into(), "application/json".into());
        }
        if let Some(key) = idempotency_key {
            headers.insert("idempotency-key".into(), key.into());
        }
        for attempt in 0..=self.max_retries {
            let response = match self.transport.send(
                method,
                &format!("{}{}", self.base_url, path),
                &headers,
                body.as_deref(),
            ) {
                Ok(value) => value,
                Err(error) => {
                    return Err(CloverError {
                        status: 0,
                        problem: None,
                        metadata: ResponseMeta::default(),
                        message: error,
                    })
                }
            };
            let metadata = response_meta(&response.headers);
            if response.body.len() > DEFAULT_MAX_RESPONSE_BODY_BYTES {
                return Err(CloverError {
                    status: response.status,
                    problem: None,
                    metadata,
                    message: "Clover response body exceeds the configured limit".into(),
                });
            }
            let parsed = match parse_json(&response.body) {
                Some(value) => value,
                None => {
                    return Err(CloverError {
                        status: response.status,
                        problem: None,
                        metadata,
                        message: "Clover returned malformed JSON".into(),
                    });
                }
            };
            if (200..300).contains(&response.status) {
                return Ok(unwrap_envelope(parsed));
            }
            if (method == "GET" || idempotency_key.is_some())
                && retryable(response.status)
                && attempt < self.max_retries
            {
                (self.sleeper)(
                    metadata
                        .retry_after_seconds
                        .map(Duration::from_secs)
                        .unwrap_or(self.retry_base_delay * 2u32.pow(attempt.into())),
                );
                continue;
            }
            let message = match &parsed {
                JsonValue::Object(map) => map
                    .get("error")
                    .and_then(|value| match value {
                        JsonValue::Object(error) => error.get("message"),
                        _ => None,
                    })
                    .and_then(|value| match value {
                        JsonValue::String(text) => Some(text.clone()),
                        _ => None,
                    })
                    .or_else(|| {
                        map.get("title").and_then(|value| {
                            if let JsonValue::String(text) = value {
                                Some(text.clone())
                            } else {
                                None
                            }
                        })
                    })
                    .unwrap_or_else(|| format!("Clover request failed ({})", response.status)),
                _ => format!("Clover request failed ({})", response.status),
            };
            return Err(CloverError {
                status: response.status,
                problem: Some(parsed),
                metadata,
                message,
            });
        }
        unreachable!()
    }
}

fn unwrap_envelope(parsed: JsonValue) -> JsonValue {
    match parsed {
        JsonValue::Object(map) if matches!(map.get("success"), Some(JsonValue::Bool(_))) => {
            if map.get("success") == Some(&JsonValue::Bool(true)) {
                map.get("data").cloned().unwrap_or(JsonValue::Object(Default::default()))
            } else {
                JsonValue::Object(map)
            }
        }
        other => other,
    }
}

fn retryable(status: u16) -> bool {
    matches!(status, 408 | 425 | 429 | 500 | 502 | 503 | 504)
}
fn valid_idempotency_key(value: &str) -> bool {
    let bytes = value.as_bytes();
    if !(8..=128).contains(&bytes.len()) {
        return false;
    }
    let first = bytes[0];
    if !first.is_ascii_alphanumeric() {
        return false;
    }
    bytes[1..]
        .iter()
        .all(|byte| byte.is_ascii_alphanumeric() || b"._:-".contains(byte))
}
fn response_meta(headers: &BTreeMap<String, String>) -> ResponseMeta {
    let get = |key: &str| {
        headers
            .iter()
            .find(|(name, _)| name.eq_ignore_ascii_case(key))
            .map(|(_, value)| value.clone())
    };
    ResponseMeta {
        request_id: get("x-request-id"),
        retry_after_seconds: get("retry-after").and_then(|value| value.parse().ok()),
        rate_limit_remaining: get("x-ratelimit-remaining").and_then(|value| value.parse().ok()),
        replayed: get("idempotency-replayed")
            .map(|value| value.eq_ignore_ascii_case("true"))
            .unwrap_or(false),
    }
}
fn escape(value: &str) -> String {
    value
        .bytes()
        .map(|byte| {
            if byte.is_ascii_alphanumeric() || b"-_.~".contains(&byte) {
                (byte as char).to_string()
            } else {
                format!("%{byte:02X}")
            }
        })
        .collect()
}

/// Default HTTPS-capable transport backed by ureq and rustls.
pub struct UreqTransport {
    agent: ureq::Agent,
}

impl Default for UreqTransport {
    fn default() -> Self {
        Self {
            agent: ureq::AgentBuilder::new()
                .timeout(Duration::from_secs(30))
                .build(),
        }
    }
}

impl Transport for UreqTransport {
    fn send(
        &self,
        method: &str,
        url: &str,
        headers: &BTreeMap<String, String>,
        body: Option<&str>,
    ) -> Result<RawResponse, String> {
        let mut request = self.agent.request(method, url);
        for (key, value) in headers {
            request = request.set(key, value);
        }
        let response = match body {
            Some(body) => request.send_string(body),
            None => request.call(),
        };
        match response {
            Ok(response) => Self::response(response),
            Err(ureq::Error::Status(_, response)) => Self::response(response),
            Err(error) => Err(error.to_string()),
        }
    }
}

impl UreqTransport {
    fn response(response: ureq::Response) -> Result<RawResponse, String> {
        let headers = response
            .headers_names()
            .into_iter()
            .filter_map(|name| response.header(&name).map(|value| (name, value.to_owned())))
            .collect();
        let status = response.status();
        let mut reader = response
            .into_reader()
            .take((DEFAULT_MAX_RESPONSE_BODY_BYTES + 1) as u64);
        let mut body = String::new();
        reader
            .read_to_string(&mut body)
            .map_err(|error| error.to_string())?;
        if body.len() > DEFAULT_MAX_RESPONSE_BODY_BYTES {
            return Err("Clover response body exceeds 4 MiB".to_owned());
        }
        Ok(RawResponse {
            status,
            headers,
            body,
        })
    }
}

fn parse_json(input: &str) -> Option<JsonValue> {
    let value: serde_json::Value = serde_json::from_str(input).ok()?;
    from_serde(value)
}
fn from_serde(value: serde_json::Value) -> Option<JsonValue> {
    Some(match value {
        serde_json::Value::Null => JsonValue::Null,
        serde_json::Value::Bool(value) => JsonValue::Bool(value),
        serde_json::Value::Number(value) => JsonValue::Number(value.to_string()),
        serde_json::Value::String(value) => JsonValue::String(value),
        serde_json::Value::Array(values) => {
            JsonValue::Array(values.into_iter().filter_map(from_serde).collect())
        }
        serde_json::Value::Object(values) => JsonValue::Object(
            values
                .into_iter()
                .filter_map(|(key, value)| from_serde(value).map(|value| (key, value)))
                .collect(),
        ),
    })
}
fn to_serde(value: &JsonValue) -> serde_json::Value {
    match value {
        JsonValue::Null => serde_json::Value::Null,
        JsonValue::Bool(value) => serde_json::Value::Bool(*value),
        JsonValue::Number(value) => serde_json::from_str(value).expect("validated JSON number"),
        JsonValue::String(value) => serde_json::Value::String(value.clone()),
        JsonValue::Array(values) => serde_json::Value::Array(values.iter().map(to_serde).collect()),
        JsonValue::Object(values) => serde_json::Value::Object(
            values
                .iter()
                .map(|(key, value)| (key.clone(), to_serde(value)))
                .collect(),
        ),
    }
}
fn validate_numbers(value: &JsonValue) -> Result<(), String> {
    match value {
        JsonValue::Number(number) => {
            if serde_json::from_str::<serde_json::Value>(number).is_err() {
                return Err("invalid JSON number".into());
            }
        }
        JsonValue::Array(values) => {
            for value in values {
                validate_numbers(value)?;
            }
        }
        JsonValue::Object(values) => {
            for value in values.values() {
                validate_numbers(value)?;
            }
        }
        JsonValue::Null | JsonValue::Bool(_) | JsonValue::String(_) => {}
    }
    Ok(())
}
