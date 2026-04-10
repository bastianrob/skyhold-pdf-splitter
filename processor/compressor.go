package processor

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"runtime"
	"sync"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"golang.org/x/image/draw"
)

// compressContextImages traverses all objects in the Context and rewrites 
// DCTDecode (JPEG) Image Streams to a lower quality and optionally smaller scale.
// This operates without external C dependencies and utilizes multiple cores.
func compressContextImages(ctx *model.Context, quality int, concurrency int, scale int, onProgress func(current, total int)) error {
	if quality <= 0 || quality > 100 {
		quality = 60 // default to 60
	}
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	opts := &jpeg.Options{Quality: quality}
	totalObjects := len(ctx.Table)

	type job struct {
		objNr int
		entry *model.XRefTableEntry
		sd    types.StreamDict
	}

	type result struct {
		objNr int
		entry *model.XRefTableEntry
		sd    types.StreamDict
		saved bool
	}

	jobs := make(chan job)
	results := make(chan result)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				imgData := j.sd.Raw
				if len(imgData) == 0 {
					results <- result{objNr: j.objNr, entry: j.entry, sd: j.sd, saved: false}
					continue
				}

				// Decode using standard library
				img, err := jpeg.Decode(bytes.NewReader(imgData))
				if err != nil {
					results <- result{objNr: j.objNr, entry: j.entry, sd: j.sd, saved: false}
					continue
				}

				// Re-encode with lower quality and optional scaling
				if scale > 0 && scale < 100 {
					newWidth := int(float64(img.Bounds().Dx()) * float64(scale) / 100.0)
					newHeight := int(float64(img.Bounds().Dy()) * float64(scale) / 100.0)
					if newWidth > 0 && newHeight > 0 {
						dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
						draw.BiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
						img = dst
						j.sd.Dict["Width"] = types.Integer(newWidth)
						j.sd.Dict["Height"] = types.Integer(newHeight)
					}
				}

				var buf bytes.Buffer
				err = jpeg.Encode(&buf, img, opts)
				if err != nil {
					results <- result{objNr: j.objNr, entry: j.entry, sd: j.sd, saved: false}
					continue
				}

				// Check if compression was beneficial
				if buf.Len() < len(imgData) {
					newData := buf.Bytes()
					j.sd.Raw = newData
					newLen := int64(len(newData))
					j.sd.StreamLength = &newLen
					j.sd.Dict["Length"] = types.Integer(newLen)
					j.entry.Object = j.sd
					results <- result{objNr: j.objNr, entry: j.entry, sd: j.sd, saved: true}
				} else {
					results <- result{objNr: j.objNr, entry: j.entry, sd: j.sd, saved: false}
				}
			}
		}()
	}

	// Identification pass and job submission
	go func() {
		for objNr, entry := range ctx.Table {
			sd, ok := entry.Object.(types.StreamDict)
			if !ok {
				if sdp, ok := entry.Object.(*types.StreamDict); ok {
					sd = *sdp
				} else {
					results <- result{objNr: objNr, entry: entry, saved: false}
					continue
				}
			}

			subtype := sd.Dict.Subtype()
			if subtype == nil || *subtype != "Image" {
				results <- result{objNr: objNr, entry: entry, saved: false}
				continue
			}

			var filterName string
			if f, found := sd.Dict["Filter"]; found {
				if name, ok := f.(types.Name); ok {
					filterName = string(name)
				} else if arr, ok := f.(types.Array); ok && len(arr) > 0 {
					if name, ok := arr[0].(types.Name); ok {
						filterName = string(name)
					}
				}
			}

			if filterName == "DCTDecode" {
				jobs <- job{objNr: objNr, entry: entry, sd: sd}
			} else {
				results <- result{objNr: objNr, entry: entry, saved: false}
			}
		}
		close(jobs)
	}()

	// Collector goroutine to apply results sequentially (preventing map race)
	done := make(chan bool)
	go func() {
		processed := 0
		modifiedTable := make(map[int]*model.XRefTableEntry)
		for res := range results {
			processed++
			if onProgress != nil && processed%10 == 0 {
				onProgress(processed, totalObjects)
			}
			if res.saved {
				modifiedTable[res.objNr] = res.entry
			}
			if processed == totalObjects {
				break
			}
		}
		// Apply modifications after iteration is complete
		for objNr, entry := range modifiedTable {
			ctx.Table[objNr] = entry
		}
		done <- true
	}()

	wg.Wait()
	<-done

	if onProgress != nil {
		onProgress(totalObjects, totalObjects)
	}

	return nil
}
