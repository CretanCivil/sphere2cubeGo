package saver

import (
	"os"
	"image/jpeg"
	"path/filepath"
	"../worker"
	"github.com/disintegration/imaging"
	"image"
	"fmt"
)

func SaveTile(tileResult worker.TileResult, outPutDir string) error {

	err := os.Mkdir(outPutDir, os.FileMode(os.ModePerm))

	if err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	err = os.Mkdir(outPutDir+"/tiles", os.FileMode(os.ModePerm))

	finalPath := filepath.Join(outPutDir, tileResult.Tile.TileName+".jpg")

	f, err := os.Create(finalPath)

	if err != nil {
		return err
	}

	defer f.Close()

	err = jpeg.Encode(f, tileResult.Image, &jpeg.Options{100})

	if err != nil {
		return err
	}

	src512 := imaging.Resize(tileResult.Image, 512, 0, imaging.Linear)
	err = imaging.Save(src512, filepath.Join(outPutDir, "tiles/512_face"+tileResult.Tile.TileName+"_0_0.jpg"))

	src51k := imaging.Resize(tileResult.Image, 1024, 0, imaging.Linear)
	for  i := 0; i < 2;i++ {
		for  j := 0; j < 2;j++ {
			src512 = imaging.Crop(src51k,image.Rectangle{Min:image.Point{i*512,j*512},Max:image.Point{(i+1)*512,(j+1)*512}})
			err = imaging.Save(src512, filepath.Join(outPutDir, fmt.Sprintf("tiles/1k_face"+tileResult.Tile.TileName+"_%d_%d.jpg",i,j)))
		}
	}

	src2k := imaging.Resize(tileResult.Image, 2048, 0, imaging.Linear)
	for  i := 0; i < 4;i++ {
		for  j := 0; j < 4;j++ {
			src512 = imaging.Crop(src2k,image.Rectangle{Min:image.Point{i*512,j*512},Max:image.Point{(i+1)*512,(j+1)*512}})
			err = imaging.Save(src512, filepath.Join(outPutDir, fmt.Sprintf("tiles/2k_face"+tileResult.Tile.TileName+"_%d_%d.jpg",i,j)))
		}
	}

	src4k := imaging.Resize(tileResult.Image, 4096, 0, imaging.Linear)
	for  i := 0; i < 8;i++ {
		for  j := 0; j < 8;j++ {
			src512 = imaging.Crop(src4k,image.Rectangle{Min:image.Point{i*512,j*512},Max:image.Point{(i+1)*512,(j+1)*512}})
			err = imaging.Save(src512, filepath.Join(outPutDir, fmt.Sprintf("tiles/4k_face"+tileResult.Tile.TileName+"_%d_%d.jpg",i,j)))
		}
	}

	return nil
}
