# Contributing

Use the Dart SDK version declared in `pubspec.yaml`. Keep public models typed,
preserve the injected `http.Client` seam, and add offline `package:test`
coverage for every protocol or retry change. Do not use live Clover endpoints
or credentials in tests.

Run `dart format`, `dart analyze`, and `dart test`; run
`dart pub publish --dry-run` before a release. Add user-visible changes to the
changelog.
