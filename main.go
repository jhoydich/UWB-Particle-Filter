package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	particlefilter "uwb-pf/particle-filter"
	"uwb-pf/readings"
)

func main() {
	// creating our particle filter object
	pf := particlefilter.CreatePF(100, .9, .5, 5.0, 5.0)

	readingChan := make(chan readings.Reading, 30)
	evalChan := make(chan []readings.Reading)

	// where readings are stored and average value is calculated
	// sent to evaluator where pf works its magic
	go func(readChan chan readings.Reading, evalChan chan []readings.Reading) {

		// TODO: handle anchor readings dynamically in a map or some other ds
		anch1 := []readings.Reading{}
		anch2 := []readings.Reading{}
		for {
			reading := <-readChan
			//fmt.Println("Recieved reading")
			switch reading.Anchor {
			case "A9CF":
				anch1 = append(anch1, reading)
			case "F95B":
				anch2 = append(anch2, reading)
			}

			if len(anch1) >= 5 && len(anch2) >= 5 {
				//fmt.Println("at least 5 in each")
				avgAnch1 := 0.0
				avgAnch2 := 0.0
				for i := range anch1 {
					avgAnch1 += anch1[i].Dist
				}

				avgAnch1 = avgAnch1 / float64(len(anch1))
				avgAnch1Reading := readings.Reading{
					Anchor: anch1[0].Anchor,
					Dist:   avgAnch1,
				}

				for i := range anch2 {
					avgAnch2 += anch2[i].Dist
				}

				avgAnch2 = avgAnch2 / float64(len(anch2))
				avgAnch2Reading := readings.Reading{
					Anchor: anch2[0].Anchor,
					Dist:   avgAnch2,
				}

				// resetting arrays to no readings
				anch1 = []readings.Reading{}
				anch2 = []readings.Reading{}

				evalChan <- []readings.Reading{avgAnch1Reading, avgAnch2Reading}
			}
		}
	}(readingChan, evalChan)

	// evaluation channel
	go func(evaluationChan chan []readings.Reading) {
		for {
			readings := <-evaluationChan
			//fmt.Println("Calculating weights")
			pf.CalculateWeights(readings)
			//fmt.Println("Resampling")
			pf.ResampleAndFuzz()
			fmt.Println("X:", pf.EstimatedX, "Y:", pf.EstimatedY)
		}
	}(evalChan)

	// reading individual readings from text file
	f, err := os.Open("readings.txt")
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var reading readings.Reading
		text := scanner.Text()
		json.Unmarshal([]byte(text), &reading)
		//fmt.Println(reading)
		readingChan <- reading
	}

	// just wait forever for the time being

	for {
		time.Sleep(5 * time.Second)
		fmt.Println(" cntrl + c to exit...")
	}

}
