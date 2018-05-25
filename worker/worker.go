package worker

import (
	"../cache"
	"log"
	"image"
	"os"
	"image/jpeg"
	"image/color"
	"io"
	"math"
)

/*
const (
	TileUp    = "up"
	TileDown  = "down"
	TileFront = "front"
	TileRight = "right"
	TileBack  = "back"
	TileLeft  = "left"
)
*/
const (
	TileUp    = "0"
	TileDown  = "5"
	TileFront = "1"
	TileRight = "2"
	TileBack  = "3"
	TileLeft  = "4"
)

type TileResult struct {
	Tile  Tile
	Image image.Image
}

type Tile struct {
	TileSize int
	TileName string
}

func (tile *Tile) getHalfSize() float64 {
	return float64(tile.TileSize-1) / 2
}

// Pixel struct
type Pixel struct {
	R uint32
	G uint32
	B uint32
	A uint32
}

func (pixel *Pixel) pixelToRGBA() color.Color {
	return color.RGBA64{uint16(pixel.R), uint16(pixel.G), uint16(pixel.B), uint16(pixel.A)}
}

// img.At(x, y).RGBA() returns four uint32 values; we want a Pixel
func rgbaToPixel(r uint32, g uint32, b uint32, a uint32) Pixel {
	return Pixel{r, g, b, a}
}

// Get the bi-dimensional pixel array
func getPixels(file io.Reader) ([][]Pixel, error) {
	img, err := jpeg.Decode(file)

	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	var pixels [][]Pixel
	for y := 0; y < height; y++ {
		var row []Pixel
		for x := 0; x < width; x++ {
			row = append(row, rgbaToPixel(img.At(x, y).RGBA()))
		}
		pixels = append(pixels, row)
	}

	return pixels, nil
}

func updatePhi(half_size float64, phi float64, major_dir int, minor_dir int, major_m float64, major_p float64, minor_m float64, minor_p float64) float64 {

	if float64(major_dir) < half_size {
		phi = phi + major_m
	} else if float64(major_dir) > half_size {
		phi = phi + major_p
	} else if float64(minor_dir) < half_size {
		phi = minor_m
	} else {
		phi = minor_p
	}

	return phi
}

func phi2Width(width int, phi float64) float64 {
	x := 0.5 * float64(width) * (phi/math.Pi + 1)

	if x < 1 {
		x += float64(width)
	} else if x > float64(width) {
		x -= float64(width)
	}

	return x
}

func theta2Height(height int, theta float64) float64 {
	return float64(height) * theta / math.Pi
}

func between(value ,min, max int) int {
	value = int(math.Max(float64(value),float64(min)))
	value = int(math.Min(float64(value),float64(max)))
	return value
}
func copyPixelBilinear(originalImage [][]Pixel, x float64, y float64) Pixel {

	/*-----p00(xl,yl)----x,y-----p01(xr,y1)----*/

	/*-----p10(xl,yr)-----x,y----p11(xr,yr)----*/

	xl := between(int(math.Floor(x)),0,len(originalImage[0]) - 1)

	xr := between(int(math.Ceil(x)),0,len(originalImage[0]) - 1)
	xf := x - float64(xl)

	yl := between(int(math.Floor(y)),0,len(originalImage) -1 )
	yr := between(int(math.Ceil(y)),0,len(originalImage) -1)
	yf := y - float64(yl)

	p00 := originalImage[yl][xl]
	p10 := originalImage[yr][xl]
	p01 := originalImage[yl][xr]
	p11 := originalImage[yr][xr]

	pixel := Pixel{}

	r0 := float64(p00.R)*(1-xf) + float64(p01.R)*xf
	r1 := float64(p10.R)*(1-xf) + float64(p11.R)*xf
	pixel.R = uint32(r0*(1-yf) + r1*yf)

	r0 = float64(p00.A)*(1-xf) + float64(p01.A)*xf
	r1 = float64(p10.A)*(1-xf) + float64(p11.A)*xf
	pixel.A = uint32(r0*(1-yf) + r1*yf)

	r0 = float64(p00.G)*(1-xf) + float64(p01.G)*xf
	r1 = float64(p10.G)*(1-xf) + float64(p11.G)*xf
	pixel.G = uint32(r0*(1-yf) + r1*yf)

	r0 = float64(p00.B)*(1-xf) + float64(p01.B)*xf
	r1 = float64(p10.B)*(1-xf) + float64(p11.B)*xf
	pixel.B = uint32(r0*(1-yf) + r1*yf)



	return pixel
}

