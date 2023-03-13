package metadata

import (
	"encoding/csv"
	"io"
	"math/rand"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

type (
	TraitStrength struct {
		Name     string
		Strength int
		Weakness string
	}

	ZombieChain struct {
		Background string `json:"background"`
		Hat        string `json:"hat"`
		Eyes       string `json:"eyes"`
		Nose       string `json:"nose"`
		Skin       string `json:"skin"`
		Mouth      string `json:"mouth"`
		Chain      string `json:"chain"`
		Weapon     string `json:"weapon"`
		Clothing   string `json:"clothing"`
		Earrings   string `json:"earrings"`
	}

	ZombieChainTraitStrength struct {
		Background map[string]TraitStrength
		Hat        map[string]TraitStrength
		Eyes       map[string]TraitStrength
		Nose       map[string]TraitStrength
		Skin       map[string]TraitStrength
		Mouth      map[string]TraitStrength
		Chain      map[string]TraitStrength
		Weapon     map[string]TraitStrength
		Clothing   map[string]TraitStrength
		Earrings   map[string]TraitStrength
	}

	ZombieHunter struct {
		Background  string `json:"background"`
		Gender      string `json:"gender"`
		Hat         string `json:"hat"`
		Eyes        string `json:"eyes"`
		Skin        string `json:"skin"`
		Mouth       string `json:"mouth"`
		Chain       string `json:"chain"`
		LeftWeapon  string `json:"leftWeapon"`
		RightWeapon string `json:"rightWeapon"`
		Clothing    string `json:"clothing"`
		Earrings    string `json:"earrings"`
		Loot        string `json:"loot"`
	}

	ZombieHunterTraitStrength struct {
		Background  map[string]TraitStrength
		Gender      map[string]TraitStrength
		Hat         map[string]TraitStrength
		Eyes        map[string]TraitStrength
		Skin        map[string]TraitStrength
		Mouth       map[string]TraitStrength
		Chain       map[string]TraitStrength
		LeftWeapon  map[string]TraitStrength
		RightWeapon map[string]TraitStrength
		Clothing    map[string]TraitStrength
		Earrings    map[string]TraitStrength
		Loot        map[string]TraitStrength
	}
)

func LoadZombieChainsMeta(path string) (map[string]ZombieChain, error) {
	// open file
	f, err := os.Open(path + "/zombie-meta-final.csv")
	if err != nil {
		return nil, err
	}

	// remember to close the file at the end of the program
	defer f.Close()

	// Parse the file
	r := csv.NewReader(f)

	zombies := make(map[string]ZombieChain)

	// skip first row
	_, err = r.Read()
	if err == io.EOF {
		return nil, err
	}
	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithError(err).Fatal("Error reading row")
		}

		name := record[0]
		zombie := ZombieChain{
			Background: record[4],
			Weapon:     record[5],
			Skin:       record[6],
			Clothing:   record[7],
			Chain:      record[8],
			Mouth:      record[9],
			Nose:       record[10],
			Hat:        record[11],
			Eyes:       record[12],
			Earrings:   record[13],
		}

		zombies[name] = zombie
	}

	return zombies, nil
}

func LoadZombieChainsFightStrength(path string) (*ZombieChainTraitStrength, error) {
	// open file
	f, err := os.Open(path + "/zc_trait_rarity.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)

	zombiesStrength := ZombieChainTraitStrength{
		Background: make(map[string]TraitStrength),
		Hat:        make(map[string]TraitStrength),
		Eyes:       make(map[string]TraitStrength),
		Nose:       make(map[string]TraitStrength),
		Skin:       make(map[string]TraitStrength),
		Mouth:      make(map[string]TraitStrength),
		Chain:      make(map[string]TraitStrength),
		Weapon:     make(map[string]TraitStrength),
		Clothing:   make(map[string]TraitStrength),
		Earrings:   make(map[string]TraitStrength),
	}

	// skip first row
	_, err = r.Read()
	if err == io.EOF {
		return nil, err
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithError(err).Fatal("Error reading row")
		}

		strengthString := record[3]
		strength, err := strconv.Atoi(strengthString)
		if err != nil {
			logrus.WithError(err).Fatal("Error converting strength to int")

		}
		traitStrength := TraitStrength{
			Name:     record[1],
			Strength: strength,
		}

		traitType := record[0]
		if traitType == "background" {
			zombiesStrength.Background[record[1]] = traitStrength
		} else if traitType == "hat" {
			zombiesStrength.Hat[record[1]] = traitStrength
		} else if traitType == "eyes" {
			zombiesStrength.Eyes[record[1]] = traitStrength
		} else if traitType == "nose" {
			zombiesStrength.Nose[record[1]] = traitStrength
		} else if traitType == "skin" {
			zombiesStrength.Skin[record[1]] = traitStrength
		} else if traitType == "mouth" {
			zombiesStrength.Mouth[record[1]] = traitStrength
		} else if traitType == "chain" {
			zombiesStrength.Chain[record[1]] = traitStrength
		} else if traitType == "weapon" {
			zombiesStrength.Weapon[record[1]] = traitStrength
		} else if traitType == "clothing" {
			zombiesStrength.Clothing[record[1]] = traitStrength
		} else if traitType == "earrings" {
			zombiesStrength.Earrings[record[1]] = traitStrength
		}
	}

	return &zombiesStrength, nil
}

