# Index
- [Simple Insurance Assessment Task](#simple-insurance-assessment-task)
- [Problem understanding](#problem-understanding)
  - [First approach](#first-approach)
  - [Later approaches](#later-approaches)
- [Usage](#usage)

# Task
Using only the standard library, create a Go HTTP server that on each request responds with a counter of the total number of requests that it has received during the previous 60 seconds (moving window). The server should continue to the return the correct numbers after restarting it, by persisting data to a file.

Most importantly... enjoy it :)

# Problem understanding
We can recognize two main problems:
- data structures: what type of data we have to store to solve this problem? Do need a timestamp for each request? or we can manage with a counter?
- persistence: how to store the state of the server? only a single file? does it grow? etc..

We can leave the persistence problem to later. As soon as we know how to implement the solution, we can think how to store the involved structures.

## First approach
I usually start with direct and simple approach to solve the problem, and look at the weakness of the solution.

As we can see for the very first commit the RateCounter is a datastructure that implements the requested behaviour.
Such a structure stores an time.Time for each request to `Increase` in a slice. When the Counter is requested `Rate` cycles
through the slice and returns the time.Time in the corrisponding window.  Since `time.Time` are appended each request 
the slice is sorted by ascending order and this simplifies the algorithm.

```go 
func (c *RateCounter) RequestCounter() int {
        c.entriesMutex.Lock()
        defer c.entriesMutex.Unlock()

        windowsStart := time.Now().Add(-c.windows)

        i := 0
        for i = range c.entries {
                if c.entries[i].After(windowsStart) {
                        break
                }
        }

        c.entries = c.entries[i:]
        return len(c.entries)
}
```

This algorithm have a lot of weaknesses:
- Time complexity is O(n) in the worst case
- Allocates time.Time per request. Even if the slice is truncated we allocate continuously in the heap, this allocation is not under our control, indeed depends on the clients requests.
- The slice management start from an external call

## Later approaches
We can move the management of the slice internally to the struct. This implies using a go routine that runs each time period. Such a routine can run the algorithm above and store a counter that is going to be returned when user calls `RequestCounter`.

We still have a slice of time.Time that grows indefinitely. To solve this problem imagine we divide the windows in various ticks:
```
  ________________________________
 |                                |
 |                                |
 |                                |
 |                                |
 |  2  5  1  0  0  1  2  3  5  9  |
 |__|__|__|__|__|__|__|__|__|__|__|
 t0                               tn 

 counter = 28
```
For each tick we store the number of requests received from the previous tick.
When we move the window right we subtract to the main `counter` the number of requests of the oldest tick (the leftmost tick).

This algorithm resolve the above weaknesses:
- Time complexity is O(1), at each tick we simply store a number. Also the counter is simply returned.
- Initially allocate a slice of len = number of ticks. No other allocation are needed.
- Counter is managed internally from a go routine 

# Usage
Build:
- `make`

Run:
- Server: `./_out/server [-help] [-persistence <file-path>] [-port <8080>]`
- Client: `./_out/client [-help] [-address <server-address>] [-frequency <nReq-per-second>]`

Test:
- `make test`
