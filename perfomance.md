# arch_lab4
# perfomance with Redis (1) and without (2)
# (1)
#
go-wrk -c 1 http://localhost:8080/user/findById/1000
Running 10s test @ http://localhost:8080/user/findById/1000
  1 goroutine(s) running concurrently
61139 requests in 9.846772367s, 11.95MB read
Requests/sec:           6209.04
Transfer/sec:           1.21MB
Avg Req Time:           161.055µs
Fastest Request:        112.972µs
Slowest Request:        21.071325ms
Number of Errors:       0

# (2)
#
go-wrk -c 1 http://localhost:8080/user/findById/1000
Running 10s test @ http://localhost:8080/user/findById/1000
  1 goroutine(s) running concurrently
20976 requests in 9.936510781s, 3.48MB read
Requests/sec:           2111.00
Transfer/sec:           358.71KB
Avg Req Time:           473.708µs
Fastest Request:        327.953µs
Slowest Request:        23.712861ms
Number of Errors:       0
