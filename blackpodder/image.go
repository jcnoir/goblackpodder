package main

import (

	//"github.com/nfnt/resize"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
)

func ImageRead(ImageFile string) (myImage image.Image, err error) {
	// open "test.jpg"
	file, err := os.Open(ImageFile)
	defer file.Close()
	// decode jpeg into image.Image
	img, _, err := image.Decode(file)
	return img, err
}

func Formatpng(img image.Image, filepath string) (err error) {
	out, err := os.Create(filepath)
	defer out.Close()
	return png.Encode(out, img)
}

func Formatjpg(img image.Image, filepath string) (err error) {
	out, err := os.Create(filepath)
	defer out.Close()
	return jpeg.Encode(out, img, &jpeg.Options{90})

}

func Formatgif(img image.Image, filepath string) (err error) {
	out, err := os.Create(filepath)
	defer out.Close()
	return gif.Encode(out, img, &gif.Options{})

}
