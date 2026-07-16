@simplechat host simplechat.minhhoccode111.com
handle @simplechat {
    reverse_proxy 127.0.0.1:8082
}
