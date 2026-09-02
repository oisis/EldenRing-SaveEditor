# ValidateReviewChanges

Validates an immutable copy of the exact current revision and authorizes Save
or Save As with an opaque, revision-bound token.

| | |
|---|---|
| EndpointID | `validate_review_changes` |
| Kind | Mutation of transient authorization state; no save revision commit |
| Status | implemented; Wails `ValidateReviewChanges` |
| Source | [validate_review_changes.go](../../../backend/endpoints/savesession/validate_review_changes.go) |

The result reports backend risk counts, completed validation stages and safe
issues. A critical operation or failed serialize/reload validation yields no
token and blocks writing. Any later session mutation invalidates an earlier
token, so validation can never authorize a different revision.
