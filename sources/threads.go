package syncdatasources

import (
	"runtime"
)

// GetThreadsNum returns the number of available CPUs
// If environment variable SDS_ST is set it retuns 1
// It can be used to debug single threaded verion
func GetThreadsNum(ctx *Ctx) int {
	// Use environment variable to have singlethreaded version
	if ctx.NCPUs > 0 {
		n := int(float64(runtime.NumCPU()) * ctx.NCPUsScale)
		if ctx.NCPUs > n {
			ctx.NCPUs = n
		}
		runtime.GOMAXPROCS(ctx.NCPUs)
		return ctx.NCPUs
	}
	if ctx.ST {
		return 1
	}
	thrN := int(float64(runtime.NumCPU()) * ctx.NCPUsScale)
	runtime.GOMAXPROCS(thrN)
	return thrN
}
