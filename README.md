# URL Shortener - A Go implementation

## Tests
- `make unittest`
- `make e2e`
- `make test-all`

### See coverage after tests
- `make see-coverage`

## TODO

### Cache strategy
*Considerations*
- many clients access same shorten URL simultaneously
- try to access non-existent shorten URL

*guideline*
- cache penetration (scenario: too many requests from different sources, maybe legitimate or malicious, concurrently access the redirect endpoint)
  - basic key filter by simple rule, or additional proof-of-work
  - normal use cases: cache empty data
  - malicious attacks: should consider `bloom filter` to filter out the data that must not be existed to avoid cache too much
- cache stampede (scenario: hot key)
  - lock (Redis's SetNX + pub/sub)

### Recycling strategy
- lazy triggering recycling process while in-memory queue is empty
- TODO:
  - create a background schedule to trigger recycling process

*thinking*
- May current implementation of in-memory queue causes memory leakage when en/dequeuing several times?

# References
- https://www.loggly.com/blog/http-status-code-diagram/
    - success delete: 204
