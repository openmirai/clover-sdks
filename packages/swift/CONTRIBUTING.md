# Contributing

Use Swift 5.9 or newer. Keep public API `Sendable`, preserve the injected
transport seam, and add an offline XCTest whenever request or response behavior
changes. Do not use live Clover credentials in tests.

Run `swift test`, `swift-format lint --strict`, and
`swiftlint lint --strict --no-cache`. Pull requests should include a changelog
entry for user-visible changes.
