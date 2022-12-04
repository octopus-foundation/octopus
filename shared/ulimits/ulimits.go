//go:build !js

package ulimits

import (
	"fmt"
	"github.com/logrusorgru/aurora"
	"syscall"
)

func SetupForHighLoad() {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		fmt.Printf("%s Failed to get rlimit: %s\n",
			aurora.Red("ERR"), err.Error())
	} else {
		var changed = false
		if rLimit.Cur < 256000 {
			rLimit.Cur = 256000
			changed = true
		}
		if rLimit.Max < 256000 {
			rLimit.Max = 256000
			changed = true
		}
		if !changed {
			fmt.Printf("%s current=%v, max=%v is enough, no changes needed\n",
				aurora.Green("rlimit"), rLimit.Cur, rLimit.Max)
			return
		}

		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			fmt.Printf("%s Failed to set rlimit: %s\n",
				aurora.Red("ERR"), err.Error())
		} else {
			err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
			if err != nil {
				fmt.Printf("%s Failed to check rlimit: %s\n",
					aurora.Red("ERR"), err.Error())
			} else {
				fmt.Printf("%s changed to current=%v, max=%v\n",
					aurora.Green("rlimit"), rLimit.Cur, rLimit.Max)
			}
		}
	}
}
