## 1. Implementation
- [ ] 1.1 Modify `internal/middleware/jwt.go`
    - [ ] Update `TokenLookup` configuration to include `header:Sec-WebSocket-Protocol` and `query:token`
    - [ ] (Optional) Implement a custom token extractor if `TokenLookup` string pattern is insufficient for extracting token from Protocol list. *For now, assume Token is the full value or verify standard behavior.*
- [ ] 1.2 Verify with a test case (or manual check) that WS handshake works with Token in Protocol.

## 2. Validation
- [ ] 2.1 Test logic with standard Authorization header (Regression test).
- [ ] 2.2 Test logic with `Sec-WebSocket-Protocol`.
- [ ] 2.3 Test logic with Query param `token`.