func LoadZombieHunterMeta(path string) (map[string]ZombieHunter, error) {
	// open file
	f, err := os.Open(path + "/hunter-meta-final.csv")
	if err != nil {
		return nil, err
	}

	// remember to close the file at the end of the program
	defer f.Close()

	// Parse the file
	r := csv.NewReader(f)

	zombies := make(map[string]ZombieHunter)

	// skip first row
	_, err = r.Read()
	if err == io.EOF {
		return nil, err
	}
	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithError(err).Fatal("Error reading row")
		}

		name := record[0]
		zombie := ZombieHunter{
			Background:  record[6],
			Gender:      record[7],
			LeftWeapon:  record[10],
			RightWeapon: record[11],
			Skin:        record[8],
			Clothing:    record[9],
			Chain:       record[12],
			Mouth:       record[13],
			Hat:         record[14],
			Eyes:        record[15],
			Earrings:    record[16],
			Loot:        record[17],
		}

		zombies[name] = zombie
	}

	return zombies, nil
}

func LoadZombieHunterFightStrength(path string) (*ZombieHunterTraitStrength, error) {
	// open file
	f, err := os.Open(path + "/zh_trait_rarity.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)

	huntersStrength := ZombieHunterTraitStrength{
		Background:  make(map[string]TraitStrength),
		Hat:         make(map[string]TraitStrength),
		Eyes:        make(map[string]TraitStrength),
		Skin:        make(map[string]TraitStrength),
		Mouth:       make(map[string]TraitStrength),
		Chain:       make(map[string]TraitStrength),
		RightWeapon: make(map[string]TraitStrength),
		LeftWeapon:  make(map[string]TraitStrength),
		Clothing:    make(map[string]TraitStrength),
		Earrings:    make(map[string]TraitStrength),
		Loot:        make(map[string]TraitStrength),
	}

	// skip first row
	_, err = r.Read()
	if err == io.EOF {
		return nil, err
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithError(err).Fatal("Error reading row")
		}

		strengthString := record[3]
		strength, err := strconv.Atoi(strengthString)
		if err != nil {
			logrus.WithError(err).Fatal("Error converting strength to int")

		}
		traitStrength := TraitStrength{
			Name:     record[1],
			Strength: strength,
		}

		traitType := record[0]
		if traitType == "background" {
			huntersStrength.Background[record[1]] = traitStrength
		} else if traitType == "hat" {
			huntersStrength.Hat[record[1]] = traitStrength
		} else if traitType == "eyes" {
			huntersStrength.Eyes[record[1]] = traitStrength
		} else if traitType == "right-weapon" {
			huntersStrength.RightWeapon[record[1]] = traitStrength
		} else if traitType == "skin" {
			huntersStrength.Skin[record[1]] = traitStrength
		} else if traitType == "mouth" {
			huntersStrength.Mouth[record[1]] = traitStrength
		} else if traitType == "chain" {
			huntersStrength.Chain[record[1]] = traitStrength
		} else if traitType == "left-weapon" {
			huntersStrength.LeftWeapon[record[1]] = traitStrength
		} else if traitType == "clothing" {
			huntersStrength.Clothing[record[1]] = traitStrength
		} else if traitType == "earrings" {
			huntersStrength.Earrings[record[1]] = traitStrength
		} else if traitType == "loot" {
			huntersStrength.Loot[record[1]] = traitStrength
		}
	}

	return &huntersStrength, nil
}

//TODO: error is strength is 0
func determineZcStrenth(zc ZombieChain, strength ZombieChainTraitStrength, randomness int) int {
	background := strength.Background[zc.Background].Strength
	chain := strength.Chain[zc.Chain].Strength
	clothing := strength.Clothing[zc.Clothing].Strength
	earring := strength.Earrings[zc.Earrings].Strength
	eyes := strength.Eyes[zc.Eyes].Strength
	hat := strength.Hat[zc.Hat].Strength
	mouth := strength.Mouth[zc.Mouth].Strength
	nose := strength.Nose[zc.Nose].Strength
	skin := strength.Skin[zc.Skin].Strength
	weapon := strength.Weapon[zc.Weapon].Strength

	power := background + chain + clothing + earring + eyes + hat + mouth + nose + skin + weapon
	return alterRandomly(power, randomness)
}

