module github.com/schollz/x-static

go 1.18

require (
	github.com/schollz/logger v1.2.0
	github.com/schollz/seamlessloop v0.1.1
)

require github.com/mpiannucci/peakdetect v0.0.0-20160920143128-9526111f1fb9 // indirect

replace github.com/schollz/seamlessloop => ../seamlessloop
