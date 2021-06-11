# URL Shortener - A Go implementation
- [URL Shortener - A Go implementation](#url-shortener---a-go-implementation)
  - [Platform Prerequisites](#platform-prerequisites)
  - [Local Tests](#local-tests)
  - [System Design Thinking](#system-design-thinking)
    - [Why use 6-letters as url id?](#why-use-6-letters-as-url-id)
    - [SQL or NoSQL?](#sql-or-nosql)
    - [About ID Generator](#about-id-generator)
    - [Caching Strategy](#caching-strategy)
  - [References](#references)

## Platform Prerequisites
- `make`
- `go` (1.15+)
- `docker`
- `timeout`
  - *via `brew install coreutils` if you run on MacOS*

## Local Tests
- `make unittest`
- `make e2e`
- `make alltest`
- `make see-coverage`
  - *see coverage report after tests*

## System Design Thinking
### Why use 6-letters as url id?
- 根據[我在我另一個 repo 中梳理的思路](https://github.com/hjcian/urlshortener-python#thoughts-about-scalability)，故此練習一樣選擇 6 碼作為短網址的 id
### SQL or NoSQL?
- 若預估儲存量達到 billions 的數量級 ([DB 選用基準](https://github.com/hjcian/urlshortener-python#3-db-%E9%81%B8%E7%94%A8%E5%9F%BA%E6%BA%96))，可能得直接選用 NoSQL 作為資料儲存較適合
- 但此練習先使用 postgres (SQL database) 作為資料儲存，並預先訂定 `interface` 供未來抽換
  - (TODO) 實作介接 MongoDB (or other NoSQL database) 的實作品

### About ID Generator
- recycle strategy
  - 此練習使用一個 in-memory 的 stack 來儲存回收的 id
    - 因為 FIFO 的 queue 會造成 memory leak (`s = s[1:]`，底下的 underlying array 並沒有被歸還)
    - 故採用 FILO 的 stack 來做，稍微減少一點 leak 的情況，但若 `slice` 的 capacity 一直成長，仍會持續佔用記憶體
    - (TODO) 改成使用 [`container/list`](https://golang.org/pkg/container/list/) 來實作，就能避免 memory leak。可再做個 benchmark 看看效能差多少
  - 當某次 request 發現 stack 已空時，則觸發回收機制
    - 但該次 request 還是即時產生 id
    - 而同時間僅允許一個 request 觸發此機制，避免高併發的情況對 DB 造成大量查詢
    - 下一個 request 進來時期待就可從 stack 中取得回收的 id
  - (TODO) 除了透過 request lazy triggering，也可再進一步設計一個 background job (background goroutine) 定期朝 DB 撈資料即可
- (TODO) 整個 id generator 可進一步考慮與此服務解耦，成為單獨的 ID generator service
  - 對 url shortener 來說，就只是向 ID generator service 取一個 ID，其餘的不管
  - ID generator service 就專心負責處理儲存資料至 DB 及從 DB 回收 ID 的任務

### Caching Strategy
- 此練習使用 in-memory cache library 實作
  - (TODO) 正式環境應再實作介接 Redis/Memcached 的實作品
- cache miss strategy
  - 面對 **existent shorten URL** 的高併發存取請求，cache miss 可能會引發 cache stampede 的問題 (hotkey)
    - 在 CAP 的妥協中，此練習選擇實作 AP，也就是在 concurrent requests 的情境下只允許一個 goroutine 可以去觸發 cache update 以避免 cache stampede，其餘的 requests 就先回應 `404`
    - 又當若此 URL 已過期，會再從 DB 中取得已過期的資訊並緩存。此步驟因為 AP 的考量也只會有一個 request 進到 DB，其餘的 requests 收到 `404` 也合理
    - (TODO) 但可能會因預設的 cache 時間過短，cache 過期時剛好遇到高併發請求，只有一個人可成功轉址，造成其餘 clients 需要重試、體驗不佳。故可在上傳時就將資料更新至 cache，並設定過期時間與 DB 內的一致
      - 此舉可降低使用者需要重試的機會；但會增加 cache 的負擔、儲存更多的資料
    - 或選擇**實作 CP**，其他 concurrent requests 都阻塞直到 cache updated，再從 cache 中取資料。但此舉是讓 client 等待，可能也是另一種不佳的體驗
    - (TODO) 又已過期或已刪除的資料，i.e. 不存在於 DB 的資料，可再進一步使用 **`bloom filter`** 放在 cache layer 之前，以降低 cache 儲存的負擔
  - 面對 **non-existent shorten URL** 的高併發存取請求，恐會有 cache penetration，此練習目前選擇先用 cache 存起來來避免
    - (TODO) 適合使用 **`bloom filter`** 放在 cache layer 之前，以降低 cache 儲存的負擔
- (TODO) 若使用 Redis 作為 cache server，可考慮利用 [`SETNX`](https://redis.io/commands/setnx)+[`Pub/Sub`](https://redis.io/topics/pubsub) 的功能來實作鎖的機制

## References
- cache related
  - [Caches, Promises and Locks](https://redislabs.com/blog/caches-promises-locks/)
  - [3 major problems and solutions in the cache world](https://medium.com/@mena.meseha/3-major-problems-and-solutions-in-the-cache-world-155ecae41d4f)
  - [有關 Cache 的一些筆記](https://kkc.github.io/2020/03/27/cache-note/)
  - [缓存更新的套路](https://coolshell.cn/articles/17416.html)
- [CAP定理101—分散式系統，有一好沒兩好](https://medium.com/%E5%BE%8C%E7%AB%AF%E6%96%B0%E6%89%8B%E6%9D%91/cap%E5%AE%9A%E7%90%86101-3fdd10e0b9a)