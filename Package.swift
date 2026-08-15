// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "CloverSDK",
    platforms: [
        .iOS(.v15),
        .macOS(.v12),
        .tvOS(.v15),
        .watchOS(.v8)
    ],
    products: [
        .library(name: "CloverSDK", targets: ["CloverSDK"])
    ],
    targets: [
        .target(
            name: "CloverSDK",
            path: "packages/swift/Sources/CloverSDK",
            swiftSettings: [.enableUpcomingFeature("StrictConcurrency")]
        ),
        .testTarget(
            name: "CloverSDKTests",
            dependencies: ["CloverSDK"],
            path: "packages/swift/Tests/CloverSDKTests"
        )
    ]
)
