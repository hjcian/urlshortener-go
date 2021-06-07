# URL Shortener - A Go implementation

## Tests
- `make unittest`
- `make e2e`
- `make test-all`

### See coverage after tests
- `make see-coverage`

## TODO

*Considerations*
- many clients access same shorten URL simultaneously
- try to access non-existent shorten URL

*guideline*
- cache penetration (scenario: too many requests from different sources, maybe legitimate or malicious, concurrently access the redirect endpoint)
  - basic key filter by simple rule, or additional proof-of-work
  - cache empty data (maybe set a short expiration time)
  - `bloom filter` to filter out the data that must not be existed
- cache stampede (scenario: hot key)
  - lock (Redis's SetNX + pub/sub)

# References
- https://www.loggly.com/blog/http-status-code-diagram/
    - success delete: 204
