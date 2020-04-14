# Elasticsearch Query Proxy

My first foray into Go, this application runs a HTTP proxy intended for Elasticsearch and performs the following functions:

 - Detects when it is a `_search` or `_msearch` query.
 - Queries are de-duplicated as the front-end library has a problematic tendency to do this.
 - Parses the query according to a set of rules into "metrics" eg `LocationMetric`.
 - Sends these "valid" search queries to a `channel` mapped by the request IP, which then performs a debounce on the queries (per IP). Reasoning being the front-end library on page refresh can send in excess of 10+ queries. This is due to the reactive nature of the library.
 - Once debounced, we can assume the last query is the "final" intended query.
 - These final queries are sent to a custom `apex/log` handler derived from the packages original `es` handler.
 - This handler after reaching a desired buffer size sends all the queries to an Elasticsearch index using the `bulk` feature.
 
 ## Todos
 
 - Separate queries used for pulling single records as opposed to actual "search" queries
 - Add in security and normal features similar to other elasticsearch proxies
 - Make the query metrics parsing / extraction extensible (custom handlers etc)
 - Add in some form of caching for the actual ES responses
 
 ## Setup
 
 - Let's Encrypt needs port 443 to perform the tls-alpn-01 challenge, use this command if you do not want to run as root:
 
 `sudo setcap cap_net_bind_service=+ep ./elasticsearch-proxy`