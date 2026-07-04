// SPDX-License-Identifier: MIT
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/migueldecimal128/decimal128-go"
)

func main() {
	sumT := decimal128.Zero()
	sumB := decimal128.Zero()
	sumD := decimal128.Zero()

	baserate := decimal128.MustParse("0.0013")
	distrate := decimal128.MustParse("0.00894")
	basetax := decimal128.MustParse("0.0675")
	disttax := decimal128.MustParse("0.0341")
	quantumHundredth := decimal128.MustParse("0.01")

	ctxHalfEven := decimal128.NewContext()
	ctxTowardZero := decimal128.NewContextWithRounding(decimal128.TowardZero)

	// Default input: the 1e6 primary workload from the decimal128-resources
	// submodule, resolved from this source file's location (cwd-independent).
	_, thisFile, _, _ := runtime.Caller(0)
	resourcesDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "resources")
	filein := filepath.Join(resourcesDir, "telco", "expon180.1e6b")
	fileou := "telco.outgo"
	calc := true
	tax := true

	args := os.Args[1:]
	j := 0
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			switch args[i] {
			case "-nocalc":
				calc = false
			case "-notax":
				tax = false
			default:
				fmt.Printf("Flag '%s' ignored\n", args[i])
			}
		} else {
			j++
			if j == 1 {
				filein = args[i]
			} else if j == 2 {
				fileou = args[i]
			} else {
				fmt.Printf("Extra parameter '%s' ignored\n", args[i])
			}
		}
	}

	fmt.Printf("telco decimal128-go benchmark; processing '%s'\n", filein)

	_ = os.Remove(fileou)

	inp, err := os.Open(filein)
	if err != nil {
		fmt.Printf("Error: file '%s' not found\n", filein)
		os.Exit(1)
	}
	defer inp.Close()

	oup, err := os.Create(fileou)
	if err != nil {
		fmt.Printf("Error: file '%s' could not be opened\n", fileou)
		os.Exit(1)
	}
	defer oup.Close()

	reader := bufio.NewReader(inp)
	writer := bufio.NewWriter(oup)

	tStart := time.Now()
	numbers := 0
	var duration uint64

	for {
		err = binary.Read(reader, binary.BigEndian, &duration)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error: cannot read file '%s': %v\n", filein, err)
			os.Exit(1)
		}
		numbers++

		var outStr string
		if calc {
			calltype := int(duration & 0x01)
			n := decimal128.FromInt64(int64(duration))
			var rate decimal128.Decimal128
			if calltype == 0 {
				rate = baserate
			} else {
				rate = distrate
			}

			p := rate.Mul(n)

			// Round price to 2 decimal places using round-half-even
			p = ctxHalfEven.Quantize(p, quantumHundredth)

			t := p
			if tax {
				// Truncate basic tax to 2 decimal places (round toward zero)
				b := p.Mul(basetax)
				b = ctxTowardZero.Quantize(b, quantumHundredth)
				sumB = sumB.Add(b)
				t = t.Add(b)

				if calltype != 0 {
					// Truncate distance tax to 2 decimal places (round toward zero)
					d := p.Mul(disttax)
					d = ctxTowardZero.Quantize(d, quantumHundredth)
					sumD = sumD.Add(d)
					t = t.Add(d)
				}
			}
			sumT = sumT.Add(t)
			outStr = t.String()
		} else {
			outStr = "0.77"
		}

		_, _ = writer.WriteString(outStr + "\r\n")
	}

	_ = writer.Flush()
	tFinis := time.Since(tStart)
	totalUs := float64(tFinis.Microseconds())

	fmt.Println("-- telco Go benchmark result --")
	fmt.Printf("   %d numbers read from '%s'\n", numbers, filein)
	fmt.Printf("   Time per number: %7.3fus [%d]\n", totalUs/float64(numbers), numbers)
	fmt.Println("--")
	fmt.Printf("   sumT = %s\n", sumT)
	fmt.Printf("   sumB = %s\n", sumB)
	fmt.Printf("   sumD = %s\n", sumD)
}
