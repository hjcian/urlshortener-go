# Flow
- upload
  - gen id
  - save id(index, PK?)/url/createAt/expireAt/deleteAt(NULL)
  - return id/shorten URL
- delete
  - update deleteAt field of the id
- redirect
  - select url from db where id=id AND expireAt > now AND deleteAt != NULL
- cache
  - (id: url) (expire at ?)
    - add by upload, expire at expireAt
    - add by
  - (id: NULL) (expire at short period)
    - consider high concurrent non-existent id queries(?)


# Error handling
- /api/v1/urls (upload with POST)
  - OK: 200 (JSON response)
  - Bad Request: 400 (lake one of field)

- /api/v1/urls/<url_id> (delete with DELETE)
  - OK: 204
  - Bad Request: 400

- /<url_id> (redirect URL)
  - OK: 30X

# Considerations
- many clients access same shorten URL simultaneously
- try to access non-existent shorten URL


# References
- https://www.loggly.com/blog/http-status-code-diagram/
    - success delete: 204


# Issues
- cache penetration (too many requests, legitimate or malicious, concurrent access the redirect endpoint)
  - key filter (simple rule, additional proof-of-work)
  - cache empty data (maybe set a short expiration time)
  - `bloom filter` to filter out the data that must not be existed
- cache stampede
  - lock (Redis's SetNX + pub/sub)