func processCords(tileX int, tileY int, originalImage [][]Pixel, tile Tile, mathCache cache.CacheAngles) Pixel {

	theta := 0.0
	phi := 0.0

	sphereHeight, sphereWidth := len(originalImage), len(originalImage[0])

	if tile.TileName == TileUp {
		theta = mathCache.ZP[tileY][tileX]
		phi = mathCache.PHI[tileX][tileY]
		phi = updatePhi(tile.getHalfSize(), phi, tileY, tileX, math.Pi, 0, -math.Pi/2, math.Pi/2)
	} else if tile.TileName == TileDown {
		theta = mathCache.ZM[tileY][tileX]
		phi = mathCache.PHI[tileX][tile.TileSize-tileY-1]
		phi = updatePhi(tile.getHalfSize(), phi, tileY, tileX, 0, math.Pi, -math.Pi/2, math.Pi/2)
	} else if tile.TileName == TileFront {
		theta = mathCache.XYPM[tile.TileSize-tileY-1][tile.TileSize-tileX-1]
		phi = mathCache.PHI[tileX][tile.TileSize-1] //tile_x, tile_size - 1
		phi = updatePhi(tile.getHalfSize(), phi, tileY, tileX, 0, 0, -math.Pi/2, math.Pi/2)
	} else if tile.TileName == TileRight {
		theta = mathCache.XYPM[tile.TileSize-tileY-1][tile.TileSize-tileX-1]
		phi = mathCache.PHI[tile.TileSize-1][tile.TileSize-tileX-1]
		phi = updatePhi(tile.getHalfSize(), phi, tileX, tileY, 0, math.Pi, math.Pi/2, math.Pi/2)
	} else if tile.TileName == TileBack {
		theta = mathCache.XYPM[tile.TileSize-tileY-1][tile.TileSize-tileX-1]
		phi = mathCache.PHI[tileX][tile.TileSize-1] + math.Pi
	} else if tile.TileName == TileLeft {
		theta = mathCache.XYPM[tile.TileSize-tileY-1][tile.TileSize-tileX-1]
		phi = mathCache.PHI[tile.TileSize-1][tile.TileSize-tileX-1]
		phi = updatePhi(tile.getHalfSize(), phi, tileX, tileY, math.Pi, 0, -math.Pi/2, -math.Pi/2)
	}

	spX := phi2Width(sphereWidth, phi)
	spY := theta2Height(sphereHeight, theta)


	return copyPixelBilinear(originalImage,spX,spY)
	//return originalImage[spY][spX]
}

func Worker(tile Tile, mathCache cache.CacheAngles, originalImagePath string, done chan TileResult) {
	log.Printf("Process for tile %v --> started", tile.TileName)
	tileImage := image.NewRGBA(image.Rect(0, 0, tile.TileSize, tile.TileSize))
	reader, err := os.Open(originalImagePath)

	if err != nil {
		panic(err)
	}

	defer reader.Close()

	originalPixels, err := getPixels(reader)

	if err != nil {
		panic(err)
	}

	sphereHeight, sphereWidth := len(originalPixels), len(originalPixels[0])

	if sphereWidth/sphereHeight != 2 {
		log.Fatal("Panorama should has 2:1 aspect ratio")
		os.Exit(2)
	}

	for tileY := 0; tileY < tile.TileSize; tileY++ {
		for tileX := 0; tileX < tile.TileSize; tileX++ {
			pixelToMove := processCords(tileX, tileY, originalPixels, tile, mathCache)
			colorPixel := pixelToMove.pixelToRGBA()
			tileImage.Set(tileX, tileY, colorPixel)
		}
	}

	result := TileResult{Tile: tile, Image: tileImage}

	done <- result

}
