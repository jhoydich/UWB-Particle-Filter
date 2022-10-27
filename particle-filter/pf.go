package particlefilter

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"uwb-pf/readings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"gonum.org/v1/gonum/stat/distuv"
)

type Particle struct {
	X      float64
	Y      float64
	Weight float64
}

func (p *Particle) UpdateWeight(weight float64) {
	p.Weight = weight
}

type ParticleFilter struct {
	NumSamples      int
	PercentResample float64
	ListParticles   []*Particle
	Sigma           float64
	XLimit          float64
	YLimit          float64
	Anchors         map[string][]float64
	EstimatedX      float64
	EstimatedY      float64
	MaxWeight       float64
	iteration       int
	printIter       bool
}

func CreatePF(numSamps int, percResamp, sigma, xLimit, yLimit float64) *ParticleFilter {
	anchors := make(map[string][]float64)
	anchors["A9CF"] = []float64{0, 3.04}
	anchors["F95B"] = []float64{0, 0}

	pf := &ParticleFilter{
		NumSamples:      numSamps,
		PercentResample: percResamp,
		ListParticles:   []*Particle{},
		Sigma:           sigma,
		XLimit:          xLimit,
		YLimit:          yLimit,
		Anchors:         anchors,
		MaxWeight:       0.0,
		iteration:       0,
		printIter:       true,
	}

	// creating initial random samples
	pf.createSampleList()

	return pf
}

// check if weight is greater than
func (pf *ParticleFilter) checkAndSetMaxWeight(weight float64, override bool) {
	if weight > pf.MaxWeight {
		pf.MaxWeight = weight
	}

	// when we want to reset to zero
	if override {
		pf.MaxWeight = weight
	}
}

// createParticle creates a particle for use in a pf
func (pf *ParticleFilter) createParticle() Particle {
	x := rand.Float64() * pf.XLimit
	y := rand.Float64() * pf.YLimit
	p := Particle{X: x, Y: y, Weight: 0}

	return p
}

func (pf *ParticleFilter) createSampleList() {
	for i := 0; i < pf.NumSamples; i++ {
		p := pf.createParticle()
		pf.ListParticles = append(pf.ListParticles, &p)
	}
}

// use pdf to get weight of particle's X or Y location
func (pf *ParticleFilter) calculateNormDist(x, mu float64) float64 {
	xMuExponent := math.Pow(((x - mu) / pf.Sigma), 2.0)
	eulerExponent := math.Pow(math.E, (-.5 * xMuExponent))
	return ((1 / (pf.Sigma * math.Sqrt(2*math.Pi))) * eulerExponent)
}

// CalculateWeights calculates the weight of each particle in the list by comparing the reading values
// to the location of the particles
func (pf *ParticleFilter) CalculateWeights(readings []readings.Reading) {
	pf.checkAndSetMaxWeight(0.0, true)
	for i := range pf.ListParticles {
		p := pf.ListParticles[i]

		// iterate over readings from each anchor
		// calculate length from each particle to anchor
		// assign weight to particle
		weights := []float64{}

		for i := range readings {
			r := readings[i]
			xAndy := pf.Anchors[r.Anchor]
			//√((x2 – x1)² + (y2 – y1)²)
			dist := math.Sqrt(math.Pow(p.X-xAndy[0], 2) + math.Pow(p.Y-xAndy[1], 2))
			weight := pf.calculateNormDist(r.Dist, dist)
			weights = append(weights, weight)
		}

		newWeight := 1.0
		// may want to explore different ways to combine weights
		for i := range weights {
			newWeight *= weights[i]
		}
		pf.checkAndSetMaxWeight(newWeight, false)
		p.UpdateWeight(newWeight)
	}
}

// ResampleAndFuzz uses a resampling wheel to choose high weight particles
// fuzzes them to right around the chosen particles location
// and adds random particles in case we did not converge on the correct  answer
func (pf *ParticleFilter) ResampleAndFuzz() {
	pf.iteration += 1
	numResample := float64(pf.NumSamples) * pf.PercentResample
	numIntResample := int(numResample)
	numSpoof := pf.NumSamples - numIntResample
	newParticleList := []*Particle{}

	// our new particle estimates
	xLoc := 0.0
	yLoc := 0.0

	// Resample wheel code reused from OMSCS RAIT course Particle Filter Section
	// Originally developed by Sebastian Thrun
	for i := 0; i < numIntResample; i++ {
		beta := rand.Float64() * pf.MaxWeight * 2
		startIdx := rand.Intn(numIntResample)
		for beta > 0 {
			p := pf.ListParticles[startIdx]
			if p.Weight > beta {

				// creating normal dists around particle x and y
				// adding noise to that and reinserting into the list of particles
				norm := distuv.Normal{
					Mu:    0,
					Sigma: .1,
				}

				xNoise := norm.Rand()
				newX := p.X + xNoise

				yNoise := norm.Rand()
				newY := p.Y + yNoise

				newP := &Particle{
					X:      newX,
					Y:      newY,
					Weight: 0,
				}

				xLoc += newX
				yLoc += newY

				newParticleList = append(newParticleList, newP)
			}

			beta -= p.Weight
			startIdx = (startIdx + 1) % (pf.NumSamples)
		}
	}

	// taking average of x and y locations and setting to estimated x and y locations
	xLoc /= float64(numIntResample)
	yLoc /= float64(numIntResample)

	pf.EstimatedX = xLoc
	pf.EstimatedY = yLoc

	// adding some random samples incase we did not converge on correct answer
	for i := 0; i < numSpoof; i++ {
		p := pf.createParticle()
		newParticleList = append(newParticleList, &p)
	}

	pf.ListParticles = newParticleList

	// if we want to generate image files
	if pf.printIter {
		// making a plot since go doesn't really have a good live plotting library
		go func(l []*Particle, xLoc, yLoc float64) {
			pts := make(plotter.XYs, pf.NumSamples)

			for i := range l {
				pts[i].X = l[i].X
				pts[i].Y = l[i].Y
			}

			// Create a new plot, set its title and
			// axis labels.
			plt := plot.New()

			plt.Title.Text = "UWB Particle Filter Locator"
			plt.X.Label.Text = "X"
			plt.Y.Label.Text = "Y"
			// Draw a grid behind the data
			plt.Add(plotter.NewGrid())

			// Make a scatter plotter and set its style.
			s, err := plotter.NewScatter(pts)
			if err != nil {
				panic(err)
			}

			s.GlyphStyle.Color = color.RGBA{R: 0, B: 0, A: 255}
			plt.Add(s)

			estimatedPt := make(plotter.XYs, 1)
			estimatedPt[0].X = xLoc
			estimatedPt[0].Y = yLoc

			s2, err := plotter.NewScatter(estimatedPt)
			if err != nil {
				panic(err)
			}

			s2.GlyphStyle.Color = color.RGBA{R: 255, B: 0, A: 255}
			plt.Add(s2)

			plt.X.Max = pf.XLimit
			plt.Y.Max = pf.YLimit

			// Save the plot to a PNG file.
			fName := fmt.Sprintf("plots/scatter%d.png", pf.iteration)
			if err := plt.Save(4*vg.Inch, 4*vg.Inch, fName); err != nil {
				panic(err)
			}
			//fmt.Println("Saved", fName)
		}(pf.ListParticles, xLoc, yLoc)
	}

}
