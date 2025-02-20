package main

import (
	"fmt"
	"math"
	"slices"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

var debug = false

var (
	winWidth, winHeight int = 800, 600
)

type color struct {
	r, g, b byte
}

type point struct {
	x int
	y int
}

type edge struct {
	p1 point
	p2 point
	m  float64
}

func (e *edge) slope() float64 {
	var m float64
	if e.p1.x == e.p2.x {
		m = -math.MaxFloat64
	} else {
		m = (float64(e.p1.y) - float64(e.p2.y)) / (float64(e.p1.x) - float64(e.p2.x))
	}
	return m
}

func newEdge(p1, p2 *point) *edge {
	e := edge{
		p1: point{p1.x, p1.y},
		p2: point{p2.x, p2.y},
	}
	e.m = e.slope()
	return &e
}

func setPixel(x, y int, c color, pixels []byte) {
	index := (y*winWidth + x) * 4

	if index < len(pixels) && index >= 0 {
		pixels[index] = c.r
		pixels[index+1] = c.g
		pixels[index+2] = c.b
		pixels[index+3] = 255
	}
}

func drawLine(x1, y1, x2, y2 int, c color, pixels []byte) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)

	steps := math.Max(math.Abs(dx), math.Abs(dy))

	xinc := dx / steps
	yinc := dy / steps

	x := float64(x1)
	y := float64(y1)

	for i := 0; i < int(steps); i++ {
		setPixel(int(x), int(y), c, pixels)
		x += xinc
		y += yinc
	}
}

func drawCircle(xc, yc, r int, c color, pixels []byte) {
	x := 0
	y := r
	p := 3 - 2*r

	for x <= y {
		setPixel(xc+x, yc+y, c, pixels)
		setPixel(xc-x, yc+y, c, pixels)
		setPixel(xc+x, yc-y, c, pixels)
		setPixel(xc-x, yc-y, c, pixels)
		setPixel(xc+y, yc+x, c, pixels)
		setPixel(xc-y, yc+x, c, pixels)
		setPixel(xc+y, yc-x, c, pixels)
		setPixel(xc-y, yc-x, c, pixels)

		if p < 0 {
			p += 4*x + 6
		} else {
			p += 4*(x-y) + 10
			y--
		}
		x++
	}
}

func centroid(x []float64, y []float64) (float64, float64) {
	l := min(len(x), len(y))
	var xs float64 = 0
	var ys float64 = 0
	for i := 0; i < l; i++ {
		xs += x[i]
		ys += y[i]
	}
	return xs / float64(l), ys / float64(l)
}

func rotate(x, y, x_c, y_c, t_rad float64) (float64, float64) {
	/*
		x_t = x-x_c, y_t = y-y_c
		x' = x_t * cos(t) - y_t * sin(t)
		y' = x_t * sin(t) + y_t * cos(t)
		x = x_c + x', y = y_c + y'
	*/

	x_t := x - x_c
	y_t := y - y_c

	return x_c + x_t*math.Cos(t_rad) - y_t*math.Sin(t_rad),
		y_c + x_t*math.Sin(t_rad) + y_t*math.Cos(t_rad)
}

func edge_function(x, y int, e *edge) int {
	const eps = 1e-6
	E := float64(y-e.p1.y) - e.m*float64(x-e.p1.x)
	if math.Abs(E) < eps {
		return 0
	}
	if e.m > 0 {
		if E > 0 {
			return -1
		} else if E < 0 {
			return 1
		}
	} else if e.m < 0 {
		if E > 0 {
			return 1
		} else if E < 0 {
			return -1
		}
	}
	return 0
}

func test(i, j int, e *edge) {
	fmt.Printf("y: %v edge: %d", (j >= e.p1.y && j <= e.p2.y) || (j <= e.p1.y && j >= e.p2.y), edge_function(i, j, e))
	if ((j >= e.p1.y && j <= e.p2.y) || (j <= e.p1.y && j >= e.p2.y)) && edge_function(i, j, e) <= 0 {
		println("I'M LEFT")
	} else {
		println("DUNNO")
	}
}

func Fill(x []float64, y []float64, c color, pixels []byte) {
	L := min(len(x), len(y))

	// find bounding
	var xb1, yb1 int = math.MaxInt, math.MaxInt
	var xb2, yb2 int = 0, 0
	for i := 0; i < L; i++ {
		xb1 = min(xb1, int(math.Round(x[i])))
		yb1 = min(yb1, int(math.Round(y[i])))
		xb2 = max(xb2, int(math.Round(x[i])))
		yb2 = max(yb2, int(math.Round(y[i])))
	}

	if debug {
		println("=======================")
		println("Bound1", xb1, yb1)
		println("Bound2", xb2, yb2)
	}

	// construct edge
	xc, yc := centroid(x, y)
	p_tmp := []*edge{}
	for i := 0; i < L; i++ {
		p_tmp = append(p_tmp, newEdge(
			&point{int(math.Round(x[i])), int(math.Round(y[i]))},
			&point{int(math.Round(xc)), int(math.Round(yc))},
		))
	}
	slices.SortFunc(p_tmp, func(e1 *edge, e2 *edge) int {
		e1_atan := math.Atan2(float64(e1.p1.y)-float64(e1.p2.y), float64(e1.p1.x)-float64(e1.p2.x))
		e2_atan := math.Atan2(float64(e2.p1.y)-float64(e2.p2.y), float64(e2.p1.x)-float64(e2.p2.x))
		if e1_atan < e2_atan {
			return -1
		} else if e1_atan > e2_atan {
			return 1
		}
		return 0
	})

	if debug {
		println("=========================")
		for i := 0; i < L; i++ {
			println(p_tmp[i].p1.x, p_tmp[i].p1.y)
		}
	}

	ps := []*edge{}
	for i := 0; i < L; i++ {
		if i == L-1 {
			ps = append(ps, newEdge(
				&point{p_tmp[i].p1.x, p_tmp[i].p1.y},
				&point{p_tmp[0].p1.x, p_tmp[0].p1.y},
			))
		} else {
			ps = append(ps, newEdge(
				&point{p_tmp[i].p1.x, p_tmp[i].p1.y},
				&point{p_tmp[i+1].p1.x, p_tmp[i+1].p1.y},
			))
		}
	}

	if debug {
		println("=========================")
		for i := 0; i < L; i++ {
			println(ps[i].p1.x, ps[i].p1.y, ps[i].p2.x, ps[i].p2.y)
		}
	}

	// fill
	for i := xb1; i < xb2; i++ {
		for j := yb1; j < yb2; j++ {
			cnt := 0
			for k := 0; k < L; k++ {
				y1, y2 := ps[k].p1.y, ps[k].p2.y
				if y1 > y2 {
					y1, y2 = y2, y1
				}
				if j >= y1 && j < y2 && edge_function(i, j, ps[k]) < 0 {
					cnt++
				}
			}
			if cnt%2 == 1 {
				setPixel(i, j, color{255, 0, 0}, pixels)
			}
		}
	}
}

