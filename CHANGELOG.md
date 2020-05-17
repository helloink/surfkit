# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.9.1] - 2020-05-17
### Added
- [Pubsub] Add support for multiple outputs

## [1.9.0] - 2019-12-02
### Added
- [Server] Allow timeouts to be configured

## [1.8.0] - 2019-11-27
### Added
- [Utils] Authenticateable Requests now also reads Bearer Tokens from ENV

## [1.7.0] - 2019-11-14
### Added
- [Pubsub] Custom acknowledgement deadlines
- [Pubsub] Enable subscription removal

## [1.6.0] - 2019-11-11
- Pubsub: Support for multiple Pubsub Inputs (Subscriptions)
- Pubsub: Make Subscription Names explicit (and required)
- Pubsub: Default Ack Deadlines upped to 60 seconds

## [1.5.0] - 2019-11-04
### Added
- GCP Bearer Authentication support for HTTPs

## [1.4.0] - 2019-10-31
### Added
- Skip Pubsub subscription setup if HOST is invalid

## [1.3.0] - 2019-10-30
### Added
- [Pubsub] Full CloudEvents support
- [Pubsub] Support for Pull Subscriptions
- [Pubsub] Fine grained control over Subscription configurations
- [Pubsub] Added Push support
- [HTTP] Middleware Support and a built in Logger Middleware

### Fixed
- Correctly pass the event type

## [1.2.0] - 2019-08-22
### Added
- Support for Pubsub publishing, based on CloudEvents

### Changed
- Goodbye srvkit, hello surfkit!

## [1.0.1] - 2019-08-14
### Fixed
- Don't fail if no Subscription is given

## [1.0.0] - 2019-08-12
### Added
- Initial Release
