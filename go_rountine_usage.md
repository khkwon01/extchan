## 1. 기본 사용법
'''
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
'''
## 2. 서브 고루틴 종료될 때까지 대기
var wg sync.WaitGroup

wg.Add(3)  // main
wg.Done()  // go rountine
wg.Wait()  // main

## 3. Mutex를 사용한 동시성 문제 해결
var mutex sync.Mutex

mutex.Lock()         // go rountine
defer mutex.Unlock() // go rountine

## 4. 고루틴끼리 통신 (채널)


