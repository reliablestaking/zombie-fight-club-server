package metadata

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	Record struct {
		Wins  int
		Loses int
		Name  string
	}
)

func TestFightCalculations(t *testing.T) {
	zcMeta, err := LoadZombieChainsMeta(".")
	if err != nil {
		t.Fatalf("Error loaindg zc meta %v", err)
	}
	logrus.Infof("Found %d zc meta", len(zcMeta))

	zhMeta, err := LoadZombieHunterMeta(".")
	if err != nil {
		t.Fatalf("Error loaindg zh meta %v", err)
	}
	logrus.Infof("Found %d zh meta", len(zhMeta))

	// load zc trait strenght
	zcTraitStrength, err := LoadZombieChainsFightStrength(".")
	if err != nil {
		t.Fatalf("Error loaindg zc strength %v", err)
	}
	zhTraitStrength, err := LoadZombieHunterFightStrength(".")
	if err != nil {
		t.Fatalf("Error loaindg zc strength %v", err)
	}

	zcMax := 0
	zcMin := 1000
	zcStrengthMap := make(map[string]int)

	totalZcStrength := 0
	// find strength for every zombie
	for i := 1; i <= 10000; i++ {
		zombieName := fmt.Sprintf("ZombieChains%05d", i)
		zombieMeta := zcMeta[zombieName]
		strength := determineZcStrenth(zombieMeta, *zcTraitStrength, 0)
		//logrus.Infof("Zombie %s has strength %d", zombieName, strength)
		totalZcStrength += strength

		zcStrengthMap[zombieName] = strength
		if strength > zcMax {
			zcMax = strength
		}
		if strength < zcMin {
			zcMin = strength
		}
	}

	// find strength for every zombie
	for i := 1; i <= 10000; i++ {
		hunterName := fmt.Sprintf("ZombieHunter%05d", i)
		hunterMeta := zhMeta[hunterName]
		strength := determineZhStrength(hunterMeta, *zhTraitStrength, 0)

		if strength > zcMax {
			zcMax = strength
		}
		if strength < zcMin {
			zcMin = strength
		}
	}

	logrus.Infof("Max Zc Strengh: %d Min Zc Strength: %d", zcMax, zcMin)
	keys := make([]string, 0, len(zcStrengthMap))
	for key := range zcStrengthMap {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return zcStrengthMap[keys[i]] < zcStrengthMap[keys[j]]
	})

	zcDiff := zcMax - zcMin

	f, err := os.Create("ZombieStrength.csv")
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	for _, key := range keys {
		element := zcStrengthMap[key]

		zcPercent := float64(element-zcMin) / float64(zcDiff) * 100
		zcRanking := int64(10 * zcPercent / 100)

		// if zcRanking == 0 {
		// 	zcRanking = 1
		// }
		//logrus.Infof("Zombie %s has a rating of %d", key, zcRanking)

		record := make([]string, 0)
		record = append(record, key)
		record = append(record, strconv.FormatInt(zcRanking, 10))

		if err := w.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}

	// keys := make([]string, 0, len(zcStrengthMap))
	// for key := range zcStrengthMap {
	// 	keys = append(keys, key)
	// }
	// sort.SliceStable(keys, func(i, j int) bool {
	// 	return zcStrengthMap[keys[i]] < zcStrengthMap[keys[j]]
	// })
	// bucketCounter := 1
	// for _, key := range keys {
	// 	//logrus.Infof("Zombie %s has a strength of %d", key, zcStrengthMap[key])
	// 	if bucketCounter == 1000 {
	// 		logrus.Infof("Zombie %s has a strength of %d", key, zcStrengthMap[key])
	// 		bucketCounter = 0
	// 	}
	// 	bucketCounter++
	// }

	// totalZhStrength := 0
	// // find strength for every hunter
	// for i := 1; i <= 10000; i++ {
	// 	hunterName := fmt.Sprintf("ZombieHunter%05d", i)
	// 	hunterMeta := zhMeta[hunterName]
	// 	strength := determineZhStrength(hunterMeta, *zhTraitStrength, 0)
	// 	//logrus.Infof("Hunter %s has strength %d", hunterName, strength)
	// 	totalZhStrength += strength
	// }

	// zhKeys := make([]string, 0, len(zhStrengthMap))
	// for zhKey := range zhStrengthMap {
	// 	zhKeys = append(zhKeys, zhKey)
	// }
	// sort.SliceStable(zhKeys, func(i, j int) bool {
	// 	return zhStrengthMap[zhKeys[i]] < zhStrengthMap[zhKeys[j]]
	// })
	// bucketCounter = 1
	// for _, key := range zhKeys {
	// 	//logrus.Infof("Zombie %s has a strength of %d", key, zcStrengthMap[key])
	// 	if bucketCounter == 1000 {
	// 		logrus.Infof("Hunter %s has a strength of %d", key, zhStrengthMap[key])
	// 		bucketCounter = 0
	// 	}
	// 	bucketCounter++
	// }

	buildHunterStrength(zhMeta, zhTraitStrength, zcMax, zcMin)

	//logrus.Infof("Total zc: %d zh: %d", totalZcStrength, totalZhStrength)
	//logrus.Infof("Average zc: %d zh: %d", totalZcStrength/10000, totalZhStrength/10000)

}

