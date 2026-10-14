# Third-party notices

AI Vault itself is licensed under the [MIT License](LICENSE). Third-party code retains its original copyright and license terms.

The files linked below are unmodified copies from the exact module versions selected by `go list -m -json all` for this repository. The list includes the complete selected dependency graph, including development/test dependencies that may not be linked into a particular release binary. [Machine-readable module versions](third_party_licenses/modules.json) are included for review.

| Go module | Version | Original notice files |
| --- | --- | --- |
| `cloud.google.com/go/compute/metadata` | `v0.3.0` | [LICENSE](third_party_licenses/cloud.google.com_go_compute_metadata@v0.3.0/LICENSE) |
| `github.com/golang-jwt/jwt/v5` | `v5.3.1` | [LICENSE](third_party_licenses/github.com_golang-jwt_jwt_v5@v5.3.1/LICENSE) |
| `github.com/google/go-cmp` | `v0.7.0` | [LICENSE](third_party_licenses/github.com_google_go-cmp@v0.7.0/LICENSE) |
| `github.com/google/jsonschema-go` | `v0.4.3` | [LICENSE](third_party_licenses/github.com_google_jsonschema-go@v0.4.3/LICENSE) |
| `github.com/modelcontextprotocol/go-sdk` | `v1.7.0` | [LICENSE](third_party_licenses/github.com_modelcontextprotocol_go-sdk@v1.7.0/LICENSE) |
| `github.com/segmentio/asm` | `v1.1.3` | [LICENSE](third_party_licenses/github.com_segmentio_asm@v1.1.3/LICENSE) |
| `github.com/segmentio/encoding` | `v0.5.4` | [LICENSE](third_party_licenses/github.com_segmentio_encoding@v0.5.4/LICENSE) |
| `github.com/yosida95/uritemplate/v3` | `v3.0.2` | [LICENSE](third_party_licenses/github.com_yosida95_uritemplate_v3@v3.0.2/LICENSE) |
| `golang.org/x/mod` | `v0.41.0` | [LICENSE](third_party_licenses/golang.org_x_mod@v0.41.0/LICENSE), [PATENTS](third_party_licenses/golang.org_x_mod@v0.41.0/PATENTS) |
| `golang.org/x/oauth2` | `v0.35.0` | [LICENSE](third_party_licenses/golang.org_x_oauth2@v0.35.0/LICENSE) |
| `golang.org/x/sync` | `v0.23.0` | [LICENSE](third_party_licenses/golang.org_x_sync@v0.23.0/LICENSE), [PATENTS](third_party_licenses/golang.org_x_sync@v0.23.0/PATENTS) |
| `golang.org/x/sys` | `v0.44.0` | [LICENSE](third_party_licenses/golang.org_x_sys@v0.44.0/LICENSE), [PATENTS](third_party_licenses/golang.org_x_sys@v0.44.0/PATENTS) |
| `golang.org/x/text` | `v0.42.0` | [LICENSE](third_party_licenses/golang.org_x_text@v0.42.0/LICENSE), [PATENTS](third_party_licenses/golang.org_x_text@v0.42.0/PATENTS) |
| `golang.org/x/time` | `v0.15.0` | [LICENSE](third_party_licenses/golang.org_x_time@v0.15.0/LICENSE), [PATENTS](third_party_licenses/golang.org_x_time@v0.15.0/PATENTS) |
| `golang.org/x/tools` | `v0.49.0` | [LICENSE](third_party_licenses/golang.org_x_tools@v0.49.0/LICENSE), [PATENTS](third_party_licenses/golang.org_x_tools@v0.49.0/PATENTS) |

## Go runtime and standard library

Release executables include Go runtime and standard library code. The initial v0.1.0 release is built with Go 1.27.2. Its original [LICENSE](third_party_licenses/go@go1.27.2/LICENSE) and [PATENTS](third_party_licenses/go@go1.27.2/PATENTS) are included. CI tracks the latest Go 1.27 patch release; refresh these notices and the [toolchain provenance](third_party_licenses/toolchain.json) when the build toolchain changes.

## Distribution

Keep this file, AI Vault's `LICENSE`, and the `third_party_licenses` directory with redistributed release archives. The upstream notices state their own applicable terms; this index does not replace them or relicense third-party code. When dependency or toolchain versions change, refresh the copied original notices together with this index.