func determineZhStrength(zc ZombieHunter, strength ZombieHunterTraitStrength, randomness int) int {
	background := strength.Background[zc.Background].Strength
	chain := strength.Chain[zc.Chain].Strength
	clothing := strength.Clothing[zc.Clothing].Strength
	earring := strength.Earrings[zc.Earrings].Strength
	eyes := strength.Eyes[zc.Eyes].Strength
	hat := strength.Hat[zc.Hat].Strength
	mouth := strength.Mouth[zc.Mouth].Strength
	leftWeapon := strength.LeftWeapon[zc.LeftWeapon].Strength
	skin := strength.Skin[zc.Skin].Strength
	rightWeapon := strength.RightWeapon[zc.RightWeapon].Strength
	loot := strength.Loot[zc.Loot].Strength

	//logrus.Infof("B: %d, C: %d, C: %d, E: %d, E: %d, H: %d, M: %d, LW: %d, S: %d, RW: %d, L: %d", background, chain, clothing, earring, eyes, hat, mouth, leftWeapon, skin, rightWeapon, loot)

	power := background + chain + clothing + earring + eyes + hat + mouth + leftWeapon + skin + rightWeapon + loot
	return alterRandomly(power, randomness)
}

func fightZombieAndHunter(zombieName string, hunterName string, zombieMeta map[string]ZombieChain, hunterMeta map[string]ZombieHunter, zcStrengthCalc ZombieChainTraitStrength, zhStrengthCalc ZombieHunterTraitStrength, randomness int) (bool, bool, int) {
	zcStrength := determineZcStrenth(zombieMeta[zombieName], zcStrengthCalc, randomness)
	zhStrength := determineZhStrength(hunterMeta[hunterName], zhStrengthCalc, randomness)
	logrus.Infof("Fighting %s with strength %d vs %s with strenth %d", zombieName, zcStrength, hunterName, zhStrength)

	if zcStrength >= zhStrength {
		return true, false, zcStrength - zhStrength
	} else {
		return false, true, zhStrength - zcStrength
	}
}

func FightZombieAndHunterReturnStrength(zombieName string, hunterName string, zombieMeta map[string]ZombieChain, hunterMeta map[string]ZombieHunter, zcStrengthCalc ZombieChainTraitStrength, zhStrengthCalc ZombieHunterTraitStrength, randomness int) (int, int) {
	zcStrength := determineZcStrenth(zombieMeta[zombieName], zcStrengthCalc, randomness)
	zhStrength := determineZhStrength(hunterMeta[hunterName], zhStrengthCalc, randomness)
	logrus.Infof("Fighting %s with strength %d vs %s with strenth %d", zombieName, zcStrength, hunterName, zhStrength)

	return zcStrength, zhStrength
}

func alterRandomly(value int, randomness int) int {
	// nothing to do if no randomness
	if randomness == 0 {
		return value
	}

	randValue := rand.Intn(randomness)
	randValue = randValue - (randomness / 2)
	updatedValue := value + randValue

	//logrus.Infof("Value was %d and is now %d", value, updatedValue)

	return updatedValue
}

func DetermineLifeBar(pointDifference int) (int, int, bool) {
	//some randonness
	if pointDifference > 140 {
		return 100, 0, true
	} else if pointDifference > 120 {
		return pickRandom(9, 7), pickRandom(2, 1), true
	} else if pointDifference > 100 {
		return pickRandom(9, 7), pickRandom(2, 1), false
	} else if pointDifference > 90 {
		return pickRandom(8, 6), pickRandom(3, 1), false
	} else if pointDifference > 80 {
		return pickRandom(8, 6), pickRandom(4, 2), false
	} else if pointDifference > 70 {
		return pickRandom(7, 4), pickRandom(3, 2), false
	} else if pointDifference > 55 {
		return pickRandom(7, 4), pickRandom(3, 2), false
	} else if pointDifference > 44 {
		return pickRandom(7, 4), pickRandom(3, 2), false
	} else if pointDifference > 25 {
		return pickRandom(6, 5), pickRandom(4, 3), false
	} else if pointDifference > 13 {
		return 50, 40, false
	} else {
		return 60, 50, false
	}
}

func pickRandom(max int, min int) int {
	randomNumber := rand.Intn(max-min) + min
	return randomNumber * 10
}