func drawRectangle(x, y, w, h, t int, c color, fill bool, pixels []byte) {
	// courner points
	x1 := float64(x - w/2)
	y1 := float64(y)

	x2 := float64(x + w/2)
	y2 := float64(y)

	x3 := float64(x + w/2)
	y3 := float64(y + h)

	x4 := float64(x - w/2)
	y4 := float64(y + h)

	// rotate cornet point
	t_rad := float64(t) * math.Pi / 180.0
	x1, y1 = rotate(x1, y1, float64(x), float64(y), t_rad)
	x2, y2 = rotate(x2, y2, float64(x), float64(y), t_rad)
	x3, y3 = rotate(x3, y3, float64(x), float64(y), t_rad)
	x4, y4 = rotate(x4, y4, float64(x), float64(y), t_rad)

	if debug {
		println("==========================")
		println(int(x1), int(y1))
		println(int(x2), int(y2))
		println(int(x3), int(y3))
		println(int(x4), int(y4))
	}

	if fill {
		Fill([]float64{x1, x2, x3, x4}, []float64{y1, y2, y3, y4}, c, pixels)
	}
	drawLine(int(x1), int(y1), int(x2), int(y2), color{255, 255, 255}, pixels)
	drawLine(int(x2), int(y2), int(x3), int(y3), color{255, 255, 255}, pixels)
	drawLine(int(x3), int(y3), int(x4), int(y4), color{255, 255, 255}, pixels)
	drawLine(int(x4), int(y4), int(x1), int(y1), color{255, 255, 255}, pixels)

}

func fractal(x, y, w, h, t, depth, max_depth, p int, pixels []byte) {
	if depth == max_depth {
		return
	}

	draw_h := h
	if depth == max_depth-1 {
		draw_h = p * h / 100
	}

	if depth == 0 {
		drawRectangle(x, y, w, draw_h, t, color{255, 0, 0}, true, pixels)
		fractal(x, y-h, w-4, h-40, t, depth+1, max_depth, p, pixels)
	} else {
		drawRectangle(x, y, w, draw_h, t-30, color{255, 0, 0}, true, pixels)
		drawRectangle(x, y, w, draw_h, t+30, color{255, 0, 0}, true, pixels)

		var x1, x2, y1, y2 float64
		x1, y1 = rotate(float64(x), float64(y-h), float64(x), float64(y), float64(t-180-30)*math.Pi/180)
		x2, y2 = rotate(float64(x), float64(y-h), float64(x), float64(y), float64(t-180+30)*math.Pi/180)

		var next_h = 0
		if h > 100 {
			next_h = h - 40
		} else if h > 50 {
			next_h = h - 20
		} else {
			next_h = h - 10
		}

		next_h = max(next_h, 10)
		next_w := max(w-4, 4)

		fractal(int(x1), int(y1), next_w, next_h, t-30, depth+1, max_depth, p, pixels)
		fractal(int(x2), int(y2), next_w, next_h, t+30, depth+1, max_depth, p, pixels)
	}
}

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Test Go SDL2", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, int32(winWidth), int32(winHeight), sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	renenderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renenderer.Destroy()

	texture, err := renenderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, int32(winWidth), int32(winHeight))
	if err != nil {
		panic(err)
	}
	defer texture.Destroy()

	// Create a pixel array
	// 4 bytes per pixel: 1 byte for each of Alpha, Blue, Green, Red
	pixels := make([]byte, winWidth*winHeight*4)

	running := true
	cnt := 0
	p := 0
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event.(type) {
			case *sdl.QuitEvent: // NOTE: Please use `*sdl.QuitEvent` for `v0.4.x` (current version).
				println("Quit")
				running = false
				break
			}
		}

		for i := 0; i < winWidth*winHeight*4; i++ {
			pixels[i] = 0
		}

		fractal(400, 580, 20, 150, 180, 0, (cnt%12)+1, p, pixels)

		texture.Update(nil, unsafe.Pointer(&pixels[0]), winWidth*4)
		renenderer.Copy(texture, nil, nil)
		renenderer.Present()

		if p == 100 {
			p = 0
			cnt++
		} else {
			p += 4
		}

		sdl.Delay(33)
	}
}
