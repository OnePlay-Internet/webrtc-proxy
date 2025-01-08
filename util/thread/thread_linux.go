package thread

import "fmt"

func HighPriorityLoop(stop chan bool, fun func()) {
	wrapper := func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("panic happened in thread %v", err)
			}
		}()

		fun()
	}
	SafeThread(func() {
		for len(stop) == 0 {
			wrapper()
		}

		<-stop
	})
}
