# Task - package for repetitive tasks

Director observes workers and tell them what and when they should do.
Workers do work.

### Available parameters

#### Director params

1. `WithIdGenerator(func() string)` - rule, how to give a name to a worker.

#### Worker params
1. `Id(string)` sets name for worker.
2. `Repeat(int)` execute worker `n` times. 0 by default. For negative values of `n` executes infinitely.
3. `Infinitely()` execute worker infinitely.
4. `WithTimeout(time.Duration)` set timeout for loop executions.
5. `WithTimeoutFunc(func() time.Duration)` set timeout rule for loop executions.
6. `WithArgs(...interface{})` sets args for first worker execution.
