package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mpiannucci/peakdetect"
	log "github.com/schollz/logger"

	"github.com/schollz/seamlessloop/src/seamless"
	sox_ "github.com/schollz/seamlessloop/src/sox"
)

var sox sox_.Sox

func main() {
	sox = sox_.New()
	for i := 1; i < 16; i++ {
		ExtractTrack(fmt.Sprintf("E-Lab - X-Static Goldmine CD1/Track%02d.wav", i))
	}
	// GuessBPM("temp/split013.wav")
	sox.Clean()
}

func ExtractTrack(fname string) (err error) {
	os.RemoveAll("temp")
	os.MkdirAll("temp", os.ModePerm)
	err = sox.SplitSilence(fname, "temp/split.wav", 0.5, 0.4)
	if err != nil {
		return
	}
	for i := 1; i < 1000; i++ {
		fname2 := fmt.Sprintf("temp/split%03d.wav", i)
		if _, err := os.Stat(fname2); errors.Is(err, os.ErrNotExist) {
			break
		}
		_, filename := path.Split(fname)
		filename = strings.TrimSuffix(filename, filepath.Ext(filename))
		_, err = ProcessSplit(fname2, fmt.Sprintf("%s_%02d", filename, i))
		if err != nil {
			log.Errorf("problem with %s: %s", fname2, err.Error())
		}
	}

	return
}

func ProcessSplit(fname string, fnameOut string) (fnameFinal string, err error) {
	fmt.Println(fname)
	// tempo, err := sox.Tempo(fname)
	// if err != nil {
	// 	return
	// }
	// tempo = math.Round(tempo)
	// log.Debugf("%s: %2.1f", fname, tempo)
	tempo, err := GuessBPM(fname)
	if err != nil {
		return
	}

	fname2, bpm, beats, err := seamless.Do(fname, true, 0, tempo)
	if err != nil {
		return
	}

	fnameFinal = fmt.Sprintf("%s_beats%d_bpm%d.flac", fnameOut, beats, bpm)
	log.Infof("%s -> %s", fname, fnameFinal)
	err = os.Rename(fname2, fnameFinal)
	return
}

func GuessBPM(fname string) (bpm float64, err error) {
	c1 := exec.Command("sox", fname, "-t", "raw", "-r", "44100", "-e", "float", "-c", "1", "-")
	c2 := exec.Command("bpm", "-g", "1.dat", "-m", "85", "-x", "179")

	r, w := io.Pipe()
	c1.Stdout = w
	c2.Stdin = r

	var b2 bytes.Buffer
	c2.Stdout = &b2

	c1.Start()
	c2.Start()
	c1.Wait()
	w.Close()
	c2.Wait()
	log.Debugf("[%s] guessed initial bpm: %s", fname, b2.String())
	r.Close()

	x, y, err := GetData("1.dat")
	if err != nil {
		log.Error(err)
		return
	}
	duration, err := sox.Length(fname)
	if err != nil {
		log.Error(err)
		return
	}
	for threshold := 0.005; threshold < 1; threshold += 0.005 {
		mini, _, _, _ := peakdetect.PeakDetect(y, threshold)
		if len(mini) > 10 {
			continue
		}
		lowestVal := 100000.0
		lowestBPM := 0.0
		closestVal := 10000.0
		closestBPM := 0.0
		for _, i := range mini {
			if y[i] < lowestVal {
				lowestBPM = x[i]
				lowestVal = y[i]
			}
			beats := duration / (60 / x[i])
			log.Debugf("[%s] %2.3f %2.3f", fname, x[i], beats)
			val := beats - 8
			if val < closestVal && val <= 1 && val > 0 {
				closestVal = val
				closestBPM = x[i]
			}
			val = beats - 4
			if val < closestVal && val <= 1 && val > 0 {
				closestVal = val
				closestBPM = x[i]
			}
		}
		bpm = closestBPM
		if bpm == 0 {
			bpm = lowestBPM
		}
		break
	}

	log.Debugf("[%s] final bpm: %2.3f", fname, bpm)
	return
}

func GetData(fname string) (x []float64, y []float64, err error) {
	file, err := os.Open(fname)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		foo := strings.Fields(scanner.Text())
		if len(foo) != 2 {
			continue
		}
		x0, errX := strconv.ParseFloat(foo[0], 64)
		y0, errY := strconv.ParseFloat(foo[1], 64)
		if errX == nil && errY == nil {
			x = append(x, x0)
			y = append(y, y0)
		}
	}

	if err = scanner.Err(); err != nil {
		return
	}
	return
}
func run(args ...string) (string, string, error) {
	log.Trace(strings.Join(args, " "))
	baseCmd := args[0]
	cmdArgs := args[1:]
	cmd := exec.Command(baseCmd, cmdArgs...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		log.Errorf("%s -> '%s'", strings.Join(args, " "), err.Error())
		log.Error(outb.String())
		log.Error(errb.String())
	}
	return outb.String(), errb.String(), err
}
