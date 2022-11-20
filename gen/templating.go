package gen

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/dofusdude/api/utils"
)

func DeleteDamageFormatter(input string) string {
	input, regex := PrepareAndCreateRangeRegex(input, false)
	if strings.Contains(input, "+#1{~1~2 to } level #2") {
		return "level"
	}

	input = strings.ReplaceAll(input, "#1{~1~2 -}#2", "#1{~1~2 - }#2") // bug from ankama
	input = regex.ReplaceAllString(input, "")

	input = strings.ReplaceAll(input, "{~1~2 to }", "")
	input = utils.DeleteReplacer(input)
	input = strings.ReplaceAll(input, "  ", " ")

	input = strings.TrimSpace(input)
	return input
}

func SingularPluralFormatter(input string, amount int, lang string) string {
	str := strings.ReplaceAll(input, "{~s}", "") // avoid only s without what to append
	str = strings.ReplaceAll(str, "{~p}", "")    // same

	// delete unknown z
	unknownZRegex := regexp.MustCompile("{~z[^}]*}")
	str = unknownZRegex.ReplaceAllString(str, "")

	var indicator rune

	if amount > 1 {
		indicator = 'p'
	} else {
		indicator = 's'
	}

	indicators := []rune{'s', 'p'}
	var regexps []*regexp.Regexp
	for _, indicatorIt := range indicators {
		regex := fmt.Sprintf("{~%c([^}]*)}", indicatorIt) // capturing with everything inside ()
		regexExtract := regexp.MustCompile(regex)
		regexps = append(regexps, regexExtract)

		//	if lang == "es" || lang == "pt" {
		if indicatorIt != indicator {
			continue
		}
		extractedEntries := regexExtract.FindAllStringSubmatch(str, -1)
		for _, extracted := range extractedEntries {
			str = strings.ReplaceAll(str, extracted[0], extracted[1])
		}
	}

	for _, regexIt := range regexps {
		str = regexIt.ReplaceAllString(str, "")
	}

	return str
}

func ElementFromCode(code string) int {
	code = strings.ToLower(code)

	switch code {
	case "cs":
		return 501945 // "Strength"
	case "ci":
		return 501944 // "Intelligence"
	case "cv":
		return 501947 // "Vitality"
	case "ca":
		return 501941 // "Agility"
	case "cc":
		return 501942 // "Chance"
	case "cw":
		return 501946 // "Wisdom"
	case "pk":
		return 422874 // "Set-Bonus"
	case "pl":
		return 837224 // "Mindestens Stufe %1"
	case "cm":
		return 67248 // "Bewegungsp. (BP)"
	case "cp":
		return 67755 // "Aktionsp. (AP)"
	case "po":
		return 335357 // Anderes Gebiet als: %1
	case "pf":
		return 644231 // Nicht ausgerüstetes %1-Reittier
	//case "": // Ps=1
	//	return 644230 // Ausgerüstetes %1-Reittier
	case "pa":
		return 66566 // Gesinunngsstufe
	//case "":
	//	return 637203 // Kein ausgerüstetes %1-Reittier haben
	case "of":
		return 637212 // Ein ausgerüstetes %1-Reittier haben
	case "pz":
		return 66351 // Abonniert sein
	}

	return -1
}

func ConditionWithOperator(input string, operator string, langs *map[string]LangDict, out *MappedMultilangCondition, data *JSONGameData) bool {
	partSplit := strings.Split(input, operator)
	rawElement := ElementFromCode(partSplit[0])
	if rawElement == -1 {
		return false
	}
	out.Element = strings.ToLower(partSplit[0])
	out.Value, _ = strconv.Atoi(partSplit[1])
	for _, lang := range utils.Languages {
		langStr := (*langs)[lang].Texts[rawElement]

		if lang == "en" {
			if langStr == "()" {
				return false
			}

			keySanitized := utils.DeleteReplacer(langStr)

			key, foundKey := utils.PersistedElements.Entries.GetKey(keySanitized)
			if foundKey {
				out.ElementId = key.(int)
			} else {
				utils.PersistedElements.Entries.Put(utils.PersistedElements.NextId, keySanitized)
				utils.PersistedElements.NextId++
			}
		}

		switch rawElement {
		case 837224: // %1 replace
			intVal, _ := strconv.Atoi(partSplit[1])
			langStr = strings.ReplaceAll(langStr, "%1", fmt.Sprint(intVal+1))
			break
		case 335357: // anderes gebiet als %1
			langStr = strings.ReplaceAll(langStr, "%1", (*langs)[lang].Texts[data.Areas[out.Value].NameId])
			break
		case 637212: // reittier %1
		case 644231:
			langStr = strings.ReplaceAll(langStr, "%1", (*langs)[lang].Texts[data.Mounts[out.Value].NameId])
			break
		}

		out.Templated[lang] = langStr
	}
	out.Operator = operator
	return true
}

