package main

import (
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "seed":
			runSeedCommand()
			return
		case "reconcile":
			runReconcileCommand()
			return
		case "worker":
			runWorkerCommand()
			return
		}
	}
	runServeCommand()
}
