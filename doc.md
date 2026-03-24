Source: https://www.builder.io/blog/visual-guide-to-nodejs-event-loop

### There are 6 different queue is the node environment

1. Timer Queue ->    holds callback with `setTimeout ` and `setInterval ` functions
2. IO Queue ->       holds methods asscoiated with Network and IO Operations
3. Check Queue ->    Specific to NodeJs, for functions with `setImmeditate`
4. Close Queue ->    Close event of an async task

The above queues are all part of `libuv`

5. Microtask Queue contains two other queues:
    a. Promise Queue
    b. NextTick Queue


### Order of Execution
From the article, the flow of execution and prirotity of these 
queues is somewhat in this manner

1. V8 Executes normal code
2. When Stack is empty it checks the MicroTaskQueue
   specifically the `nextTick` and then `promise` queue
   and feeds the callback functions to the V8 Stack
3. Then the `Timer` queue is checked, fed to the v8 stack 
   and after, the `MicroQueues` are checked again

And So on

```
1. main_stack -> micro_queue
            executes [next_tick and promise_queue]
2. micro_queue -> timer_queue
3. timer_queue -> micro_queue
4. micro_queue -> io_queue
5. io_queue -> micro_queue
6. micro_queue -> check_queue
7. check_queue -> micro_queue
8. micro_queue -> close_queue
9. close_queue -> micro_queue
```
