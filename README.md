# URL Shortener - A Go implementation
- [URL Shortener - A Go implementation](#url-shortener---a-go-implementation)
  - [Platform Prerequisites](#platform-prerequisites)
  - [Run databases and app](#run-databases-and-app)
  - [Run Local Tests](#run-local-tests)
  - [System Design Thinking](#system-design-thinking)
    - [Why use 6-letters as URL id?](#why-use-6-letters-as-url-id)
    - [SQL or NoSQL?](#sql-or-nosql)
    - [About ID Generator](#about-id-generator)
    - [Caching Strategy](#caching-strategy)
      - [Cache Miss Strategy](#cache-miss-strategy)
  - [References](#references)

## Platform Prerequisites
- `make`
- `go` (1.15+)
- `docker`
- `timeout`
  - *via `brew install coreutils` if you run on MacOS*

## Run databases and app
- `make restart-all`
  - run postgres and redis containers
- `make run`
  - run url-shortener app with in-memory cache
- `make run-with-redis`
  - run url-shortener app with redis cache

## Run Local Tests
- `make unittest`
- `make e2e`
- `make alltest`
- `make see-coverage`
  - *see coverage report after tests*

## System Design Thinking
### Why use 6-letters as URL id?
- 根據我在[我的另一個 repo](https://github.com/hjcian/urlshortener-python#thoughts-about-scalability)中梳理過的思路，節錄重點：
  - 假設以 100 QPS 的寫入流量，且最長的網址上傳有效期限為五年，則總計需要約 **15 billions** 的短網址 id
  - 並且使用 10個數字+26個大小寫英文字母，共 62 個 letters 作為 id 的編碼字元，那麼僅需要 6 位數即可 ([ref: Token generation strategy](https://github.com/hjcian/urlshortener-python#token-generation-strategy))
  - 故此練習選擇 6 碼作為短網址的 id 並實作之

### SQL or NoSQL?
- 若預估儲存量達到 billions 的數量級 ([red: DB 選用基準](https://github.com/hjcian/urlshortener-python#3-db-%E9%81%B8%E7%94%A8%E5%9F%BA%E6%BA%96))，可能 NoSQL 較適合
- 但此練習先簡單地使用 postgres (SQL database) 作為資料儲存，並訂定 `Repository interface` 供抽換儲存方案時使用
  - 🚧 (TODO) 完成介接 MongoDB (or other NoSQL database) 的實作品

### About ID Generator
- ID 回收策略
  - 首先，此練習實作一個 in-memory 的 goroutine-safe stack 來儲存回收的 id
    - 因為 FIFO 的 queue 會造成 memory leak (i.e. 假如使用 `s = s[1:]`，underlying array 並沒有被歸還)，故採用 FILO 的 stack 來做，稍微減緩 leakage 的情況。但若 `slice` 的 capacity 一直成長，仍會持續佔用記憶體
    - 🚧 (TODO) 故考慮改成使用 [`container/list`](https://golang.org/pkg/container/list/) 來實作 stack(or queue) 來避免 memory leak
  - 觸發回收機制的時機為某次 request 發現 stack 為空時
    - 但該次 request 還是使用即時產生 id、不等待回收處理完成；回收處理流程則移至背景作業
    - 在回收處理流程結束前，僅允許一個 request 觸發；避免高併發的情況下，多個回收處理程序對 DB 造成大量 queries
    - 回收處理流程結束後就會填充 stack，後續的 requests 就可從 stack 中取得回收的 id
  - 🚧 (TODO) 除了透過被動地觸發，可再進一步做一個 background goroutine 定期向 DB 回收 id
- 🚧 (TODO) 整個 id generator 可進一步考慮與此服務解耦，成為單獨的 ID generator service
  - 對 url shortener 來說，就只是向 ID generator service 取一個 ID，其餘的不管
  - ID generator service 就專心負責處理儲存資料至 DB，及從 DB 回收 ID 的任務
  - 解耦之後也能專心處理此節點的效率瓶頸 ([ref: Online token generation 可能會是效率瓶頸，如何解決？](https://github.com/hjcian/urlshortener-python#2-online-token-generation-%E5%8F%AF%E8%83%BD%E6%9C%83%E6%98%AF%E6%95%88%E7%8E%87%E7%93%B6%E9%A0%B8%E5%A6%82%E4%BD%95%E8%A7%A3%E6%B1%BA))

### Caching Strategy
- 此練習在 [`cache/cacher/cacher.go`](./cache/cacher/cacher.go) 中定義 `Engine interface` 提供**快取引擎**需實作的接口，以支援在 [`cache/cache.go`](./cache/cache.go) 中的業務邏輯
  - 至於實際的**快取引擎**的實作品，此練習實作了以下方案：
    - ✔️ env 提供 `CACHE_MODE=inmemory` 來使用 in-memory cache 方案
    - ✔️ env 提供 `CACHE_MODE=redis` 來使用外部 Redis server 作為快取伺服器
      - 由於 application 本身因版本更迭、修 BUG 而重啟的機會很高，故使用外部 cache server 來儲存才能避免因 app 重啟造成的 cache avalanche
        - 📓 *cache avalanche (快取雪崩): 指 cache server 重啟時要成大量 requests 因 cache miss 打進 DB*
      - 🚧 (TODO) 尋找適合的 mocking 方法，於 unittest 中測試 redis 的實作品

#### Cache Miss Strategy
- 面對 **existent shorten URL** 的高併發存取請求，假設存取的是同一個 id，在 cache miss 時的 cache updating 可能會引起 cache stampede 的問題 (hotkey)
  - 故在 CAP 的妥協中，此練習選擇實作 AP，也就是在 concurrent requests 的情境下只允許一個 goroutine 可以去觸發 cache update 以避免 cache stampede，其餘的 requests 就先回應 `404`
  - 又考慮到可能會因預設的 cache 過期時間可能小於資料真實過期時間，結果 cache 過期後剛好遇到高併發請求，造成只有一個 client 可成功執行 cache update 及轉址、其餘 clients 需要重試、體驗不佳的情況，故此練習選擇在首次上傳時就將資料更新至 cache，並設定過期時間與真實過期時間一致
    - 🤔 (trade-off) 此舉讓 cache 與 storage 資料一致，理論上不會有 clients 需要重試的機會。***但會增加 cache 的負擔、儲存更多的資料***
  - 當 cached URL 過期時，仍需要再從 DB 中取得資訊並緩存
    - 此步驟因為 AP 的考量，也只會有一個 request 進到 DB 取得該筆已過期的資訊。其餘的 requests 即時收到 `404` 也與未來從快取中取得 `404` 結果一致
    - 🤔 (trade-off) 或選擇**實作 CP**，其他 concurrent requests 都阻塞直到 cache updated，再從 cache 中取資料。***但此舉是讓 client 等待，可能也是另一種不佳的體驗***
  - 🚧 (TODO) 可使用 **`bloom filter`** 放在 cache layer 之前，來確定***一定不在 storage 的資料***，以降低 cache 儲存的負擔、也減少進到 database 的機會

- 面對 **non-existent shorten URL** 的高併發存取請求，恐會有 cache penetration，此練習目前選擇先用 cache 存起來來避免
  - 🚧 (TODO) 適合使用 **`bloom filter`** 放在 cache layer 之前，以降低 cache 儲存的負擔


## References
- Cache related discussions
  - [Caches, Promises and Locks](https://redislabs.com/blog/caches-promises-locks/)
  - [3 major problems and solutions in the cache world](https://medium.com/@mena.meseha/3-major-problems-and-solutions-in-the-cache-world-155ecae41d4f)
  - [有關 Cache 的一些筆記](https://kkc.github.io/2020/03/27/cache-note/)
  - [缓存更新的套路](https://coolshell.cn/articles/17416.html)
- [CAP定理101—分散式系統，有一好沒兩好](https://medium.com/%E5%BE%8C%E7%AB%AF%E6%96%B0%E6%89%8B%E6%9D%91/cap%E5%AE%9A%E7%90%86101-3fdd10e0b9a)