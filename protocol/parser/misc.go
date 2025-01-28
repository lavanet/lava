package parser

import (
	"fmt"
	"strings"

	"github.com/lavanet/lava/v4/utils"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	MinLettersForPattern = 10
	wildcard             = "%*s"
)

func CapStringLen(inp string) string {
	if len(inp) > 250 {
		return inp[:150] + "...Truncated..." + inp[len(inp)-100:]
	}
	return inp
}

func isDigitOrLetter(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func ParseNumberFromPattern(pattern string, input string) (fitsPattern bool, parsedHeight int64) {
	if len(pattern) > 0 {
		percentIndex := strings.Index(pattern, "%")
		if percentIndex == -1 {
			return false, 0
		}
		if pattern[:percentIndex] != input[:percentIndex] {
			return false, 0
		}
	} else {
		// too short to be a pattern
		return false, 0
	}
	dmp := diffmatchpatch.New()
	nextOneFormat := ""
	diffs := dmp.DiffMain(pattern, CapStringLen(input), false)
	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffEqual {
			nextOneFormat = ""
			continue
		}
		if diff.Type == diffmatchpatch.DiffDelete {
			if (len(diff.Text) == 2 || len(diff.Text) == 3) && diff.Text[0] == '%' {
				nextOneFormat = diff.Text
			} else {
				// something is different other than %s
				return false, 0
			}
		} else if diff.Type == diffmatchpatch.DiffInsert {
			if nextOneFormat == "%*s" {
				// skip these
				nextOneFormat = ""
			} else if nextOneFormat != "" {
				var err error
				_, err = fmt.Sscanf(diff.Text, nextOneFormat, &parsedHeight)
				if err != nil {
					utils.LavaFormatError("invalid height in pattern", err, utils.Attribute{Key: "diff.Text", Value: diff.Text}, utils.Attribute{Key: "pattern", Value: pattern}, utils.Attribute{Key: "input", Value: input})
					return false, 0
				}
				nextOneFormat = ""
			} else {
				// something is different other than %s
				return false, 0
			}
		}
	}
	if parsedHeight == 0 {
		return false, 0
	}
	return true, parsedHeight
}

func LongestCommonSubsequenceWithFormat(s1, s2 string) (string, int) {
	lettersDiff := 0
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(s1, s2, false)
	stringsArr := []string{}

	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffEqual {
			text := diff.Text
			// trim digit or number elements right after a wildcard
			if len(stringsArr) > 0 {
				last_element := stringsArr[len(stringsArr)-1]
				if last_element == wildcard {
					trim := 0
					for _, element := range text {
						if !isDigitOrLetter(element) {
							break
						}
						trim++
					}
					if trim > 0 {
						text = text[trim:]
					}
				}
			}
			stringsArr = append(stringsArr, text)
			lettersDiff += len(text)
		} else {
			if len(stringsArr) > 0 {
				last_element := stringsArr[len(stringsArr)-1]
				// ignore duplicate wildcards
				if last_element == wildcard {
					continue
				}
				// trim digit or number elements right before the wildcard
				trim := 0
				for i := len(last_element) - 1; i >= 0; i-- {
					if !isDigitOrLetter(rune(last_element[i])) {
						break
					}
					trim++
				}
				if trim > 0 {
					stringsArr[len(stringsArr)-1] = last_element[:len(last_element)-trim]
					lettersDiff -= trim
				}
			}
			stringsArr = append(stringsArr, "%*s")
		}
	}
	return strings.Join(stringsArr, ""), lettersDiff
}