func buildHunterStrength(zhMeta map[string]ZombieHunter, zhTraitStrength *ZombieHunterTraitStrength, zhMax int, zhMin int) {
	zhStrengthMap := make(map[string]int)

	totalZhStrength := 0
	// find strength for every zombie
	for i := 1; i <= 10000; i++ {
		hunterName := fmt.Sprintf("ZombieHunter%05d", i)
		hunterMeta := zhMeta[hunterName]
		strength := determineZhStrength(hunterMeta, *zhTraitStrength, 0)
		//logrus.Infof("Zombie %s has strength %d", zombieName, strength)
		totalZhStrength += strength

		zhStrengthMap[hunterName] = strength
	}

	logrus.Infof("Max Zh Strengh: %d Min Zh Strength: %d", zhMax, zhMin)
	keys := make([]string, 0, len(zhStrengthMap))
	for key := range zhStrengthMap {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return zhStrengthMap[keys[i]] < zhStrengthMap[keys[j]]
	})

	zhDiff := zhMax - zhMin

	zhf, _ := os.Create("HunterStrength.csv")
	defer zhf.Close()
	zhw := csv.NewWriter(zhf)
	defer zhw.Flush()

	for _, key := range keys {
		element := zhStrengthMap[key]

		zhPercent := float64(element-zhMin) / float64(zhDiff) * 100
		zhRanking := int64(10 * zhPercent / 100)

		// if zhRanking == 0 {
		// 	zhRanking = 1
		// }
		//logrus.Infof("Hunter %s has a rating of %d", key, zhRanking)

		record := make([]string, 0)
		record = append(record, key)
		record = append(record, strconv.FormatInt(zhRanking, 10))

		if err := zhw.Write(record); err != nil {
			log.Fatalln("error writing record to file", err)
		}
	}
}

func TestSimulateFights(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	zcMeta, err := LoadZombieChainsMeta(".")
	if err != nil {
		t.Fatalf("Error loaindg zc meta %v", err)
	}
	logrus.Infof("Found %d zc meta", len(zcMeta))

	zhMeta, err := LoadZombieHunterMeta(".")
	if err != nil {
		t.Fatalf("Error loaindg zh meta %v", err)
	}
	logrus.Infof("Found %d zh meta", len(zhMeta))

	// load zc trait strenght
	zcTraitStrength, err := LoadZombieChainsFightStrength(".")
	if err != nil {
		t.Fatalf("Error loaindg zc strength %v", err)
	}
	zhTraitStrength, err := LoadZombieHunterFightStrength(".")
	if err != nil {
		t.Fatalf("Error loaindg zc strength %v", err)
	}

	zombieRecords := make(map[string]*Record)
	hunterRecords := make(map[string]*Record)

	numKnockouts := 0
	totalZombieWins := 0
	totalHunterWins := 0
	totalDifference := 0
	for i := 1; i <= 100000; i++ {
		// pick random
		randomZombieNumber := rand.Intn(10000)
		randomHunterNumber := rand.Intn(10000)
		zombieName := fmt.Sprintf("ZombieChains%05d", randomZombieNumber)
		hunterName := fmt.Sprintf("ZombieHunter%05d", randomHunterNumber)

		zombieWin, hunterWin, difference := fightZombieAndHunter(zombieName, hunterName, zcMeta, zhMeta, *zcTraitStrength, *zhTraitStrength, 120)
		logrus.Info(difference)
		totalDifference += difference
		if zombieWin {
			totalZombieWins++
			if val, ok := zombieRecords[zombieName]; ok {
				val.Wins++
			} else {
				zombieRecords[zombieName] = &Record{Wins: 1, Name: zombieName}
			}
			if val, ok := hunterRecords[hunterName]; ok {
				val.Loses++
			} else {
				hunterRecords[hunterName] = &Record{Loses: 1, Name: hunterName}
			}
		} else if hunterWin {
			totalHunterWins++
			if val, ok := hunterRecords[hunterName]; ok {
				val.Wins++
			} else {
				hunterRecords[hunterName] = &Record{Wins: 1, Name: hunterName}
			}
			if val, ok := zombieRecords[zombieName]; ok {
				val.Loses++
			} else {
				zombieRecords[zombieName] = &Record{Loses: 1, Name: zombieName}
			}
		}

		_, _, ko := DetermineLifeBar(difference)
		if ko {
			numKnockouts++
		}
	}

	logrus.Infof("Total zombie wins %d, total hunter wins %d, average difference %d", totalZombieWins, totalHunterWins, totalDifference/100000)

	bestZombieRecord := Record{}
	for _, element := range zombieRecords {
		if element.Wins > bestZombieRecord.Wins {
			bestZombieRecord = *element
		}
	}
	logrus.Infof("Most wins %d-%d for zombie %s", bestZombieRecord.Wins, bestZombieRecord.Loses, bestZombieRecord.Name)

	bestHunterRecord := Record{}
	for _, element := range hunterRecords {
		if element.Wins > bestHunterRecord.Wins {
			bestHunterRecord = *element
		}
	}
	logrus.Infof("Most wins %d-%d for hunter %s", bestHunterRecord.Wins, bestHunterRecord.Loses, bestHunterRecord.Name)

	logrus.Infof("Knockouts: %d", numKnockouts)
}
