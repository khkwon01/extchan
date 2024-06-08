## 1. 기본 사용법
```
package main
import (
  "fmt"
  "time"
  "sync"
)

func gotest1() {
  fmt.Printf("go1")
}

func gotest2() {
  fmt.Printf("go2")
}

func main() {
  go gotest1()
  go gotest2()

  time.Sleep(3 * time.Second)
}
```

## 2. 서브 고루틴 종료될 때까지 대기
```
var wg sync.WaitGroup

wg.Add(3)  // main
wg.Done()  // go rountine
wg.Wait()  // main
```

## 3. Mutex를 사용한 동시성 문제 해결
```
var mutex sync.Mutex

mutex.Lock()         // go rountine
defer mutex.Unlock() // go rountine
```

## 4. 고루틴끼리 통신 (채널)
```
var messages chan string = make(chan string)  // channel 생성

messages <- "Test1"     // 채널 데이터 넣기

var msg string = <- messages  // 채널 데이터 얻기

close(messages)

// 예제...
package main
import (
  "fmt"
  "sync"
  "time"
)

func main() {
  var wg sync.WaitGroup
  ch := make(chan int)

  wg.Add(1)
  go gotest(&wg, ch)
  ch <- 1
  wg.Wait()
}

func gotest(wg *sync.WaitGroup, ch chan int) {
  n := <-ch

  time.Sleep(time.Second)
  fmt.Printf("square: %d\n", n*n)
  wg.Done()
}
```

## 5. 고루틴 다채널 통신 (select)
```
package main

import (
  "fmt"
  "sync"
  "time"
)

func square(wg *sync.WaitGroup, ch chan int, quit chan bool) {
  tick := time.Tick(time.Second)

  for {
    select {
      case <- tick:
        fmt.Println("Tick")   // 1초 마다 실행
      case n := <-ch:
        fmt.Printf("square: %d\n", n*n)
        time.Sleep(time.Second)
      case <-quit:
        wg.Done()
        return
    }
  }
}

func main() {
  var wg sync.WaitGroup
  ch := make(chan int)
  quit := make(chan bool)

  wg.Add(1)
  go square(&wg, ch, quit)

  for i := 0; i < 10; i++ {
    ch <- i * 2
  }

  quit <- true
  wg.Wait()
}
```
