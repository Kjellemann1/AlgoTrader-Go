
package indicator

import (
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)

// Let's assume the size of your arrays is 100

// RSI using static arrays
// TODO: Need to verify the correctness of this function
func RSI(inReal [constant.WINDOW_SIZE]float64, inTimePeriod int) [constant.WINDOW_SIZE]float64 {
	var outReal [constant.WINDOW_SIZE]float64

	if inTimePeriod < 2 {
		return outReal
	}

	// Variable declarations
	tempValue1 := 0.0
	tempValue2 := 0.0
	outIdx := inTimePeriod
	today := 0
	prevValue := inReal[today]
	prevGain := 0.0
	prevLoss := 0.0
	today++

	// Initial calculations
	for i := inTimePeriod; i > 0; i-- {
		tempValue1 = inReal[today]
		today++
		tempValue2 = tempValue1 - prevValue
		prevValue = tempValue1
		if tempValue2 < 0 {
			prevLoss -= tempValue2
		} else {
			prevGain += tempValue2
		}
	}

	prevLoss /= float64(inTimePeriod)
	prevGain /= float64(inTimePeriod)

	if today > 0 {
		tempValue1 = prevGain + prevLoss
		if tempValue1 != 0 {
			outReal[outIdx] = 100.0 * (prevGain / tempValue1)
		} else {
			outReal[outIdx] = 0.0
		}
		outIdx++
	} else {
		for today < 0 {
			tempValue1 = inReal[today]
			tempValue2 = tempValue1 - prevValue
			prevValue = tempValue1
			prevLoss *= float64(inTimePeriod - 1)
			prevGain *= float64(inTimePeriod - 1)
			if tempValue2 < 0 {
				prevLoss -= tempValue2
			} else {
				prevGain += tempValue2
			}
			prevLoss /= float64(inTimePeriod)
			prevGain /= float64(inTimePeriod)
			today++
		}
	}

	// Main loop
	for today < constant.WINDOW_SIZE {
		tempValue1 = inReal[today]
		today++
		tempValue2 = tempValue1 - prevValue
		prevValue = tempValue1
		prevLoss *= float64(inTimePeriod - 1)
		prevGain *= float64(inTimePeriod - 1)
		if tempValue2 < 0 {
			prevLoss -= tempValue2
		} else {
			prevGain += tempValue2
		}
		prevLoss /= float64(inTimePeriod)
		prevGain /= float64(inTimePeriod)
		tempValue1 = prevGain + prevLoss
		if tempValue1 != 0 {
			outReal[outIdx] = 100.0 * (prevGain / tempValue1)
		} else {
			outReal[outIdx] = 0.0
		}
		outIdx++
	}

	return outReal
}
