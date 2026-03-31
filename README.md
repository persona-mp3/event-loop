# NodeJS Eventloop Simulation


Node's event loop seems to be a mystery to most people, especially 
the ```async/await``` concpets, but at the lower-level, it's really...
not complex, would even almost say simple. Alongside trying to read the 
source code, you should look at these websites, they help with understanding 
and explaining the EventLoop in simple and quick terms

## Sources

1. [Builder Blog](https://www.builder.io/blog/visual-guide-to-nodejs-event-loop). They have illustrative
diagrams. Follow the blogs all through as the get more fine-grained to 
a particular API.

2. [EventLoop visualiser](https://event-loop-visualizer-ruby.vercel.app/). They have really good visuals and simulations
there, with actual javascript code. Although, watch out, there are some things I noticed there, with the Promise Queue
vs the NextTicker Queue (under MicroTask Queue). You should also run the code in your Node envrionment, just to be sure
I already have some of them in [challenges.js](./pocs/event-loop-challenges.js) file. 


### Progress
Now the aim of this project is just to simply simulate the order of execution of code by the v8 Engine 
and how it assigns different functions to different API's withing the Node Runtime. 

1. Highest Order Priority Queues: `Microtask` queue which consists of `Nextticker` and 'Promise' queues
3. IO Queue

The left rest to implement are:
2. Timer Queue 
4. Check Queue
5. Close Queue


For a summary guide, [doc.md](./doc.md)

# Run the program
Make sure you have the Go Compiler installed, visit [Go](https://go.dev/dl)
```bash
git clone https://github.com/persona-mp3/event-loop.git ~/event-loop
cd ~/event-loop
go run main.go
```

## To schedule a program
You can use the `Task` type
```go
package main

import (
	"persona/runtime"
	"sync"
    "fmt"
)

type Task struct {
	// Name of the function or any uuid
	Id      string
    // The actual function
	Execute func() (any, error)
	// Kind of function, right now only Sync , NextTicker and Promises function
    // Metas have been integrated
	Meta
}


func myFunction() (any, error){
    // does something
    return result, nil
}


func main() {
    newTask := &runtime.Task{
        Id: "my-function-name", // not compulsory
        Execute: myFunction,
        Meta: PromiseMeta,
    }


    // we create a new runtime
    rt := runtime.NewRuntime()
    // we provide a src-channel, to simulate how a node 
    // process consumes your parses you file

    src := make(chan *runtime.Task)
    // so we can wait for our runtime to finish executing tasks
    done := make(chan any) 

    // start feeding the runtime
    // when the channel closes, the runtime knows it 
    // can fully commit to executing on other queues
	go func() {
		defer close(src)
        src <- newTask
	}()


	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		rt.Start(src, done)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		res, open := <-done // could change it to return exit code
		if !open {
			fmt.Println("the runtime closed the done channel!")
			return
		}

		fmt.Println("Done executing all code", res)

	}()

	wg.Wait()
    
}
```


### Contribution
Feel free to contribute, project is open to Pull Requests
Make sure there are no race conditions and unreleased locks, which are more 
harder. You can use go's built in race detector
```bash
go run --race main.go
```

It will interrupt the program if there are race conditions
