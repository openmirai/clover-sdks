# Releasing

Run `mvn -B -ntp verify` from a clean checkout and update `CHANGELOG.md`. Use
the Maven Release Plugin to prepare the next version, create a signed
`java/vX.Y.Z` tag, and push it. The tag workflow repeats verification, signs source,
Javadoc, and POM artifacts with the configured GPG key, publishes through
Sonatype's Central Publisher Portal (`central-publishing-maven-plugin`), and
creates a GitHub release. Configure the `central` Maven server with a
Sonatype Portal user-token pair, register/verify the namespace, and provide
`CENTRAL_USERNAME`, `CENTRAL_PASSWORD`, `MAVEN_GPG_PRIVATE_KEY`, and
`MAVEN_GPG_PASSPHRASE` repository secrets. The old OSSRH staging endpoint is
not used.