// returns info about min max with in. -1 "only_min", -2 "no_min_max"
func NumSpellFormatter(input string, lang string, gameData *JSONGameData, langs *map[string]LangDict, diceNum *int, diceSide *int, value *int, effectNameId int, numIsSpell bool, useDice bool) (string, int) {
	diceNumIsSpellId := *diceNum > 12000 || numIsSpell
	diceSideIsSpellId := *diceSide > 12000
	valueIsSpellId := *value > 12000

	onlyNoMinMax := 0

	// when + xp
	if !useDice && *diceNum == 0 && *value == 0 && *diceSide != 0 {
		*value = *diceSide
		*diceSide = 0
	}

	delValue := false

	input, concatRegex := PrepareAndCreateRangeRegex(input, true)
	numSigned, sideSigned := ParseSigness(input)
	concatEntries := concatRegex.FindAllStringSubmatch(input, -1)

	if *diceSide == 0 { // only replace #1 with dice_num
		for _, extracted := range concatEntries {
			input = strings.ReplaceAll(input, extracted[0], "")
		}
	} else {
		for _, extracted := range concatEntries {
			input = strings.ReplaceAll(input, extracted[0], fmt.Sprintf(" %s", extracted[1]))
		}
	}

	num1Regex := regexp.MustCompile("([-,+]?)#1")
	num1Entries := num1Regex.FindAllStringSubmatch(input, -1)
	for _, extracted := range num1Entries {
		var diceNumStr string
		if diceNumIsSpellId {
			diceNumStr = (*langs)[lang].Texts[gameData.Spells[*diceNum].NameId]
		} else {
			diceNumStr = fmt.Sprint(*diceNum)
		}
		input = strings.ReplaceAll(input, extracted[0], fmt.Sprintf("%s%s", extracted[1], diceNumStr))
	}

	if *diceSide == 0 {
		input = strings.ReplaceAll(input, "#2", "")
	} else {
		var diceSideStr string
		if diceSideIsSpellId {
			diceSideStr = (*langs)[lang].Texts[gameData.Spells[*diceSide].NameId]
			//del_dice_side = true
		} else {
			diceSideStr = fmt.Sprint(*diceSide)
		}
		input = strings.ReplaceAll(input, "#2", diceSideStr)
	}

	var valueStr string
	if valueIsSpellId {
		valueStr = (*langs)[lang].Texts[gameData.Spells[*value].NameId]
		delValue = true
	} else {
		valueStr = fmt.Sprint(*value)
	}
	if effectNameId == 427090 { // go to <npc> for more info
		return "", -2
	}
	input = strings.ReplaceAll(input, "#3", valueStr)

	if delValue {
		*diceNum = utils.Min(*diceNum, *diceSide)
	}

	if !useDice {
		// avoid min = 0, max > x
		if *diceNum == 0 && *diceSide != 0 {
			*diceNum = *diceSide
			*diceSide = 0
		}
	}

	if *diceNum == 0 && *diceSide == 0 {
		onlyNoMinMax = -2
	}

	if *diceNum != 0 && *diceSide == 0 {
		onlyNoMinMax = -1
	}

	input = strings.TrimSpace(input)

	if numSigned {
		*diceNum *= -1
	}

	if sideSigned {
		*diceSide *= -1
	}

	return input, onlyNoMinMax
}

func PrepareTextForRegex(input string) string {
	input = strings.ReplaceAll(input, "{~1~2 -}", "{~1~2 - }")
	input = strings.ReplaceAll(input, "{~1~2 to}level", "{~1~2 to } level") // {~1~2 to}level
	input = strings.ReplaceAll(input, "{~1~2 to}", "{~1~2 to }")
	input = strings.ReplaceAll(input, "\"\"", "")
	input = strings.TrimPrefix(input, ":")
	input = strings.TrimSuffix(input, ":")
	input = strings.TrimPrefix(input, "+")
	return input
}

func PrepareAndCreateRangeRegex(input string, extract bool) (string, *regexp.Regexp) {
	var regexStr string
	combiningWords := "(und|et|and|bis|to|a|à|-|auf)"
	if extract {
		regexStr = fmt.Sprintf("{~1~2 (%s [-,+]?)}", combiningWords)
	} else {
		regexStr = fmt.Sprintf("[-,+]?#1{~1~2 %s [-,+]?}#2", combiningWords)
	}

	concatRegex := regexp.MustCompile(regexStr)

	return PrepareTextForRegex(input), concatRegex
}

func ParseSigness(input string) (bool, bool) {
	numSigness := false
	sideSigness := false

	regexNum := regexp.MustCompile("(([+,-])?#1)")
	entriesNum := regexNum.FindAllStringSubmatch(input, -1)
	for _, extracted := range entriesNum {
		for _, entry := range extracted {
			if entry == "-" {
				numSigness = true
			}
		}
	}

	regexSide := regexp.MustCompile("([+,-])?}?#2")
	entriesSide := regexSide.FindAllStringSubmatch(input, -1)
	for _, extracted := range entriesSide {
		for _, entry := range extracted {
			if entry == "-" {
				sideSigness = true
			}
		}
	}

	return numSigness, sideSigness
}
