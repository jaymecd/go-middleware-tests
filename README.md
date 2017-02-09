## Middleware tests in GO

Inspired by Mat Ryer

https://medium.com/@matryer/writing-middleware-in-golang-and-how-go-makes-it-so-much-fun-4375c1246e81

### Install

```
$ go get -d ./...
$ go build -o test 
```

### Usage

Start server in *graceful* mode:

```
$ ./test
```

Detect request ID from request:

```
$ curl -i localhost:8080 -H 'X-Request-ID: 42'

HTTP/1.1 200 OK
X-Pre: Logging
X-Pre: Tracing
Date: Thu, 09 Feb 2017 17:06:51 GMT
Content-Length: 18
Content-Type: text/plain; charset=utf-8

My Request-Id: 42
```

Generate new request ID, if it's not provided:

```
$ curl -i localhost:8080

HTTP/1.1 200 OK
X-Pre: Logging
X-Pre: Tracing
Date: Thu, 09 Feb 2017 17:07:27 GMT
Content-Length: 42
Content-Type: text/plain; charset=utf-8

My Request-Id: 01B8HXG516XA434N5J5682WBZJ
```

2. Start server in *strict* mode:

```
$ ./test -strict
```

Detect request ID from request:

```
$ curl -i localhost:8080 -H 'X-Request-ID: 42'

HTTP/1.1 200 OK
X-Pre: Logging
X-Pre: Tracing
Date: Thu, 09 Feb 2017 17:06:51 GMT
Content-Length: 18
Content-Type: text/plain; charset=utf-8

My Request-Id: 42
```

Throw error if no request id detected:

```
$ curl -i localhost:8080

HTTP/1.1 400 Bad Request
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
X-Pre: Logging
X-Pre: Tracing
Date: Thu, 09 Feb 2017 17:10:30 GMT
Content-Length: 37

Header key is not provided or empty.
```
