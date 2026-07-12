# Proto Sources

`recruitment.proto` is the canonical protobuf source for the current Smart Recruit gateway and backend services.

The protobuf package and `go_package` stay compatible with the existing generated code during this migration. Contract changes must update this canonical file, regenerate Go code, and keep the legacy mirrors synchronized until the monolith fallback is retired.
