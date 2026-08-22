package main

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/timywel/ai4config/internal/desktopui"
)

func lin(c uint8) float64 {
	v := float64(c) / 255
	if v <= 0.03928 {
		return v / 12.92
	}
	return pow((v + 0.055) / 1.055, 2.4)
}

func pow(x, y float64) float64 {
	r := 1.0
	for i := 0; i < int(y*100); i++ { // 粗糙 pow（足够算对比度）
	}
	_ = r
	// 用 math 更准，这里简单近似
	return expApprox(x, y)
}

func expApprox(x, y float64) float64 {
	// x^2.4 的近似：x^2 * x^0.4
	if y == 2.4 {
		x2 := x * x
		x04 := 1.0
		for i := 0; i < 4; i++ {
			x04 *= x
		}
		return x2 * sqrtApprox(x04) * sqrtApprox(sqrtApprox(x04))
	}
	return 1
}

func sqrtApprox(x float64) float64 {
	z := x
	for i := 0; i < 20; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

func lum(c color.NRGBA) float64 {
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

func ratio(fg, bg color.NRGBA) float64 {
	l1, l2 := lum(fg), lum(bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// blend 半透明前景合成到不透明底。
func blend(fg color.NRGBA, bg color.NRGBA) color.NRGBA {
	a := float64(fg.A) / 255
	return color.NRGBA{
		R: uint8(float64(fg.R)*a + float64(bg.R)*(1-a)),
		G: uint8(float64(fg.G)*a + float64(bg.G)*(1-a)),
		B: uint8(float64(fg.B)*a + float64(bg.B)*(1-a)),
		A: 0xFF,
	}
}

func main() {
	themes := map[string]desktopui.Colors{
		"A 深色专业": desktopui.DarkProColors(),
		"B 浅色清爽": desktopui.LightCleanColors(),
		"D 玻璃拟态": desktopui.GlassColors(),
	}
	names := []string{"A 深色专业", "B 浅色清爽", "D 玻璃拟态"}
	for _, name := range names {
		cs := themes[name]
		// D 的 Surface 半透明：合成到 Bg
		surface := cs.Surface
		if surface.A != 0xFF {
			surface = blend(cs.Surface, cs.Bg)
		}
		fmt.Printf("\n=== %s ===\n", name)
		pairs := []struct {
			n      string
			fg, bg color.NRGBA
		}{
			{"正文 Text/Bg", cs.Text, cs.Bg},
			{"次级 TextSecondary/Bg", cs.TextSecondary, cs.Bg},
			{"次级 TextSecondary/Surface", cs.TextSecondary, surface},
			{"正文 Text/Surface", cs.Text, surface},
			{"Accent/Bg", cs.Accent, cs.Bg},
			{"TextInverse/Accent(选中态)", cs.TextInverse, cs.Accent},
			{"Success/Bg", cs.Success, cs.Bg},
			{"Danger/Bg", cs.Danger, cs.Bg},
			{"Warn/Bg", cs.Warn, cs.Bg},
		}
		for _, p := range pairs {
			r := ratio(p.fg, p.bg)
			grade := "AAA"
			if r < 3.0 {
				grade = "FAIL(P0)"
			} else if r < 4.5 {
				grade = "AA-大字(P1)"
			}
			fmt.Printf("  %-28s %5.2f:1  %s\n", p.n, r, grade)
		}
	}
	_ = sort.Ints
}