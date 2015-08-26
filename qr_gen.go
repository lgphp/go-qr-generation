package main

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/nfnt/resize"
)

func markWithLogo(r *http.Request, qrcode image.Image) image.Image {
	logoUrl := r.URL.Query().Get("logo")
	if logoUrl == "" {
		return qrcode
	} else {
		log.Println("Begin to mark with logo:", logoUrl)
		resp, err := http.Get(logoUrl)
		if err != nil {
			log.Println(err)
			return qrcode
		}
		if resp.StatusCode != 200 {
			log.Println("Failed to load logo from:" + logoUrl)
			return qrcode
		}
		filetype := strings.ToLower(resp.Header.Get("Content-Type"))
		if filetype != "image/jpeg" && filetype != "image/jpg" && filetype != "image/png" && filetype != "image/gif" {
			log.Println("Unsupported logo type:", filetype)
			return qrcode
		}
		defer resp.Body.Close()
		logo, _, err := image.Decode(resp.Body)
		if err != nil {
			log.Println(err)
			return qrcode
		}
		scale := uint(float64(qrcode.Bounds().Dx()) * 0.2)
		logo = resize.Resize(scale, 0, logo, resize.Lanczos3)
		offset := image.Pt((qrcode.Bounds().Dx()-logo.Bounds().Dx())/2, (qrcode.Bounds().Dy()-logo.Bounds().Dy())/2)
		b := qrcode.Bounds()
		m := image.NewNRGBA(b)
		draw.Draw(m, b, qrcode, image.ZP, draw.Src)
		draw.Draw(m, qrcode.Bounds().Add(offset), logo, image.ZP, draw.Over)
		return m
	}
}

func QrGenerator(w http.ResponseWriter, r *http.Request) {
	data := r.URL.Query().Get("data")
	if data == "" {
		http.Error(w, "Param data is required", http.StatusBadRequest)
		return
	}

	s, err := url.QueryUnescape(data)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	log.Println("Begin to generate qr for data:", s)
	code, err := qr.Encode(s, qr.L, qr.Auto)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	size := r.URL.Query().Get("size")
	if size == "" {
		size = "250"
	}
	intsize, err := strconv.Atoi(size)
	if err != nil {
		intsize = 250
	}

	if intsize < 21 {
		http.Error(w, "Can not generate an qr code image smaller than 21x21", http.StatusInternalServerError)
		return
	}

	if intsize > 500 {
		http.Error(w, "The request size is too big, please set a size smaller than 500.", http.StatusInternalServerError)
		return
	}

	// Scale the barcode to the appropriate size
	code, err = barcode.Scale(code, intsize, intsize)
	if err != nil {
		panic(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	buffer := new(bytes.Buffer)

	qrImage := markWithLogo(r, code)

	if err := png.Encode(buffer, qrImage); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))

	if _, err := w.Write(buffer.Bytes()); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

// Get the Port from the environment so we can run on Heroku
func GetPort() string {
	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "4747"
		fmt.Println("INFO: No PORT environment variable detected, defaulting to " + port)
	}
	return ":" + port
}

// The main functions
func main() {
	http.HandleFunc("/qr", QrGenerator)
	err := http.ListenAndServe(GetPort(), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
